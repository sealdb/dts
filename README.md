# PostgreSQL Data Transfer Service (DTS)

基于逻辑复制和 WAL 日志解析的 PostgreSQL 数据库迁移工具。

## 功能特性

- ✅ 基于逻辑复制（Logical Replication）的实时数据同步
- ✅ WAL 日志解析和应用
- ✅ 状态机驱动的迁移流程
- ✅ RESTful API 接口
- ✅ 支持表名后缀映射
- ✅ 自动数据校验和验证

## 架构设计

### 目录结构

```
dts/
├── cmd/
│   └── server/              # 应用入口
├── internal/
│   ├── api/                 # API 层
│   │   ├── handler/        # 请求处理器
│   │   └── routes.go       # 路由定义
│   ├── service/            # 业务逻辑层
│   ├── repository/         # 数据访问层
│   ├── state/               # 状态机
│   ├── wal/                 # WAL 日志处理
│   ├── replication/         # 逻辑复制
│   ├── model/               # 数据模型
│   └── config/              # 配置
├── configs/                 # 配置文件
├── migrations/              # 数据库迁移
└── go.mod
```

### 迁移流程

```
INIT → CREATING_TABLES → MIGRATING_DATA →
SYNCING_WAL → STOPPING_WRITES → VALIDATING →
FINALIZING → COMPLETED
```

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

### 2. 配置数据库

编辑 `configs/config.yaml`：

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "postgres"
  dbname: "dts_meta"
  sslmode: "disable"
```

### 3. 创建元数据库

```sql
CREATE DATABASE dts_meta;
```

### 4. 运行服务

```bash
go run cmd/server/main.go
```

### 5. 创建迁移任务

```bash
curl -X POST http://localhost:8080/api/v1/migrations \
  -H "Content-Type: application/json" \
  -d '{
    "source_db": {
      "host": "localhost",
      "port": 5432,
      "user": "postgres",
      "password": "postgres",
      "dbname": "source_db",
      "sslmode": "disable"
    },
    "target_db": {
      "host": "localhost",
      "port": 5432,
      "user": "postgres",
      "password": "postgres",
      "dbname": "target_db",
      "sslmode": "disable"
    },
    "tables": ["users", "orders"],
    "table_suffix": "_migrated"
  }'
```

### 6. 启动迁移

```bash
curl -X POST http://localhost:8080/api/v1/migrations/{task_id}/start
```

## API 文档

### 创建迁移任务

```
POST /api/v1/migrations
```

### 获取任务列表

```
GET /api/v1/migrations?limit=10&offset=0
```

### 获取任务详情

```
GET /api/v1/migrations/{id}
```

### 启动任务

```
POST /api/v1/migrations/{id}/start
```

### 暂停任务

```
POST /api/v1/migrations/{id}/pause
```

### 恢复任务

```
POST /api/v1/migrations/{id}/resume
```

### 取消任务

```
POST /api/v1/migrations/{id}/cancel
```

### 获取任务状态

```
GET /api/v1/migrations/{id}/status
```

### 健康检查

```
GET /api/v1/health
```

## 注意事项

1. **源库要求**：源库的 `wal_level` 必须设置为 `logical`
   ```sql
   ALTER SYSTEM SET wal_level = logical;
   SELECT pg_reload_conf();
   ```

2. **权限要求**：
   - 源库需要 `REPLICATION` 权限
   - 需要创建 `Publication` 和 `Replication Slot` 的权限

3. **表名映射**：目标表名会自动添加后缀，例如 `users` → `users_migrated`

4. **切流机制**：迁移过程中会停止源库写操作，确保数据一致性

## 开发状态

⚠️ 当前版本为初始开发版本，部分功能尚未完全实现：

- [x] 项目框架和状态机
- [x] API 接口定义
- [x] WAL 解析器框架
- [ ] 表结构解析和 DDL 生成
- [ ] 数据迁移逻辑
- [ ] WAL 消息处理实现
- [ ] 数据校验逻辑
- [ ] 错误处理和重试机制

## License

Apache 2.0

