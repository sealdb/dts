# 开发进度文档

## 已完成的工作

### 1. 项目框架搭建 ✅

- [x] Go 项目目录结构
- [x] go.mod 依赖管理
- [x] 配置文件系统
- [x] 基础模型定义

### 2. 状态机实现 ✅

- [x] 状态接口定义 (`State`)
- [x] 状态机实现 (`StateMachine`)
- [x] 所有状态实现：
  - [x] InitState - 初始化
  - [x] CreatingTablesState - 创建表
  - [x] MigratingDataState - 迁移数据
  - [x] SyncingWALState - 同步WAL
  - [x] StoppingWritesState - 停止写操作
  - [x] ValidatingState - 数据校验
  - [x] FinalizingState - 完成迁移
  - [x] CompletedState - 已完成
  - [x] FailedState - 失败
  - [x] PausedState - 暂停

### 3. 数据模型 ✅

- [x] MigrationTask - 迁移任务模型
- [x] DBConfig - 数据库配置
- [x] TableInfo - 表结构信息
- [x] 状态类型定义和转换规则

### 4. Repository 层 ✅

- [x] MigrationRepository - 任务仓储
- [x] SourceRepository - 源库操作（框架）
- [x] TargetRepository - 目标库操作（框架）

### 5. WAL 处理框架 ✅

- [x] Decoder - WAL 消息解码器
- [x] Message - WAL 消息类型定义
- [x] Handler - WAL 变更处理器

### 6. 逻辑复制框架 ✅

- [x] SlotManager - 复制槽管理
- [x] PublicationManager - 发布管理
- [x] Subscriber - WAL 订阅者

### 7. Service 层 ✅

- [x] MigrationService - 迁移服务
- [x] 任务创建、查询、启动、暂停、恢复、取消

### 8. API 层 ✅

- [x] RESTful API 路由定义
- [x] MigrationHandler - 迁移任务 API
- [x] HealthHandler - 健康检查 API
- [x] 所有 API 端点实现

### 9. 应用入口 ✅

- [x] main.go - 服务器启动
- [x] 数据库连接和迁移
- [x] 路由注册

## 待实现的功能

### 1. 表结构解析 ⏳

- [ ] `SourceRepository.GetTableInfo()` - 从 information_schema 查询表结构
- [ ] 解析列信息（类型、约束、默认值等）
- [ ] 解析索引信息
- [ ] 解析约束信息（主键、外键、唯一约束等）
- [ ] 生成 DDL 语句

### 2. 目标表创建 ⏳

- [ ] `TargetRepository.CreateTable()` - 创建目标表
- [ ] 表名映射（添加后缀）
- [ ] 执行 DDL 语句
- [ ] 创建索引和约束

### 3. 数据迁移 ⏳

- [ ] `TargetRepository.CopyData()` - 数据复制逻辑
- [ ] 使用 COPY 或批量 INSERT
- [ ] 进度跟踪
- [ ] 错误处理

### 4. WAL 同步实现 ⏳

- [ ] `Handler.handleInsert()` - 处理 INSERT 消息
- [ ] `Handler.handleUpdate()` - 处理 UPDATE 消息
- [ ] `Handler.handleDelete()` - 处理 DELETE 消息
- [ ] `Handler.handleTruncate()` - 处理 TRUNCATE 消息
- [ ] Tuple 数据解析
- [ ] 应用到目标库

### 5. 源库写操作控制 ⏳

- [ ] `SourceRepository.SetReadOnly()` - 设置只读模式
- [ ] `SourceRepository.RevokeWritePermissions()` - 撤销写权限
- [ ] `SourceRepository.RestoreWritePermissions()` - 恢复写权限

### 6. 数据校验 ⏳

- [ ] 行数对比
- [ ] 可选：抽样数据对比
- [ ] 校验结果报告

### 7. 状态执行逻辑 ⏳

- [ ] InitState.Execute() - 验证源库和目标库
- [ ] CreatingTablesState.Execute() - 创建目标表
- [ ] MigratingDataState.Execute() - 迁移数据
- [ ] SyncingWALState.Execute() - 启动 WAL 同步
- [ ] StoppingWritesState.Execute() - 停止源库写操作
- [ ] ValidatingState.Execute() - 数据校验
- [ ] FinalizingState.Execute() - 清理资源、恢复权限

### 8. 错误处理和重试 ⏳

- [ ] 错误分类和处理策略
- [ ] 重试机制
- [ ] 资源清理（复制槽、Publication）
- [ ] 失败回滚

### 9. 进度跟踪 ⏳

- [ ] 实时进度更新
- [ ] 迁移速度统计
- [ ] 预估完成时间

### 10. 日志和监控 ⏳

- [ ] 结构化日志
- [ ] 操作审计
- [ ] 性能指标收集

## 技术要点

### WAL 消息处理

使用 `pglogrepl` 库解析 PostgreSQL 逻辑复制消息：

1. **RelationMessage** - 表结构定义
2. **InsertMessage** - 插入操作
3. **UpdateMessage** - 更新操作
4. **DeleteMessage** - 删除操作
5. **TruncateMessage** - 截断操作
6. **BeginMessage** - 事务开始
7. **CommitMessage** - 事务提交

### 表名映射

源表名 → 目标表名（带后缀）：
- `users` → `users_migrated`
- `orders` → `orders_migrated`

### 切流机制

1. 设置数据库只读模式
2. 或撤销应用用户的写权限
3. 等待所有写操作完成
4. 校验数据一致性
5. 恢复写权限

## 下一步计划

1. **实现表结构解析** - 从 information_schema 查询并生成 DDL
2. **实现数据复制** - 使用 COPY 命令批量复制数据
3. **实现 WAL 消息处理** - 解析并应用变更到目标库
4. **完善状态执行逻辑** - 实现每个状态的具体逻辑
5. **测试和优化** - 单元测试、集成测试、性能优化

## 参考资源

- [PostgreSQL Logical Replication](https://www.postgresql.org/docs/current/logical-replication.html)
- [pglogrepl Library](https://github.com/jackc/pglogrepl)
- [GORM Documentation](https://gorm.io/docs/)
- [Gin Framework](https://gin-gonic.com/docs/)





