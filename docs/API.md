# DTS RESTful API 文档

## 概述

本文档描述了 DTS (Data Transfer Service) 的 RESTful API 接口。所有 API 遵循 RESTful 设计原则，使用 JSON 格式进行数据交换。

## 基础信息

- **Base URL**: `http://<host>:<port>/dts/api`
- **Content-Type**: `application/json`
- **字符编码**: UTF-8

## 通用响应格式

所有 API 响应都遵循以下格式：

```json
{
  "state": "OK" | "ERROR",
  "message": "错误描述（仅在 state 为 ERROR 时存在）"
}
```

## API 接口列表

### 1. 启动数据同步任务

**接口路径**: `POST /dts/api/tasks`

**功能描述**: 创建并启动一个新的数据同步任务。

**请求体**:

```json
{
  "task_id": "uuid-string",
  "source": {
    "domin": "127.0.0.1",
    "port": "6432",
    "username": "postgres",
    "password": "myPass%7ui8&UI*",
    "database": "mydb"
  },
  "dest": {
    "domin": "127.0.0.1",
    "port": "5432",
    "username": "postgres",
    "password": "myPass%7ui8&UI*",
    "database": "mydb"
  },
  "tables": ["table1", "table2"]
}
```

**字段说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| task_id | string | 是 | 任务ID，可以是 UUID 或其他字符串格式 |
| source | object | 是 | 源数据库连接信息 |
| source.domin | string | 是 | 源数据库主机地址（注意：API 规范中使用 "domin"） |
| source.port | string | 是 | 源数据库端口 |
| source.username | string | 是 | 源数据库用户名 |
| source.password | string | 是 | 源数据库密码 |
| source.database | string | 否 | 源数据库名称，默认为 username |
| dest | object | 是 | 目标数据库连接信息 |
| dest.domin | string | 是 | 目标数据库主机地址 |
| dest.port | string | 是 | 目标数据库端口 |
| dest.username | string | 是 | 目标数据库用户名 |
| dest.password | string | 是 | 目标数据库密码 |
| dest.database | string | 否 | 目标数据库名称，默认为 username |
| tables | array | 是 | 要同步的表列表 |

**响应示例**:

成功响应:
```json
{
  "state": "OK",
  "message": ""
}
```

错误响应:
```json
{
  "state": "ERROR",
  "message": "Invalid request body: task_id is required"
}
```

**HTTP 状态码**:
- `200 OK`: 任务创建并启动成功
- `400 Bad Request`: 请求参数错误
- `500 Internal Server Error`: 服务器内部错误

---

### 2. 查询同步任务状态

**接口路径**: `GET /dts/api/tasks/{task_id}`

**功能描述**: 查询指定任务的同步状态。

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| task_id | string | 任务ID |

**响应体**:

```json
{
  "state": "OK" | "ERROR",
  "message": "错误描述",
  "stage": "none" | "syncing" | "waiting" | "switching" | "finished",
  "duration": 20000,
  "delay": 5000
}
```

**字段说明**:

| 字段 | 类型 | 说明 |
|------|------|------|
| state | string | 响应状态：OK 或 ERROR |
| message | string | 错误描述（仅在 state 为 ERROR 时存在） |
| stage | string | 任务阶段：<br>- `none`: 没有同步任务<br>- `syncing`: 同步数据中<br>- `waiting`: 等待切流<br>- `switching`: 切流中<br>- `finished`: 任务完成 |
| duration | int64 | 从切流开始到完成的时间，单位毫秒（ms）。只有 `finished` 阶段该字段才有意义，其他阶段为 `-1` |
| delay | int64 | 同步延迟，单位毫秒（ms）。`-1` 表示无意义或无法计算 |

**响应示例**:

同步中:
```json
{
  "state": "OK",
  "message": "",
  "stage": "syncing",
  "duration": -1,
  "delay": 5000
}
```

切流中:
```json
{
  "state": "OK",
  "message": "",
  "stage": "switching",
  "duration": -1,
  "delay": -1
}
```

已完成:
```json
{
  "state": "OK",
  "message": "",
  "stage": "finished",
  "duration": 20000,
  "delay": -1
}
```

任务不存在:
```json
{
  "state": "ERROR",
  "message": "Task not found: task_id does not exist",
  "stage": "none",
  "duration": -1,
  "delay": -1
}
```

**HTTP 状态码**:
- `200 OK`: 查询成功
- `404 Not Found`: 任务不存在
- `500 Internal Server Error`: 服务器内部错误

---

### 3. 切流

**接口路径**: `POST /dts/api/tasks/{task_id}/switch`

**功能描述**: 触发切流操作。切流包括：停止源库写入、验证数据一致性、恢复源库写入。

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| task_id | string | 任务ID |

**请求体**: 空

**响应体**:

```json
{
  "state": "OK" | "ERROR",
  "message": "错误描述"
}
```

**响应示例**:

成功:
```json
{
  "state": "OK",
  "message": "Switchover triggered successfully"
}
```

错误（任务不在可切流状态）:
```json
{
  "state": "ERROR",
  "message": "Task is not in a state that allows switchover. Current state: init"
}
```

**HTTP 状态码**:
- `200 OK`: 切流操作已触发
- `400 Bad Request`: 任务状态不允许切流
- `404 Not Found`: 任务不存在
- `500 Internal Server Error`: 服务器内部错误

**注意事项**:
- 只有在 `syncing` 阶段的任务才能触发切流
- 切流操作包括：停止源库写入 → 验证数据 → 恢复源库写入
- 切流过程中，任务状态会依次变为 `switching` → `finished`

---

### 4. 结束任务

**接口路径**: `DELETE /dts/api/tasks/{task_id}`

**功能描述**: 结束并删除指定的同步任务。

**路径参数**:

| 参数 | 类型 | 说明 |
|------|------|------|
| task_id | string | 任务ID |

**请求体**: 空

**响应体**:

```json
{
  "state": "OK" | "ERROR",
  "message": "错误描述"
}
```

**响应示例**:

成功:
```json
{
  "state": "OK",
  "message": "Task deleted successfully"
}
```

错误:
```json
{
  "state": "ERROR",
  "message": "Failed to delete task: task not found"
}
```

**HTTP 状态码**:
- `200 OK`: 任务删除成功
- `404 Not Found`: 任务不存在
- `500 Internal Server Error`: 服务器内部错误

**注意事项**:
- 删除任务会先取消正在运行的任务，然后删除任务记录
- 删除任务会关闭所有相关的数据库连接
- 删除操作不可恢复

---

## 任务阶段说明

### stage 字段说明

| stage | 说明 | 对应内部状态 |
|-------|------|-------------|
| `none` | 没有同步任务 | init, failed, cancelled |
| `syncing` | 同步数据中 | creating_tables, migrating_data, syncing_wal |
| `waiting` | 等待切流 | paused |
| `switching` | 切流中 | stopping_writes, validating, finalizing |
| `finished` | 任务完成 | completed |

### duration 字段说明

- **含义**: 从切流开始到完成的总耗时
- **单位**: 毫秒（ms）
- **有效值**: 仅在 `stage` 为 `finished` 时有效，其他阶段为 `-1`

### delay 字段说明

- **含义**: 同步延迟，即源库与目标库之间的数据延迟
- **单位**: 毫秒（ms）
- **有效值**: 在 `syncing`、`waiting`、`switching` 阶段可能有效，其他阶段为 `-1`
- **计算方式**: 从 WAL 复制状态中获取（待实现）

---

## 错误处理

### 错误响应格式

所有错误响应都遵循以下格式：

```json
{
  "state": "ERROR",
  "message": "详细的错误描述信息"
}
```

### 常见错误码

| HTTP 状态码 | 说明 | 示例 |
|------------|------|------|
| 400 | 请求参数错误 | 缺少必填字段、参数格式错误 |
| 404 | 资源不存在 | 任务ID不存在 |
| 500 | 服务器内部错误 | 数据库连接失败、内部处理错误 |

---

## 使用示例

### 示例 1: 创建并启动同步任务

```bash
curl -X POST http://localhost:8080/dts/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "task-001",
    "source": {
      "domin": "127.0.0.1",
      "port": "6432",
      "username": "postgres",
      "password": "password123"
    },
    "dest": {
      "domin": "127.0.0.1",
      "port": "5432",
      "username": "postgres",
      "password": "password123"
    },
    "tables": ["users", "orders"]
  }'
```

### 示例 2: 查询任务状态

```bash
curl -X GET http://localhost:8080/dts/api/tasks/task-001
```

### 示例 3: 触发切流

```bash
curl -X POST http://localhost:8080/dts/api/tasks/task-001/switch
```

### 示例 4: 删除任务

```bash
curl -X DELETE http://localhost:8080/dts/api/tasks/task-001
```

---

## 注意事项

1. **任务ID唯一性**: 每个任务ID必须唯一，重复创建相同ID的任务会失败
2. **数据库连接**: 确保源数据库的 `wal_level` 设置为 `logical`
3. **表列表**: 如果不指定 `tables` 字段，需要从源库获取所有表（当前版本需要显式指定）
4. **切流时机**: 建议在数据同步完成且延迟较小时进行切流
5. **任务删除**: 删除任务会关闭所有相关连接，请谨慎操作

---

## 版本信息

- **API 版本**: v1.0
- **最后更新**: 2025-11-06

