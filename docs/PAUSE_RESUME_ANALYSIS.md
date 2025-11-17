# PostgreSQL 逻辑复制中的暂停/恢复机制分析

## 问题背景

在使用 PostgreSQL 的发布（Publication）和订阅（Subscription）机制进行数据同步时，是否存在暂停同步、恢复同步的可能性？

## PostgreSQL 逻辑复制机制

### 1. 原生订阅（Subscription）机制

PostgreSQL 提供了原生的逻辑复制订阅功能：

```sql
-- 创建订阅
CREATE SUBSCRIPTION sub_name
CONNECTION 'host=source_host port=5432 user=replicator password=xxx dbname=source_db'
PUBLICATION pub_name;

-- 禁用订阅（停止复制）
ALTER SUBSCRIPTION sub_name DISABLE;

-- 启用订阅（恢复复制）
ALTER SUBSCRIPTION sub_name ENABLE;
```

**关键限制：**
- ✅ **可以暂停**：使用 `ALTER SUBSCRIPTION ... DISABLE` 可以停止复制
- ⚠️ **恢复有风险**：使用 `ALTER SUBSCRIPTION ... ENABLE` 恢复时，如果复制槽（replication slot）中的数据已被清理，可能需要**重新同步**（resync）
- ⚠️ **复制槽增长**：禁用订阅后，如果源库继续产生 WAL，复制槽会继续增长，可能导致磁盘空间问题

### 2. 基于 pglogrepl 的流式复制（当前实现方式）

当前 DTS 系统使用的是基于 `pglogrepl` 库的流式复制，而不是原生的订阅机制。

**工作原理：**
1. 使用 `pglogrepl.StartReplication()` 启动复制流
2. 通过 `ProcessReplicationStream()` 持续接收 WAL 数据
3. 处理消息并发送确认（acknowledgment）

**暂停/恢复的技术可行性：**

#### ✅ **暂停是可行的，但需要正确处理**

**方案 1：停止接收消息（不推荐）**
```go
// 暂停：停止 ProcessReplicationStream 循环
// 问题：复制槽会继续增长，因为源库不知道订阅者已停止
```

**方案 2：继续接收但不应用（推荐）**
```go
// 暂停：继续接收 WAL 消息并发送确认，但不应用变更
// 优点：复制槽不会增长，可以随时恢复
// 实现：在 handler.Handle() 中根据任务状态决定是否应用
```

#### ✅ **恢复是可行的**

恢复时可以从暂停点继续处理，因为：
- WAL 数据已接收并确认
- 复制槽位置已更新
- 只需继续应用后续的 WAL 变更

## 当前实现的问题

查看 `internal/state/syncing_wal.go`，发现 WAL 同步的实现还是 TODO 状态：

```go
// TODO: Start WAL subscriber
// subscriber, err := replication.NewSubscriber(sourceDB.DSN(), slotName)
// ...
```

**当前暂停/恢复的实现：**
- `PauseTask()`: 只是更新任务状态为 `paused`，**没有实际停止 WAL 流**
- `ResumeTask()`: 调用 `StartTask()`，**没有处理 WAL 流的恢复**

## 推荐的实现方案

### 方案 A：应用层暂停（推荐）

**原理：** 继续接收 WAL 消息并确认，但在应用层暂停处理

```go
// 在 ProcessReplicationStream 中
func (s *Subscriber) ProcessReplicationStream(ctx context.Context, taskState *TaskState) error {
    for {
        msg, err := s.conn.ReceiveMessage(ctx)
        if err != nil {
            return err
        }

        // 处理消息
        switch v := msg.(type) {
        case *pgproto3.CopyData:
            if v.Data[0] == pglogrepl.XLogDataByteID {
                // 解析并解码消息
                decodedMsg, err := s.decoder.Decode(logicalMsg)

                // 关键：检查任务状态
                if taskState.IsPaused() {
                    // 暂停状态：只确认，不应用
                    s.sendAcknowledgment(xld)
                    continue
                }

                // 正常状态：应用变更
                if err := s.handler.Handle(ctx, decodedMsg); err != nil {
                    return err
                }
                s.sendAcknowledgment(xld)
            }
        }
    }
}
```

**优点：**
- ✅ 复制槽不会增长
- ✅ 可以随时恢复，无需重新同步
- ✅ 实现简单

**缺点：**
- ⚠️ 需要保持连接和接收消息（消耗少量资源）

### 方案 B：连接层暂停（不推荐）

**原理：** 停止接收消息，但定期发送确认防止复制槽增长

```go
// 暂停：停止 ProcessReplicationStream
// 启动后台 goroutine 定期发送确认
go func() {
    ticker := time.NewTicker(10 * time.Second)
    for range ticker.C {
        if taskState.IsPaused() {
            // 发送确认，但不接收新消息
            s.sendStandbyStatusUpdate(lastLSN)
        }
    }
}()
```

**缺点：**
- ⚠️ 实现复杂
- ⚠️ 可能丢失部分 WAL 数据
- ⚠️ 恢复时需要处理数据一致性

### 方案 C：使用原生订阅（不适合当前架构）

**原理：** 使用 PostgreSQL 的 `ALTER SUBSCRIPTION DISABLE/ENABLE`

**缺点：**
- ⚠️ 需要目标库支持订阅（当前架构是应用层处理）
- ⚠️ 恢复时可能需要重新同步
- ⚠️ 不符合当前的设计架构

## 建议的实现

### 1. 修改 WAL 处理逻辑

在 `internal/replication/subscriber.go` 中：

```go
type Subscriber struct {
    conn     *pgconn.PgConn
    decoder  *wal.Decoder
    handler  *wal.Handler
    slotName string
    paused   atomic.Bool  // 添加暂停标志
}

// SetPaused sets the paused state
func (s *Subscriber) SetPaused(paused bool) {
    s.paused.Store(paused)
}

// ProcessReplicationStream processes replication stream
func (s *Subscriber) ProcessReplicationStream(ctx context.Context) error {
    for {
        msg, err := s.conn.ReceiveMessage(ctx)
        if err != nil {
            return fmt.Errorf("failed to receive message: %w", err)
        }

        switch v := msg.(type) {
        case *pgproto3.CopyData:
            if err := s.handleCopyData(ctx, v); err != nil {
                return err
            }
        // ...
        }
    }
}

func (s *Subscriber) handleCopyData(ctx context.Context, msg *pgproto3.CopyData) error {
    switch msg.Data[0] {
    case pglogrepl.XLogDataByteID:
        xld, err := pglogrepl.ParseXLogData(msg.Data[1:])
        if err != nil {
            return err
        }

        logicalMsg, err := pglogrepl.Parse(xld.WALData)
        if err != nil {
            return err
        }

        decodedMsg, err := s.decoder.Decode(logicalMsg)
        if err != nil {
            return err
        }

        // 关键：检查是否暂停
        if !s.paused.Load() {
            // 正常状态：处理消息
            if err := s.handler.Handle(ctx, decodedMsg); err != nil {
                return fmt.Errorf("failed to handle message: %w", err)
            }
        }
        // 无论是否暂停，都要发送确认，防止复制槽增长
        err = pglogrepl.SendStandbyStatusUpdate(
            ctx,
            s.conn,
            pglogrepl.StandbyStatusUpdate{
                WALWritePosition: xld.WALStart + pglogrepl.LSN(len(xld.WALData)),
            },
        )
        if err != nil {
            return fmt.Errorf("failed to send status update: %w", err)
        }
    }
    return nil
}
```

### 2. 修改服务层

在 `internal/service/migration.go` 中：

```go
// 需要维护每个任务的 subscriber 实例
type MigrationService struct {
    // ...
    subscribers map[string]*replication.Subscriber  // taskID -> subscriber
    subscribersMu sync.RWMutex
}

func (s *MigrationService) PauseTask(id string) error {
    // ... 现有逻辑 ...

    // 暂停 WAL 流处理
    s.subscribersMu.RLock()
    subscriber, ok := s.subscribers[id]
    s.subscribersMu.RUnlock()

    if ok && subscriber != nil {
        subscriber.SetPaused(true)
    }

    return s.taskRepo.UpdateState(id, model.StatePaused, "")
}

func (s *MigrationService) ResumeTask(ctx context.Context, id string) error {
    // ... 现有逻辑 ...

    // 恢复 WAL 流处理
    s.subscribersMu.RLock()
    subscriber, ok := s.subscribers[id]
    s.subscribersMu.RUnlock()

    if ok && subscriber != nil {
        subscriber.SetPaused(false)
    }

    return s.StartTask(ctx, id)
}
```

## 总结

### 回答：是否存在暂停/恢复的可能？

**答案：✅ 是的，技术上完全可行**

1. **暂停是可行的**：
   - 继续接收 WAL 消息并确认（防止复制槽增长）
   - 在应用层暂停处理变更
   - 任务状态更新为 `paused`

2. **恢复是可行的**：
   - 从暂停点继续处理 WAL 变更
   - 无需重新同步
   - 数据一致性有保障

3. **当前实现的问题**：
   - WAL 同步逻辑还未完全实现（TODO 状态）
   - 暂停/恢复没有实际控制 WAL 流
   - 需要完善实现

### 建议

1. **优先实现方案 A（应用层暂停）**
2. **在 `SyncingWALState` 中完善 WAL 订阅逻辑**
3. **在服务层维护 subscriber 实例，支持暂停/恢复控制**
4. **添加复制槽监控，防止空间问题**

