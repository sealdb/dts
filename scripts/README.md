# PostgreSQL Cluster Manager

用于管理多个 PostgreSQL 集群的 Shell 脚本，支持主从复制和多种同步模式。

## 功能特性

- ✅ 支持多集群部署（默认 2 个集群）
- ✅ 支持每个集群多节点配置（默认 3 节点）
- ✅ 支持三种复制模式：异步、半同步（法定人数）、强同步
- ✅ 自动配置主从复制和流复制槽
- ✅ 支持 init/start/stop/status 操作
- ✅ 灵活的命令行参数格式（-c 或 --config，参数顺序无要求）
- ✅ 支持强制初始化（--force 选项）

## 命令行用法

### 基本语法

```bash
pg_cluster_manager.sh <action> [options]
```

### 参数说明

| 参数 | 简写 | 说明 | 默认值 | 必需 |
|------|------|------|--------|------|
| `--config` | `-c` | 指定配置文件 | ./config.conf | 否 |
| `--base-port` | 无 | 基础端口号 | 5432 | 否 |
| `--wal-level` | 无 | WAL 级别 (replica/logical) | replica | 否 |
| `--cluster-prefix` | 无 | 集群前缀 | s1_,s2_,... | 否 |
| `--force` | 无 | 强制初始化（仅限 init） | - | 否 |
| `--help` | `-h` | 显示帮助信息 | - | 否 |

### 操作（Actions）

| 操作 | 说明 |
|------|------|
| `init` | 初始化 PostgreSQL 集群 |
| `start` | 启动所有集群 |
| `stop` | 停止所有集群 |
| `status` | 查看集群状态 |

### 命令示例

```bash
# 使用默认配置（自动查找 config.conf）
./pg_cluster_manager.sh init
./pg_cluster_manager.sh start
./pg_cluster_manager.sh status

# 指定自定义配置文件
./pg_cluster_manager.sh -c my_config.conf init
./pg_cluster_manager.sh init --config my_config.conf

# 命令行覆盖配置文件参数
./pg_cluster_manager.sh init --base-port 6000 --wal-level logical
./pg_cluster_manager.sh start --base-port 5432 --cluster-prefix "s1_,s2_"
./pg_cluster_manager.sh -c config.conf init --cluster-prefix "1:primary_,2:standby_"

# 不使用配置文件，完全使用命令行参数和默认值
./pg_cluster_manager.sh init --base-port 7000 --cluster-prefix "db1_,db2_"

# 强制初始化（清除已有数据）
./pg_cluster_manager.sh init --force
./pg_cluster_manager.sh -c config.conf init --force --base-port 6000

# 查看帮助
./pg_cluster_manager.sh --help
```

### 参数详解

#### --config（配置文件）

- **可选参数**：如果不指定，自动查找脚本目录下的 `config.conf`
- **不存在也可运行**：如果配置文件不存在，使用内置默认值
- **格式**：`-c FILE` 或 `--config FILE`

```bash
# 使用默认 config.conf
./pg_cluster_manager.sh init

# 指定自定义配置
./pg_cluster_manager.sh -c prod.conf init
```

#### --base-port（基础端口）

- **默认值**：5432
- **端口分配规则**：
  - 集群1：BASE_PORT + 0, BASE_PORT + 1, BASE_PORT + 2, ...
  - 集群2：BASE_PORT + 100, BASE_PORT + 101, BASE_PORT + 102, ...
- **命令行优先**：覆盖配置文件中的 `BASE_PORT`

```bash
# 使用端口 6000 开始
./pg_cluster_manager.sh init --base-port 6000
# 集群1: 6000, 6001, 6002
# 集群2: 6100, 6101, 6102
```

#### --wal-level（WAL 级别）

- **默认值**：replica
- **可选值**：
  - `replica`：物理复制（适用于流复制和 PITR）
  - `logical`：逻辑复制（支持逻辑解码和逻辑复制）
- **命令行优先**：覆盖配置文件中的 `WAL_LEVEL`

```bash
# 启用逻辑复制
./pg_cluster_manager.sh init --wal-level logical

# 使用物理复制（默认）
./pg_cluster_manager.sh init --wal-level replica
```

#### --cluster-prefix（集群前缀）

- **默认值**：s1_, s2_, s3_, ...
- **两种格式**：
  1. 顺序格式：`"prefix1,prefix2,prefix3"` - 按顺序分配给集群1、2、3
  2. 指定格式：`"1:prefix1,2:prefix2"` - 明确指定集群编号
- **命令行优先**：覆盖配置文件中的 `CLUSTER_N_PREFIX`

```bash
# 顺序格式
./pg_cluster_manager.sh init --cluster-prefix "s1_,s2_"
# 集群1: s1_1, s1_2, s1_3
# 集群2: s2_1, s2_2, s2_3

# 指定格式
./pg_cluster_manager.sh init --cluster-prefix "1:primary_,2:standby_"
# 集群1: primary_1, primary_2, primary_3
# 集群2: standby_1, standby_2, standby_3

# 业务相关命名
./pg_cluster_manager.sh init --cluster-prefix "order_,user_"
```

#### --force（强制初始化）

- **仅用于 init**：只在初始化时有效
- **执行步骤**：
  1. 停止所有运行中的 PostgreSQL 节点
  2. 删除所有现有数据目录
  3. 从零开始重新初始化

⚠️ **警告**：使用 `--force` 会永久删除所有数据，请谨慎使用！

```bash
# 强制重新初始化
./pg_cluster_manager.sh init --force

# 结合其他参数
./pg_cluster_manager.sh init --force --base-port 6000 --wal-level logical
```

### 参数优先级

当同一参数在多处定义时，优先级如下：

```
命令行参数 > 配置文件参数 > 内置默认值
```

示例：
```bash
# config.conf 中定义：BASE_PORT=5432
# 命令行指定：--base-port 6000
# 实际使用：6000（命令行优先）

./pg_cluster_manager.sh -c config.conf init --base-port 6000
```

### 完全默认运行

脚本可以在没有任何配置文件的情况下运行，使用以下默认值：

| 参数 | 默认值 |
|------|--------|
| BASE_DATA_DIR | /tmp/pg_clusters |
| BASE_PORT | 5432 |
| WAL_LEVEL | replica |
| CLUSTER_COUNT | 2 |
| NODE_COUNT | 3 |
| CLUSTER_PREFIX | s1_, s2_ |
| REPLICATION_MODE | async |

```bash
# 完全使用默认值（无需任何配置文件）
./pg_cluster_manager.sh init
./pg_cluster_manager.sh start
```

## 快速开始

### 方式 1：使用默认配置（最简单）

```bash
# 直接运行，无需任何配置文件
cd /home/wslu/work/pg/dts
./scripts/pg_cluster_manager.sh init
./scripts/pg_cluster_manager.sh start
./scripts/pg_cluster_manager.sh status
```

### 方式 2：使用配置文件

1. **准备配置文件**（可选，如不存在会使用默认值）

```bash
# 使用脚本目录下的 config.conf
cp scripts/config.conf my_config.conf
# 编辑配置文件...
```

2. **初始化集群**

```bash
# 使用默认 config.conf
./scripts/pg_cluster_manager.sh init

# 或指定自定义配置
./scripts/pg_cluster_manager.sh -c my_config.conf init

# 强制重新初始化
./scripts/pg_cluster_manager.sh init --force
```

3. **启动集群**

```bash
./scripts/pg_cluster_manager.sh start
```

4. **检查状态**

```bash
./scripts/pg_cluster_manager.sh status
```

5. **停止集群**

```bash
./scripts/pg_cluster_manager.sh stop
```

### 方式 3：命令行参数覆盖

```bash
# 使用命令行参数覆盖配置文件或默认值
./scripts/pg_cluster_manager.sh init \
    --base-port 6000 \
    --wal-level logical \
    --cluster-prefix "db1_,db2_"

# 启动时也需要使用相同的参数
./scripts/pg_cluster_manager.sh start --base-port 6000

# 查看状态
./scripts/pg_cluster_manager.sh status --base-port 6000
```

### 初始化过程说明

无论使用哪种方式，初始化过程都会：
- 在指定的数据目录创建集群结构
- 初始化所有节点的数据库
- 配置主从复制
- 设置复制槽和流复制
- 根据 WAL 级别配置物理或逻辑复制

## 配置说明

### 配置文件参数

所有参数都是可选的，未指定时使用默认值。

| 参数 | 说明 | 默认值 | 可被命令行覆盖 |
|------|------|--------|----------------|
| `BASE_DATA_DIR` | 基础数据目录 | `/tmp/pg_clusters` | 否 |
| `PG_BIN_DIR` | PostgreSQL 二进制目录 | 空（使用 PATH） | 否 |
| `BASE_PORT` | 基础端口号 | 5432 | ✅ 是 |
| `WAL_LEVEL` | WAL 级别 | replica | ✅ 是 |
| `CLUSTER_COUNT` | 集群数量 | 2 | 否 |
| `NODE_COUNT` | 每集群节点数 | 3 | 否 |
| `REPLICATION_MODE` | 复制模式 | async | 否 |
| `CLUSTER_N_PREFIX` | 集群 N 的数据目录前缀 | `s<N>_` | ✅ 是 |

### 复制模式详解

#### 1. 异步复制 (async)
```bash
REPLICATION_MODE="async"
```
- **特点**：性能最高，主节点不等待从节点确认
- **风险**：主节点故障时可能丢失部分数据
- **适用**：开发环境、对数据一致性要求不高的场景

#### 2. 半同步复制 (quorum)
```bash
REPLICATION_MODE="quorum"
```
- **特点**：等待**大多数**从节点确认（例如 3 节点集群等待 1 个，5 节点集群等待 2 个）
- **风险**：平衡了性能和数据安全
- **适用**：生产环境推荐配置

#### 3. 强同步复制 (sync)
```bash
REPLICATION_MODE="sync"
```
- **特点**：等待**所有**从节点确认
- **风险**：性能最低，但数据安全性最高
- **适用**：金融、医疗等对数据一致性要求极高的场景

### 自定义集群前缀

每个集群可以配置独立的数据目录前缀，便于识别和管理。

#### 配置方法

```bash
# 集群 1 使用前缀 "s1_"
CLUSTER_1_PREFIX="s1_"

# 集群 2 使用前缀 "s2_"
CLUSTER_2_PREFIX="s2_"

# 可以使用任意有意义的前缀
CLUSTER_1_PREFIX="primary_"
CLUSTER_2_PREFIX="backup_"
```

#### 前缀效果

- **未配置前缀**：数据目录为 `BASE_DATA_DIR/cluster1_1`, `cluster1_2`, ...
- **配置 `s1_` 前缀**：数据目录为 `BASE_DATA_DIR/s1_1`, `s1_2`, `s1_3`, ...

#### 应用场景

1. **区分环境**：`dev1_`, `test1_`, `prod1_`
2. **明确角色**：`primary_`, `standby_`, `readonly_`
3. **业务区分**：`order_`, `user_`, `payment_`
4. **简洁命名**：`s1_`, `s2_`, `c1_`, `c2_`

## 集群架构

### 默认配置（2 集群 × 3 节点）

```
集群1:
  - Node 1 (主节点): localhost:5432
  - Node 2 (从节点): localhost:5433
  - Node 3 (从节点): localhost:5434

集群2:
  - Node 1 (主节点): localhost:5532
  - Node 2 (从节点): localhost:5533
  - Node 3 (从节点): localhost:5534
```

### 端口分配规则

```
Port = BASE_PORT + (集群ID - 1) × 100 + (节点ID - 1)
```

### 数据目录结构

#### 默认结构（不指定前缀）
```
BASE_DATA_DIR/
├── cluster1_1  (集群1主节点)
├── cluster1_2  (集群1从节点)
├── cluster1_3  (集群1从节点)
├── cluster2_1  (集群2主节点)
├── cluster2_2  (集群2从节点)
└── cluster2_3  (集群2从节点)
```

#### 自定义前缀结构
使用配置 `CLUSTER_1_PREFIX="s1_"` 和 `CLUSTER_2_PREFIX="s2_"`：
```
BASE_DATA_DIR/
├── s1_1  (集群1主节点)
├── s1_2  (集群1从节点)
├── s1_3  (集群1从节点)
├── s2_1  (集群2主节点)
├── s2_2  (集群2从节点)
└── s2_3  (集群2从节点)
```

## 使用示例

### 开发环境（快速测试）

```bash
cat > dev_config.conf << EOF
BASE_DATA_DIR="/tmp/pg_dev"
BASE_PORT=6000
CLUSTER_COUNT=1
NODE_COUNT=2
REPLICATION_MODE="async"
EOF

./scripts/pg_cluster_manager.sh -c dev_config.conf init
./scripts/pg_cluster_manager.sh -c dev_config.conf start
```

### 生产环境（5节点高可用）

```bash
cat > prod_config.conf << EOF
BASE_DATA_DIR="/data/pg_clusters"
PG_BIN_DIR="/usr/local/pgsql/bin"
BASE_PORT=5432
CLUSTER_COUNT=2
NODE_COUNT=5
REPLICATION_MODE="quorum"
EOF

./scripts/pg_cluster_manager.sh -c prod_config.conf init
./scripts/pg_cluster_manager.sh -c prod_config.conf start
```

### 使用自定义前缀

```bash
cat > custom_prefix_config.conf << EOF
BASE_DATA_DIR="/tmp/pg_data"
BASE_PORT=5432
CLUSTER_COUNT=2
NODE_COUNT=3
REPLICATION_MODE="quorum"

# 自定义每个集群的数据目录前缀
CLUSTER_1_PREFIX="s1_"
CLUSTER_2_PREFIX="s2_"
EOF

./scripts/pg_cluster_manager.sh -c custom_prefix_config.conf init
./scripts/pg_cluster_manager.sh -c custom_prefix_config.conf start

# 查看创建的目录结构
ls -l /tmp/pg_data/
# 输出：
# s1_1/  s1_2/  s1_3/  (集群1的3个节点)
# s2_1/  s2_2/  s2_3/  (集群2的3个节点)
```

## 连接到集群

```bash
# 连接到集群1的主节点
psql -h localhost -p 5432 -U postgres

# 连接到集群1的从节点
psql -h localhost -p 5433 -U postgres

# 连接到集群2的主节点
psql -h localhost -p 5532 -U postgres
```

## 验证复制状态

在主节点执行：

```sql
-- 查看复制状态
SELECT * FROM pg_stat_replication;

-- 查看复制槽
SELECT * FROM pg_replication_slots;
```

在从节点执行：

```sql
-- 查看恢复状态
SELECT * FROM pg_stat_wal_receiver;

-- 检查是否为从节点
SELECT pg_is_in_recovery();
```

## 故障排查

### 检查日志

```bash
# 查看主节点日志
tail -f /tmp/pg_clusters/cluster1/node1/logfile

# 查看从节点日志
tail -f /tmp/pg_clusters/cluster1/node2/logfile
```

### 重新初始化

如果需要重新开始，有两种方式：

#### 方式 1：使用 --force 参数（推荐）

```bash
# 一条命令完成：停止服务、删除数据、重新初始化
./scripts/pg_cluster_manager.sh -c my_config.conf init --force
./scripts/pg_cluster_manager.sh -c my_config.conf start
```

#### 方式 2：手动清理

```bash
# 停止所有集群
./scripts/pg_cluster_manager.sh -c my_config.conf stop

# 删除数据目录
rm -rf /tmp/pg_clusters

# 重新初始化
./scripts/pg_cluster_manager.sh -c my_config.conf init
./scripts/pg_cluster_manager.sh -c my_config.conf start
```

## 注意事项

1. **首次初始化**：`init` 操作会自动启动主节点来创建复制用户和复制槽，完成后会自动停止
2. **端口冲突**：确保配置的端口未被占用
3. **磁盘空间**：确保有足够的磁盘空间存储数据
4. **PostgreSQL 版本**：脚本支持 PostgreSQL 10+ 版本
5. **网络配置**：默认配置允许任何主机连接（trust），生产环境请修改 `pg_hba.conf`

## 高级功能

### 自定义 PostgreSQL 参数

初始化后可以手动编辑 `postgresql.conf`：

```bash
vi /tmp/pg_clusters/cluster1/node1/postgresql.conf
```

然后重启集群：

```bash
./scripts/pg_cluster_manager.sh my_config.conf stop
./scripts/pg_cluster_manager.sh my_config.conf start
```

### 添加更多从节点

修改配置文件增加 `NODE_COUNT`，然后重新运行 `init`（会跳过已存在的节点）。

## 许可证

本脚本用于项目内部测试和开发使用。

