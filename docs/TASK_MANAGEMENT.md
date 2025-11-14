# 任务管理文档

## 概述

DTS 支持同时运行多个迁移任务，每个任务都有独立的状态机和连接池。任务管理器负责统一管理所有正在运行的任务。

## 任务管理器 (TaskManager)

### 功能特性

1. **任务注册**：启动任务时自动注册到管理器
2. **连接管理**：任务完成或失败时自动清理所有数据库连接
3. **并发安全**：使用 `sync.RWMutex` 保护并发访问
4. **自动清理**：定期清理已完成的任务

### API 使用

```go
// 获取任务管理器
taskManager := migrationService.GetTaskManager()

// 获取运行中的任务
task, ok := taskManager.GetTask(taskID)

// 列出所有任务
tasks := taskManager.ListTasks()

// 获取任务数量
count := taskManager.GetTaskCount()

// 手动移除任务（会关闭所有连接）
err := taskManager.RemoveTask(taskID)
```

## 任务生命周期

### 1. 创建任务

```go
task, err := service.CreateTask(&CreateTaskRequest{
    SourceDB:    sourceDBConfig,
    TargetDB:    targetDBConfig,
    Tables:      []string{"users"},
    TableSuffix: "_migrated",
})
```

### 2. 启动任务

```go
err := service.StartTask(ctx, task.ID)
```

**启动流程**：
1. 检查任务状态（不能是终止状态）
2. 检查任务是否已在运行
3. 初始化连接池
4. 添加到任务管理器
5. 启动状态机（goroutine）

### 3. 任务执行

状态机自动执行各个状态：
- Init → CreatingTables → MigratingData → SyncingWAL →
- StoppingWrites → Validating → Finalizing → Completed

### 4. 任务完成

**自动清理**：
- 任务完成或失败时，`defer` 自动调用 `RemoveTask()`
- `RemoveTask()` 会关闭任务的所有数据库连接
- 从任务管理器中移除任务

## 连接管理

### 连接键格式

连接键由 `host:port:dbname` 组成：

```go
connectionKey := fmt.Sprintf("%s:%d:%s", host, port, dbname)
// 例如: "localhost:5432:source_db"
```

### 连接复用

同一任务的多个状态共享相同的数据库连接：

```go
// 在 InitState 中创建连接
sourceRepo, _ := repository.NewSourceRepositoryFromTask(task)

// 在 CreatingTablesState 中复用同一个连接
sourceRepo, _ := repository.NewSourceRepositoryFromTask(task)
```

### 连接清理

任务完成时自动清理：

```go
// 在 StartTask 的 goroutine 中
defer func() {
    s.taskManager.RemoveTask(id)  // 自动关闭所有连接
}()
```

## 并发控制

### 防止重复启动

```go
if _, exists := s.taskManager.GetTask(id); exists {
    return fmt.Errorf("task %s is already running", id)
}
```

### 并发安全

- `TaskManager` 使用 `sync.RWMutex` 保护
- `MigrationTask.Connections` 使用 `sync.RWMutex` 保护
- 多个状态可以安全地并发访问连接

## 定期清理

后台 goroutine 每 5 分钟清理一次已完成的任务：

```go
go func() {
    for {
        time.Sleep(5 * time.Minute)
        errors := migrationService.GetTaskManager().CleanupCompletedTasks()
        if len(errors) > 0 {
            log.Printf("Warning: errors during cleanup: %v", errors)
        }
    }
}()
```

## 监控和调试

### 查看运行中的任务

```go
tasks := taskManager.ListTasks()
for _, task := range tasks {
    fmt.Printf("Task %s: State=%s, Connections=%d\n",
        task.ID, task.State, task.GetConnectionCount())
}
```

### 查看任务连接

```go
task, _ := taskManager.GetTask(taskID)
if task != nil {
    count := task.GetConnectionCount()
    fmt.Printf("Task has %d connections\n", count)
}
```

## 最佳实践

1. **不要手动管理连接**
   - 让任务管理器自动管理
   - 状态执行时不要调用 `defer repo.Close()`

2. **任务完成后及时清理**
   - 系统会自动清理，但可以手动调用 `RemoveTask()`

3. **监控任务数量**
   - 定期检查 `GetTaskCount()` 避免资源耗尽

4. **错误处理**
   - 任务失败时连接会自动清理
   - 检查清理错误日志

## 示例

### 完整流程

```go
// 1. 创建任务
task, err := service.CreateTask(&CreateTaskRequest{
    SourceDB:    model.DBConfig{Host: "localhost", Port: 5432, ...},
    TargetDB:    model.DBConfig{Host: "localhost", Port: 5433, ...},
    Tables:      []string{"users", "orders"},
    TableSuffix: "_migrated",
})

// 2. 启动任务
err = service.StartTask(ctx, task.ID)

// 3. 监控任务（可选）
go func() {
    for {
        task, ok := service.GetTaskManager().GetTask(task.ID)
        if !ok {
            break  // 任务已完成或移除
        }
        fmt.Printf("Task %s: %s (%d%%)\n",
            task.ID, task.State, task.Progress)
        time.Sleep(5 * time.Second)
    }
}()

// 4. 等待任务完成
// 任务完成后会自动清理连接
```





