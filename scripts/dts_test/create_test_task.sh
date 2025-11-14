#!/bin/bash

# 创建测试任务的脚本

BASE_URL="http://localhost:8080/rdscheduler/api"
TASK_ID="test-task-$(date +%s)"

SOURCE_HOST="127.0.0.1"
SOURCE_PORT="5533"
SOURCE_USER="postgres"
SOURCE_PASS="postgres"
SOURCE_DB="postgres"

DEST_HOST="127.0.0.1"
DEST_PORT="5633"
DEST_USER="postgres"
DEST_PASS="postgres"
DEST_DB="postgres"

echo "=== 创建测试任务 ==="
echo "Task ID: $TASK_ID"
echo "Source: $SOURCE_HOST:$SOURCE_PORT"
echo "Dest: $DEST_HOST:$DEST_PORT"
echo ""

# 创建任务（不指定表，自动获取所有表）
echo "创建任务..."
RESPONSE=$(curl -s -X POST "$BASE_URL/tasks" \
  -H "Content-Type: application/json" \
  -d "{
    \"task_id\": \"$TASK_ID\",
    \"source\": {
      \"domin\": \"$SOURCE_HOST\",
      \"port\": \"$SOURCE_PORT\",
      \"username\": \"$SOURCE_USER\",
      \"password\": \"$SOURCE_PASS\",
      \"database\": \"$SOURCE_DB\"
    },
    \"dest\": {
      \"domin\": \"$DEST_HOST\",
      \"port\": \"$DEST_PORT\",
      \"username\": \"$DEST_USER\",
      \"password\": \"$DEST_PASS\",
      \"database\": \"$DEST_DB\"
    }
  }")

echo "Response:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

if echo "$RESPONSE" | grep -q '"state":"OK"'; then
    echo "✓ 任务创建成功！"
    echo ""
    echo "Task ID: $TASK_ID"
    echo ""
    echo "可以使用以下命令："
    echo "  查询状态: curl -X GET $BASE_URL/tasks/$TASK_ID | jq"
    echo "  触发切流: curl -X POST $BASE_URL/tasks/$TASK_ID/switch | jq"
    echo "  删除任务: curl -X DELETE $BASE_URL/tasks/$TASK_ID | jq"
else
    echo "✗ 任务创建失败"
    exit 1
fi

