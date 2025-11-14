# PostgreSQL Cluster Manager Scripts

本目录包含用于管理 PostgreSQL 集群的脚本工具。

## 脚本说明

### pgctl.sh

PostgreSQL 集群管理脚本，支持创建和管理多个 PostgreSQL 集群，每个集群可以配置主从复制。

#### 功能特性

- ✅ 支持多集群部署（可指定集群数量）
- ✅ 支持每个集群 2 或 3 节点配置（1 主 + 1 从 或 1 主 + 2 从）
- ✅ 支持物理复制和逻辑复制
- ✅ 自动配置主从复制和流复制槽
- ✅ 支持 init/start/stop/status 操作
- ✅ 自动内存配置优化（基于系统内存的 60%）
- ✅ 支持指定 PostgreSQL 二进制目录或使用 PATH
- ✅ 灵活的配置方式（配置文件或命令行参数）

#### 基本用法

```bash
# 查看帮助
./scripts/pgctl.sh --help

# 初始化集群（使用默认配置）
./scripts/pgctl.sh init

# 启动集群
./scripts/pgctl.sh start

# 查看状态
./scripts/pgctl.sh status

# 停止集群
./scripts/pgctl.sh stop
```

#### 命令行参数

| 参数 | 简写 | 说明 | 默认值 | 必需 |
|------|------|------|--------|------|
| `<action>` | - | 操作：init\|start\|stop\|status | - | ✅ 是 |
| `--config` | `-c` | 指定配置文件 | ./config.conf | 否 |
| `--bin-dir` | - | PostgreSQL 二进制目录 | 使用 PATH | 否 |
| `--base-port` | - | 基础端口号 | 5432 | 否 |
| `--wal-level` | - | WAL 级别：replica\|logical | logical | 否 |
| `--cluster-prefix` | - | 集群前缀 | s1, s2, ... | 否 |
| `--cluster-count` | - | 集群数量 | 1 | 否 |
| `--node-count` | - | 每集群节点数：2 或 3 | 3 | 否 |
| `--force` | - | 强制初始化（仅限 init） | - | 否 |
| `--help` | `-h` | 显示帮助信息 | - | 否 |

#### 操作说明

##### init - 初始化集群

初始化 PostgreSQL 集群，包括：
- 创建数据目录
- 初始化数据库
- 配置主从复制
- 设置复制槽

```bash
# 使用默认配置初始化
./scripts/pgctl.sh init

# 初始化多个集群
./scripts/pgctl.sh init --cluster-count 2 --node-count 3

# 指定自定义端口和前缀
./scripts/pgctl.sh init --base-port 6000 --cluster-prefix "s1,s2" --cluster-count 2

# 使用逻辑复制
./scripts/pgctl.sh init --wal-level logical

# 强制重新初始化（会删除现有数据）
./scripts/pgctl.sh init --force
```

##### start - 启动集群

启动所有已初始化的集群节点。

```bash
# 启动所有集群
./scripts/pgctl.sh start

# 使用自定义参数启动
./scripts/pgctl.sh start --base-port 6000 --cluster-count 2
```

##### stop - 停止集群

停止所有运行中的集群节点。

```bash
# 停止所有集群
./scripts/pgctl.sh stop

# 使用自定义参数停止
./scripts/pgctl.sh stop --base-port 6000 --cluster-count 2
```

##### status - 查看状态

查看所有集群的运行状态。

```bash
# 查看所有集群状态
./scripts/pgctl.sh status

# 使用自定义参数查看状态
./scripts/pgctl.sh status --base-port 6000 --cluster-count 2
```

#### PostgreSQL 二进制目录

脚本支持两种方式指定 PostgreSQL 二进制文件：

##### 方式 1：使用 PATH（推荐）

如果 PostgreSQL 已安装并配置在 PATH 中：

```bash
# 直接使用，脚本会自动从 PATH 查找
./scripts/pgctl.sh init
./scripts/pgctl.sh start
```

##### 方式 2：指定 bin 目录

如果 PostgreSQL 二进制文件不在 PATH 中，可以使用 `--bin-dir` 参数：

```bash
# 指定 PostgreSQL 二进制目录
./scripts/pgctl.sh init --bin-dir /usr/local/pgsql/bin
./scripts/pgctl.sh start --bin-dir /usr/local/pgsql/bin
./scripts/pgctl.sh stop --bin-dir /usr/local/pgsql/bin
./scripts/pgctl.sh status --bin-dir /usr/local/pgsql/bin
```

**注意**：如果指定了 `--bin-dir`，脚本会验证以下二进制文件是否存在且可执行：
- `initdb`
- `pg_ctl`
- `psql`
- `pg_basebackup`

如果未指定 `--bin-dir`，脚本会从 PATH 中查找这些命令，如果找不到会报错并提示使用 `--bin-dir` 选项。

#### 配置方式

##### 方式 1：使用配置文件

创建配置文件 `scripts/config.conf`：

```bash
# PostgreSQL binary directory (optional)
PG_BIN_DIR="/usr/local/pgsql/bin"

# Base data directory
BASE_DATA_DIR="/path/to/data"

# Base port number
BASE_PORT=5432

# Number of clusters
CLUSTER_COUNT=2

# Number of nodes per cluster (2 or 3)
NODE_COUNT=3

# WAL level: replica or logical
WAL_LEVEL="logical"

# Cluster prefixes
CLUSTER_1_PREFIX="s1"
CLUSTER_2_PREFIX="s2"

# Replication mode: async, quorum, or sync
REPLICATION_MODE="async"
```

使用配置文件：

```bash
# 使用默认配置文件（scripts/config.conf）
./scripts/pgctl.sh init

# 指定自定义配置文件
./scripts/pgctl.sh -c /path/to/config.conf init
```

##### 方式 2：使用命令行参数

命令行参数会覆盖配置文件中的设置：

```bash
# 完全使用命令行参数，不使用配置文件
./scripts/pgctl.sh init \
    --base-port 6000 \
    --wal-level logical \
    --cluster-count 2 \
    --node-count 3 \
    --cluster-prefix "s1,s2" \
    --bin-dir /usr/local/pgsql/bin
```

##### 方式 3：混合使用

配置文件 + 命令行参数覆盖：

```bash
# 使用配置文件，但覆盖部分参数
./scripts/pgctl.sh init --base-port 7000 --wal-level replica
```

**参数优先级**：命令行参数 > 配置文件 > 默认值

#### 数据目录结构

默认数据目录为当前工作目录下的 `data/` 目录。

节点命名格式：`data/{prefix}-{node_id}`

示例：
- 集群 1，节点 1：`data/s1-1`
- 集群 1，节点 2：`data/s1-2`
- 集群 1，节点 3：`data/s1-3`
- 集群 2，节点 1：`data/s2-1`
- 集群 2，节点 2：`data/s2-2`
- 集群 2，节点 3：`data/s2-3`

#### 端口分配规则

端口分配公式：`BASE_PORT + (cluster_id - 1) * 100 + (node_id - 1)`

示例（BASE_PORT=5432）：
- 集群 1，节点 1：5432
- 集群 1，节点 2：5433
- 集群 1，节点 3：5434
- 集群 2，节点 1：5532
- 集群 2，节点 2：5533
- 集群 2，节点 3：5534

#### 集群前缀格式

支持两种格式：

1. **顺序格式**：`"s1,s2"` - 按顺序分配给集群 1、2、3...
   ```bash
   ./scripts/pgctl.sh init --cluster-prefix "s1,s2" --cluster-count 2
   ```

2. **指定格式**：`"1:s1,2:s2"` - 明确指定集群编号
   ```bash
   ./scripts/pgctl.sh init --cluster-prefix "1:primary,2:standby" --cluster-count 2
   ```

#### 内存自动配置

脚本会自动检测系统总内存，并根据以下规则配置 PostgreSQL 内存参数：

- **总内存限制**：不超过系统总内存的 60%
- **shared_buffers**：可用内存的 25%，向下取整到 128MB 的倍数，最小 128MB
- **maintenance_work_mem**：根据系统大小自动调整（128MB-1GB）
- **max_wal_size** 和 **min_wal_size**：根据系统大小自动调整（4GB-48GB）

所有内存配置都符合 PostgreSQL 官方推荐值。

#### PostgreSQL 配置

脚本会自动生成优化的 PostgreSQL 配置文件，包括：

- 网络配置（listen_addresses, port）
- 连接配置（max_connections, superuser_reserved_connections）
- 内存配置（自动计算）
- WAL 配置（wal_level, max_wal_size, min_wal_size）
- 复制配置（max_wal_senders, hot_standby）
- 日志配置（自动日志轮转）
- 时区配置（timezone = 'PRC'）
- 其他优化配置

#### 日志轮转配置

脚本已启用 PostgreSQL 自带的日志轮转功能，配置如下：

- **日志格式**：CSV 格式（便于分析和导入数据库）
- **日志目录**：`data/{prefix}-{node_id}/log/`
- **日志文件名**：`postgresql-%d.log`（按月份中的日期命名，01-31）
- **日志文件权限**：0600（仅所有者可读写）
- **轮转策略**：
  - 按时间轮转：每天自动轮转（`log_rotation_age = 1d`）
  - 按大小轮转：当日志文件达到 100MB 时自动轮转（`log_rotation_size = 100MB`）
  - 轮转时截断：`log_truncate_on_rotation = on`（新日志覆盖旧日志）
  - **自动保留**：通过 `log_filename = 'postgresql-%d.log'` 格式，PostgreSQL 会自动覆盖超过 31 天的旧日志（只保留最近一个月）
- **日志内容**：
  - 记录所有连接和断开（`log_connections`, `log_disconnections`）
  - 记录所有 DDL 语句（`log_statement = 'ddl'`）
  - 记录慢查询（执行时间超过 1 秒的查询，`log_min_duration_statement = 1000`）
  - 记录锁等待（`log_lock_waits = on`）
  - 详细错误信息（`log_error_verbosity = verbose`）

**日志文件位置示例**：
```
data/s1-1/log/postgresql-01.log  # 每月1号的日志
data/s1-1/log/postgresql-02.log  # 每月2号的日志
...
data/s1-1/log/postgresql-31.log  # 每月31号的日志
```

**说明**：使用 `%d` 格式（月份中的日期），PostgreSQL 会在同一天自动覆盖同名文件，因此最多保留 31 天的日志（一个月）。

**查看日志**：
```bash
# 查看今天的日志（例如今天是15号）
tail -f data/s1-1/log/postgresql-$(date +%d).log

# 查看所有日志文件
ls -lh data/s1-1/log/

# 查看特定日期的日志（例如查看5号的日志）
cat data/s1-1/log/postgresql-05.log

# 使用 psql 查看 CSV 格式日志（需要先导入到表）
```

#### 使用示例

##### 示例 1：单集群，3 节点（1 主 2 从）

```bash
# 初始化
./scripts/pgctl.sh init --node-count 3

# 启动
./scripts/pgctl.sh start

# 查看状态
./scripts/pgctl.sh status

# 连接主节点
psql -h localhost -p 5432 -U postgres
```

##### 示例 2：多集群，自定义配置

```bash
# 初始化 2 个集群，每个 3 节点
./scripts/pgctl.sh init \
    --cluster-count 2 \
    --node-count 3 \
    --base-port 6000 \
    --cluster-prefix "s1,s2" \
    --wal-level logical

# 启动所有集群
./scripts/pgctl.sh start \
    --cluster-count 2 \
    --base-port 6000

# 查看状态
./scripts/pgctl.sh status \
    --cluster-count 2 \
    --base-port 6000
```

##### 示例 3：指定 PostgreSQL 二进制目录

```bash
# 如果 PostgreSQL 安装在 /usr/local/pgsql
./scripts/pgctl.sh init --bin-dir /usr/local/pgsql/bin
./scripts/pgctl.sh start --bin-dir /usr/local/pgsql/bin
```

##### 示例 4：使用配置文件

```bash
# 创建配置文件 scripts/config.conf
cat > scripts/config.conf << EOF
BASE_PORT=6000
CLUSTER_COUNT=2
NODE_COUNT=3
WAL_LEVEL="logical"
CLUSTER_1_PREFIX="s1"
CLUSTER_2_PREFIX="s2"
PG_BIN_DIR="/usr/local/pgsql/bin"
EOF

# 使用配置文件初始化
./scripts/pgctl.sh init

# 启动（使用相同配置）
./scripts/pgctl.sh start
```

#### 故障排查

##### 问题：找不到 PostgreSQL 二进制文件

**错误信息**：
```
[ERROR] initdb not found in PATH. Please install PostgreSQL or use --bin-dir option
```

**解决方法**：
1. 确保 PostgreSQL 已安装
2. 将 PostgreSQL bin 目录添加到 PATH，或
3. 使用 `--bin-dir` 参数指定二进制目录

##### 问题：端口已被占用

**错误信息**：
```
[ERROR] port 5432 is already in use
```

**解决方法**：
1. 使用 `--base-port` 指定其他端口
2. 或停止占用端口的进程

##### 问题：数据目录已存在

**错误信息**：
```
[WARN] Data directory already exists: data/s1-1 (skipping initialization)
```

**解决方法**：
1. 使用 `--force` 选项强制重新初始化（会删除现有数据）
2. 或手动删除数据目录

#### 注意事项

1. **数据安全**：使用 `--force` 选项会删除现有数据，请谨慎使用
2. **端口冲突**：确保指定的端口范围未被其他服务占用
3. **权限要求**：需要有创建目录和文件的权限
4. **PostgreSQL 版本**：建议使用 PostgreSQL 12 或更高版本
5. **内存配置**：自动内存配置基于系统总内存，如有特殊需求可手动调整配置文件

---

### pgctl_demo.sh

整合了快速演示、全面测试和服务器启动功能的演示脚本。

#### 功能特性

- ✅ 快速演示（quick）：展示基本功能
- ✅ 全面测试（test）：测试所有功能
- ✅ 服务器启动（server）：启动 DTS 服务器
- ✅ 全部运行（all）：依次运行所有演示

#### 基本用法

```bash
# 运行所有演示
./scripts/pgctl_demo.sh

# 运行快速演示
./scripts/pgctl_demo.sh quick

# 运行全面测试
./scripts/pgctl_demo.sh test

# 启动 DTS 服务器
./scripts/pgctl_demo.sh server

# 使用自定义参数
./scripts/pgctl_demo.sh quick --base-port 6000 --cluster-count 2
```

#### 参数说明

支持与 `pgctl.sh` 相同的命令行参数，用于传递给底层脚本。

---

## 常见问题

### Q: 如何查看集群的详细日志？

A: 日志文件位于每个节点的数据目录下的 `log/` 目录中，按月份中的日期命名（01-31）：
```bash
# 查看今天的日志（例如今天是15号）
tail -f data/s1-1/log/postgresql-$(date +%d).log

# 查看所有日志文件
ls -lh data/s1-1/log/

# 查看特定日期的日志（例如查看5号的日志）
cat data/s1-1/log/postgresql-05.log
```

**注意**：日志文件是 CSV 格式，可以使用文本编辑器查看，也可以导入到 PostgreSQL 表中进行分析。

### Q: 如何修改 PostgreSQL 配置？

A: 编辑对应节点的 `postgresql.conf` 文件：
```bash
vi data/s1-1/postgresql.conf
```

然后重启对应节点：
```bash
./scripts/pgctl.sh stop
./scripts/pgctl.sh start
```

### Q: 如何添加更多节点？

A: 目前脚本只支持 2 或 3 节点配置。如需更多节点，需要手动配置或修改脚本。

### Q: 如何备份集群数据？

A: 可以使用 PostgreSQL 的标准备份工具：
```bash
# 使用 pg_dump
pg_dump -h localhost -p 5432 -U postgres database_name > backup.sql

# 或使用 pg_basebackup 进行物理备份
pg_basebackup -h localhost -p 5432 -U replicator -D /path/to/backup -Fp -Xs -P
```

---

## 更新日志

### v2.0
- ✅ 支持指定 PostgreSQL 二进制目录（--bin-dir）
- ✅ 添加二进制文件存在性检查
- ✅ 数据目录默认改为当前路径下的 data/
- ✅ 节点命名格式改为使用连字符（s1-1, s1-2）
- ✅ 支持指定集群数量和节点数量
- ✅ 自动内存配置优化
- ✅ 完整的 PostgreSQL 配置选项

### v1.0
- ✅ 基本集群管理功能
- ✅ 主从复制支持
- ✅ 配置文件支持

---

## 贡献

如有问题或建议，请提交 Issue 或 Pull Request。

