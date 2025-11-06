#!/bin/bash

# DTS API 测试脚本

BASE_URL="http://localhost:8080/rdscheduler/api"
TASK_ID="test-task-$(date +%s)"

echo "=== DTS API 测试 ==="
echo "Base URL: $BASE_URL"
echo "Task ID: $TASK_ID"
echo ""

# 1. 创建任务
echo "1. 创建同步任务..."
RESPONSE=$(curl -s -X POST "$BASE_URL/tasks" \
  -H "Content-Type: application/json" \
  -d "{
    \"task_id\": \"$TASK_ID\",
    \"source\": {
      \"domin\": \"127.0.0.1\",
      \"port\": \"5533\",
      \"username\": \"postgres\",
      \"password\": \"postgres\"
    },
    \"dest\": {
      \"domin\": \"127.0.0.1\",
      \"port\": \"5633\",
      \"username\": \"postgres\",
      \"password\": \"postgres\"
    },
    \"tables\": []
  }")

echo "Response: $RESPONSE"
echo ""

# 2. 查询任务状态
echo "2. 查询任务状态..."
sleep 2
STATUS_RESPONSE=$(curl -s -X GET "$BASE_URL/tasks/$TASK_ID")
echo "Status: $STATUS_RESPONSE"
echo ""

# 3. 等待一段时间后再次查询
echo "3. 等待 5 秒后再次查询状态..."
sleep 5
STATUS_RESPONSE=$(curl -s -X GET "$BASE_URL/tasks/$TASK_ID")
echo "Status: $STATUS_RESPONSE"
echo ""

echo "=== 测试完成 ==="
echo "Task ID: $TASK_ID"
echo "可以使用以下命令继续测试："
echo "  curl -X GET $BASE_URL/tasks/$TASK_ID"
echo "  curl -X POST $BASE_URL/tasks/$TASK_ID/switch"
echo "  curl -X DELETE $BASE_URL/tasks/$TASK_ID"

