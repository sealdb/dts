# 连接管理文档

## 概述

DTS 支持同时运行多个迁移任务，每个任务需要管理自己的数据库连接。为了优化资源使用和避免连接泄漏，系统实现了任务级别的连接池管理。

## 架构设计

### 1. 任务管理器 (TaskManager)

`TaskManager` 负责管理所有正在运行的迁移任务：

```go
type TaskManager struct {
    tasks map[string]*MigrationTask  // key: task ID
    mu    sync.RWMutex               // 保护并发访问
}
```

**功能**：
- `AddTask()` - 添加任务到管理器
- `GetTask()` - 获取任务
- `RemoveTask()` - 移除任务并关闭所有连接
- `ListTasks()` - 列出所有任务
- `CleanupCompletedTasks()` - 定期清理已完成的任务

### 2. 任务连接池 (MigrationTask.Connections)

每个 `MigrationTask` 维护自己的连接池：

```go
type MigrationTask struct {
    // ... 其他字段

    Connections map[string]interface{}  // key: connectionKey (host:port:dbname)
    mu          sync.RWMutex             // 保护并发访问
}
```

**连接键格式**：`host:port:dbname`

例如：
- `localhost:5432:source_db`
- `192.168.1.100:5432:target_db`

### 3. 连接管理方法

#### AddConnection
添加连接到任务连接池。

```go
task.AddConnection(connectionKey, db)
```

#### GetConnection
从任务连接池获取连接。

```go
conn, ok := task.GetConnection(connectionKey)
```

#### CloseAllConnections
关闭任务的所有连接并清空连接池。

```go
err := task.CloseAllConnections()
```

## 连接生命周期

### 1. 连接创建

连接在首次使用时创建，并缓存到任务连接池：

```go
// 在 repository/connection_pool.go 中
func GetOrCreateConnection(task *MigrationTask, dbConfig *DBConfig) (*sql.DB, error) {
    connectionKey := dbConfig.ConnectionKey()

    // 尝试获取已有连接
    if conn, ok := task.GetConnection(connectionKey); ok {
        if sqlDB, ok := conn.(*sql.DB); ok {
            if err := sqlDB.Ping(); err == nil {
                return sqlDB, nil  // 连接有效，复用
            }
        }
    }

    // 创建新连接
    db, err := sql.Open("pgx", dbConfig.DSN())
    // ... 验证连接 ...

    // 添加到任务连接池
    task.AddConnection(connectionKey, db)
    return db, nil
}
```

### 2. 连接复用

所有状态执行时都使用任务连接池：

```go
// 在 state/init.go, creating_tables.go 等中
sourceRepo, err := repository.NewSourceRepositoryFromTask(task)
// 使用连接池，不关闭连接
```

### 3. 连接清理

#### 自动清理
- 任务完成时（`StateCompleted`）
- 任务失败时（`StateFailed`）
- 任务取消时

```go
defer func() {
    s.taskManager.RemoveTask(id)  // 自动关闭所有连接
}()
```

#### 定期清理
后台 goroutine 每 5 分钟清理一次已完成的任务：

```go
go func() {
    for {
        time.Sleep(5 * time.Minute)
        migrationService.GetTaskManager().CleanupCompletedTasks()
    }
}()
```

## 使用示例

### 创建任务并启动

```go
// 1. 创建任务
task, err := service.CreateTask(&CreateTaskRequest{
    SourceDB:    sourceDBConfig,
    TargetDB:    targetDBConfig,
    Tables:      []string{"users", "orders"},
    TableSuffix: "_migrated",
})

// 2. 启动任务（自动管理连接）
err = service.StartTask(ctx, task.ID)
```

### 手动管理连接（如果需要）

```go
// 获取源库连接
sourceConn, err := repository.GetOrCreateSourceConnection(task)

// 获取目标库连接
targetConn, err := repository.GetOrCreateTargetConnection(task)

// 任务完成后，所有连接会自动关闭
```

## 连接池配置

每个连接的连接池参数：

```go
db.SetMaxOpenConns(10)  // 最大打开连接数
db.SetMaxIdleConns(5)   // 最大空闲连接数
```

## 注意事项

1. **不要手动关闭连接**
   - 状态执行时不要调用 `defer repo.Close()`
   - 连接由任务管理器统一管理

2. **连接复用**
   - 同一个任务的多个状态共享相同的数据库连接
   - 通过 `connectionKey` 确保连接唯一性

3. **连接验证**
   - `GetOrCreateConnection` 会验证连接有效性
   - 无效连接会自动重新创建

4. **并发安全**
   - 所有连接操作都有 `sync.RWMutex` 保护
   - 多个状态可以安全地访问同一个连接

5. **资源清理**
   - 任务完成后必须清理连接，避免泄漏
   - 使用 `defer` 确保清理执行

## 故障处理

### 连接断开
如果连接在使用中断开，系统会：
1. 检测到连接无效（`Ping()` 失败）
2. 自动重新创建连接
3. 更新任务连接池

### 任务失败
如果任务失败：
1. 任务状态更新为 `StateFailed`
2. `defer` 中自动调用 `CloseAllConnections()`
3. 从任务管理器中移除

### 服务重启
如果服务重启：
- 已完成的任务不会自动恢复连接
- 需要重新启动任务才会创建新连接

## 监控和调试

### 获取连接数量
```go
count := task.GetConnectionCount()
fmt.Printf("Task %s has %d connections\n", task.ID, count)
```

### 列出所有运行中的任务
```go
tasks := taskManager.ListTasks()
for _, task := range tasks {
    fmt.Printf("Task %s: %d connections\n", task.ID, task.GetConnectionCount())
}
```

## 性能优化

1. **连接复用**：同一任务的多个状态共享连接，减少连接开销
2. **连接池**：使用 `sql.DB` 的连接池，自动管理连接生命周期
3. **延迟清理**：已完成的任务延迟 5 分钟清理，避免频繁创建/销毁连接

