# DTS 调试指南

## 准备工作

### 1. 确保元数据库存在

```bash
# 创建元数据库（如果不存在）
createdb -h localhost -p 5432 -U postgres dts_meta
```

元数据库配置在 `configs/config.yaml` 中，也可以通过命令行参数覆盖：
```bash
./bin/dts --db-host localhost --db-port 5432 --db-user postgres --db-password postgres --db-name dts_meta
```

### 2. 确保源数据库 wal_level 为 logical

```sql
-- 连接到源数据库（5533）
ALTER SYSTEM SET wal_level = logical;
-- 需要重启 PostgreSQL 使配置生效
```

### 3. 准备测试数据

确保源数据库（127.0.0.1:5533）中有业务表和数据。

## 启动服务器

### 方式 1: 使用脚本启动

```bash
./scripts/start_server.sh
```

### 方式 2: 直接启动

```bash
# 使用默认配置（从 configs/config.yaml 读取）
./bin/dts --log-level debug --port 8080

# 或通过命令行参数覆盖元数据库配置
./bin/dts --log-level debug --port 8080 \
  --db-host localhost \
  --db-port 5432 \
  --db-user postgres \
  --db-password postgres \
  --db-name dts_meta
```

## 创建测试任务

### 方式 1: 使用脚本

```bash
./scripts/create_test_task.sh
```

### 方式 2: 使用 curl

```bash
curl -X POST http://localhost:8080/rdscheduler/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "test-task-001",
    "source": {
      "domin": "127.0.0.1",
      "port": "5533",
      "username": "postgres",
      "password": "postgres",
      "database": "postgres"
    },
    "dest": {
      "domin": "127.0.0.1",
      "port": "5633",
      "username": "postgres",
      "password": "postgres",
      "database": "postgres"
    }
  }'
```

**注意**: 如果不指定 `tables` 字段，系统会自动从源库获取所有表。

## 查询任务状态

```bash
curl -X GET http://localhost:8080/rdscheduler/api/tasks/test-task-001 | jq
```

## 触发切流

```bash
curl -X POST http://localhost:8080/rdscheduler/api/tasks/test-task-001/switch | jq
```

## 删除任务

```bash
curl -X DELETE http://localhost:8080/rdscheduler/api/tasks/test-task-001 | jq
```

## 常见问题

### 1. 连接元数据库失败

**错误**: `Failed to connect to metadata database`

**解决**:
- 检查元数据库是否存在
- 检查环境变量配置
- 检查数据库连接权限

### 2. 源数据库连接失败

**错误**: `Failed to connect to source database`

**解决**:
- 检查源数据库是否运行
- 检查用户名密码是否正确
- 检查网络连接

### 3. wal_level 不是 logical

**错误**: `Source database wal_level must be logical`

**解决**:
```sql
ALTER SYSTEM SET wal_level = logical;
-- 重启 PostgreSQL
```

### 4. 没有找到表

**错误**: `No tables found in source database`

**解决**:
- 检查源数据库中是否有表
- 或者手动指定表列表：
```json
{
  "tables": ["table1", "table2"]
}
```

## 调试技巧

### 1. 查看日志

启动时使用 `--log-level debug` 可以看到详细日志：

```bash
./bin/dts --log-level debug --port 8080
```

### 2. 检查任务状态

使用内部管理 API 查看详细任务信息：

```bash
curl http://localhost:8080/api/v1/migrations/test-task-001 | jq
```

### 3. 查看健康状态

```bash
curl http://localhost:8080/api/v1/health
```

## 测试流程

1. **启动服务器**
   ```bash
   ./bin/dts --log-level debug --port 8080
   ```

2. **创建任务**
   ```bash
   ./scripts/create_test_task.sh
   ```

3. **监控任务状态**
   ```bash
   watch -n 2 'curl -s http://localhost:8080/rdscheduler/api/tasks/test-task-001 | jq'
   ```

4. **触发切流**（当数据同步完成后）
   ```bash
   curl -X POST http://localhost:8080/rdscheduler/api/tasks/test-task-001/switch
   ```

5. **验证数据**
   - 检查目标数据库中的数据
   - 验证行数是否一致

6. **清理任务**
   ```bash
   curl -X DELETE http://localhost:8080/rdscheduler/api/tasks/test-task-001
   ```

