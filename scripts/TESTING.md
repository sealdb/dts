# PostgreSQL Cluster Manager - 测试指南

## 测试脚本说明

### test_all_features.sh

全面测试脚本，测试所有功能特性。

**注意事项**：
- 脚本会自动使用 `--force` 参数确保每个测试的干净状态
- 默认使用脚本目录下的 `config.conf`（如果存在）
- 测试完成后会自动清理测试数据

**运行方法**：
```bash
cd /home/wslu/work/pg/dts
./scripts/test_all_features.sh
```

**测试内容**：
1. ✅ 默认配置测试（无配置文件）
2. ✅ 命令行 `--base-port` 参数覆盖
3. ✅ 命令行 `--wal-level` 参数（replica/logical）
4. ✅ 命令行 `--cluster-prefix` 参数
5. ✅ `--force` 强制重新初始化
6. ✅ 配置文件与命令行参数组合
7. ✅ 参数优先级验证（CLI > Config > Defaults）

## 快速测试示例

### 测试 1：完全使用默认值

```bash
# 无需任何配置文件
./scripts/pg_cluster_manager.sh init --force
./scripts/pg_cluster_manager.sh start
./scripts/pg_cluster_manager.sh status
./scripts/pg_cluster_manager.sh stop

# 清理
rm -rf /tmp/pg_clusters
```

### 测试 2：命令行参数覆盖

```bash
# 自定义端口和前缀
./scripts/pg_cluster_manager.sh init \
    --base-port 6000 \
    --cluster-prefix "test1_,test2_" \
    --force

./scripts/pg_cluster_manager.sh start --base-port 6000
./scripts/pg_cluster_manager.sh status --base-port 6000
./scripts/pg_cluster_manager.sh stop --base-port 6000

# 清理
rm -rf /tmp/pg_clusters
```

### 测试 3：逻辑复制

```bash
# 启用逻辑复制
./scripts/pg_cluster_manager.sh init \
    --wal-level logical \
    --base-port 7000 \
    --force

./scripts/pg_cluster_manager.sh start --base-port 7000

# 验证 WAL 级别
psql -h localhost -p 7000 -U postgres -c "SHOW wal_level;"

./scripts/pg_cluster_manager.sh stop --base-port 7000

# 清理
rm -rf /tmp/pg_clusters
```

### 测试 4：使用配置文件

```bash
# 创建自定义配置
cat > /tmp/my_test.conf << EOF
BASE_DATA_DIR="/tmp/my_pg_test"
BASE_PORT=8000
WAL_LEVEL="logical"
CLUSTER_COUNT=2
NODE_COUNT=2
CLUSTER_1_PREFIX="app1_"
CLUSTER_2_PREFIX="app2_"
REPLICATION_MODE="async"
EOF

# 使用配置文件初始化
./scripts/pg_cluster_manager.sh -c /tmp/my_test.conf init --force

# 启动并查看状态
./scripts/pg_cluster_manager.sh -c /tmp/my_test.conf start
./scripts/pg_cluster_manager.sh -c /tmp/my_test.conf status

# 命令行参数可以覆盖配置文件
./scripts/pg_cluster_manager.sh -c /tmp/my_test.conf stop
./scripts/pg_cluster_manager.sh -c /tmp/my_test.conf init \
    --base-port 9000 \
    --cluster-prefix "new1_,new2_" \
    --force

./scripts/pg_cluster_manager.sh -c /tmp/my_test.conf start --base-port 9000
./scripts/pg_cluster_manager.sh -c /tmp/my_test.conf status --base-port 9000
./scripts/pg_cluster_manager.sh -c /tmp/my_test.conf stop --base-port 9000

# 清理
rm -rf /tmp/my_pg_test /tmp/my_test.conf
```

## 常见问题

### Q1: 测试失败，提示端口冲突

**原因**：数据目录已存在，但端口配置不同

**解决方案**：
```bash
# 方案 1：使用 --force 清理旧数据
./scripts/pg_cluster_manager.sh init --base-port 新端口 --force

# 方案 2：手动清理数据目录
rm -rf /tmp/pg_clusters  # 或你的 BASE_DATA_DIR
./scripts/pg_cluster_manager.sh init --base-port 新端口
```

### Q2: pg_basebackup 连接失败

**原因**：主节点端口与预期不符

**检查步骤**：
1. 确认数据目录中的 postgresql.conf 配置的端口
2. 确保使用相同的 --base-port 参数
3. 使用 --force 重新初始化

```bash
# 检查配置
cat /tmp/pg_clusters/s1_1/postgresql.conf | grep port

# 重新初始化
./scripts/pg_cluster_manager.sh init --base-port 正确的端口 --force
```

### Q3: 如何测试不同的 WAL 级别？

```bash
# 物理复制（默认）
./scripts/pg_cluster_manager.sh init --wal-level replica --force

# 逻辑复制
./scripts/pg_cluster_manager.sh init --wal-level logical --force

# 验证
psql -h localhost -p 5432 -U postgres -c "SHOW wal_level;"
```

### Q4: 测试时如何避免影响现有集群？

使用不同的端口和数据目录：

```bash
./scripts/pg_cluster_manager.sh init \
    --base-port 10000 \
    --cluster-prefix "test_" \
    --force

# 这样会创建在默认目录，但使用不同端口
# 不会影响端口 5432 的集群
```

## 性能测试

### 异步 vs 同步复制性能对比

```bash
# 配置文件中设置不同的 REPLICATION_MODE
# async, quorum, sync

# 使用 pgbench 测试
pgbench -h localhost -p 5432 -U postgres -i postgres
pgbench -h localhost -p 5432 -U postgres -c 10 -j 2 -t 1000 postgres
```

## 自动化测试

可以将测试集成到 CI/CD 流程：

```bash
#!/bin/bash
set -e

# 运行所有测试
./scripts/test_all_features.sh

# 检查退出码
if [ $? -eq 0 ]; then
    echo "All tests passed"
    exit 0
else
    echo "Tests failed"
    exit 1
fi
```

## 清理所有测试数据

```bash
# 停止所有可能运行的实例
./scripts/pg_cluster_manager.sh stop 2>/dev/null || true

# 清理默认目录
rm -rf /tmp/pg_clusters

# 清理配置文件指定的目录
rm -rf /home/wslu/work/pg/dts/pg_data

# 清理其他测试目录
rm -rf /tmp/pg_*
```

