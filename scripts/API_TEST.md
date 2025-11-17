# DTS API 测试指南

本文档提供了所有 DTS API 的 curl 测试命令示例。

## 前提条件

1. 确保 DTS 服务器正在运行（默认端口：8080）
2. 确保已安装 `curl` 和 `jq`（可选，用于格式化 JSON 输出）

## API 基础路径

```
http://localhost:8080/dts/api/tasks
```

## API 列表

### 1. 创建任务 (POST /dts/api/tasks)

创建并自动启动数据同步任务。

**请求示例：**

```bash
curl -X POST http://localhost:8080/dts/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "task-001",
    "source": {
      "domin": "localhost",
      "port": "5432",
      "username": "postgres",
      "password": "postgres",
      "database": "source_db"
    },
    "dest": {
      "domin": "localhost",
      "port": "5433",
      "username": "postgres",
      "password": "postgres",
      "database": "target_db"
    },
    "tables": ["users", "orders"]
  }'
```

**不指定表（同步所有表）：**

```bash
curl -X POST http://localhost:8080/dts/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "task-002",
    "source": {
      "domin": "localhost",
      "port": "5432",
      "username": "postgres",
      "password": "postgres",
      "database": "source_db"
    },
    "dest": {
      "domin": "localhost",
      "port": "5433",
      "username": "postgres",
      "password": "postgres",
      "database": "target_db"
    }
  }'
```

**响应示例：**

```json
{
  "state": "OK",
  "message": "Task created and started successfully"
}
```

---

### 2. 查询任务状态 (GET /dts/api/tasks/:task_id/status)

查询同步任务的状态。

**请求示例：**

```bash
curl -X GET http://localhost:8080/dts/api/tasks/task-001/status
```

**响应示例：**

```json
{
  "state": "OK",
  "message": "",
  "stage": "syncing",
  "duration": -1,
  "delay": -1
}
```

**状态说明：**
- `stage`: `none`, `syncing`, `waiting`, `switching`, `finished`
- `duration`: 从切流开始到完成的时间（毫秒），-1 表示无意义
- `delay`: 同步延迟（毫秒），-1 表示无意义

---

### 3. 启动任务 (POST /dts/api/tasks/:task_id/start)

启动已创建的任务。

**请求示例：**

```bash
curl -X POST http://localhost:8080/dts/api/tasks/task-001/start
```

**响应示例：**

```json
{
  "state": "OK",
  "message": "Task started successfully"
}
```

---

### 4. 停止任务 (POST /dts/api/tasks/:task_id/stop)

停止任务运行（任务保留，不会被删除）。

**请求示例：**

```bash
curl -X POST http://localhost:8080/dts/api/tasks/task-001/stop
```

**响应示例：**

```json
{
  "state": "OK",
  "message": "Task stopped successfully"
}
```

---

### 5. 暂停任务 (POST /dts/api/tasks/:task_id/pause)

暂停任务执行。

**请求示例：**

```bash
curl -X POST http://localhost:8080/dts/api/tasks/task-001/pause
```

**响应示例：**

```json
{
  "state": "OK",
  "message": "Task paused successfully"
}
```

---

### 6. 恢复任务 (POST /dts/api/tasks/:task_id/resume)

恢复暂停的任务。

**请求示例：**

```bash
curl -X POST http://localhost:8080/dts/api/tasks/task-001/resume
```

**响应示例：**

```json
{
  "state": "OK",
  "message": "Task resumed successfully"
}
```

---

### 7. 切流 (POST /dts/api/tasks/:task_id/switch)

执行切流操作（停止源库写入，验证数据，恢复写入）。

**请求示例：**

```bash
curl -X POST http://localhost:8080/dts/api/tasks/task-001/switch
```

**响应示例：**

```json
{
  "state": "OK",
  "message": "Switchover triggered successfully"
}
```

---

### 8. 删除任务 (DELETE /dts/api/tasks/:task_id)

删除任务（会先取消任务，然后删除）。

**请求示例：**

```bash
curl -X DELETE http://localhost:8080/dts/api/tasks/task-001
```

**响应示例：**

```json
{
  "state": "OK",
  "message": "Task deleted successfully"
}
```

---

## 完整测试流程示例

```bash
# 1. 创建任务
TASK_ID="test-task-$(date +%s)"
curl -X POST http://localhost:8080/dts/api/tasks \
  -H "Content-Type: application/json" \
  -d "{
    \"task_id\": \"${TASK_ID}\",
    \"source\": {
      \"domin\": \"localhost\",
      \"port\": \"5432\",
      \"username\": \"postgres\",
      \"password\": \"postgres\",
      \"database\": \"source_db\"
    },
    \"dest\": {
      \"domin\": \"localhost\",
      \"port\": \"5433\",
      \"username\": \"postgres\",
      \"password\": \"postgres\",
      \"database\": \"target_db\"
    },
    \"tables\": [\"users\"]
  }"

# 2. 查询状态
curl -X GET http://localhost:8080/dts/api/tasks/${TASK_ID}/status

# 3. 暂停任务
curl -X POST http://localhost:8080/dts/api/tasks/${TASK_ID}/pause

# 4. 恢复任务
curl -X POST http://localhost:8080/dts/api/tasks/${TASK_ID}/resume

# 5. 停止任务
curl -X POST http://localhost:8080/dts/api/tasks/${TASK_ID}/stop

# 6. 启动任务
curl -X POST http://localhost:8080/dts/api/tasks/${TASK_ID}/start

# 7. 切流
curl -X POST http://localhost:8080/dts/api/tasks/${TASK_ID}/switch

# 8. 删除任务
curl -X DELETE http://localhost:8080/dts/api/tasks/${TASK_ID}
```

## 使用测试脚本

项目提供了自动化测试脚本：

```bash
# 使用默认地址 (http://localhost:8080)
./scripts/test_api.sh

# 指定服务器地址
./scripts/test_api.sh http://192.168.1.100:8080
```

## 错误处理

所有 API 在出错时都会返回以下格式：

```json
{
  "state": "ERROR",
  "message": "错误描述信息"
}
```

常见错误：
- `400 Bad Request`: 请求参数错误
- `404 Not Found`: 任务不存在
- `500 Internal Server Error`: 服务器内部错误

## 注意事项

1. **task_id**: 必须唯一，建议使用 UUID 或时间戳
2. **domin**: API 规范中使用 `domin` 而不是 `domain`
3. **database**: 可选字段，如果不指定，默认使用 `postgres`
4. **tables**: 可选字段，如果不指定，会同步源库中的所有表
5. **stop vs delete**:
   - `stop`: 停止任务运行，任务保留
   - `delete`: 删除任务，任务会被永久删除

