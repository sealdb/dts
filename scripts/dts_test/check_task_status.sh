#!/bin/bash

# 查询任务状态脚本

TASK_ID="${1:-test-task-001}"
BASE_URL="http://localhost:8080/dts/api"

echo "=== 查询任务状态 ==="
echo "Task ID: $TASK_ID"
echo ""

RESPONSE=$(curl -s -X GET "$BASE_URL/tasks/$TASK_ID")

echo "Response:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

# 解析状态
STAGE=$(echo "$RESPONSE" | jq -r '.stage' 2>/dev/null)
STATE=$(echo "$RESPONSE" | jq -r '.state' 2>/dev/null)
DURATION=$(echo "$RESPONSE" | jq -r '.duration' 2>/dev/null)
DELAY=$(echo "$RESPONSE" | jq -r '.delay' 2>/dev/null)

if [ "$STATE" = "OK" ]; then
    echo "状态: $STAGE"
    if [ "$DURATION" != "-1" ] && [ "$DURATION" != "null" ]; then
        echo "耗时: ${DURATION}ms"
    fi
    if [ "$DELAY" != "-1" ] && [ "$DELAY" != "null" ]; then
        echo "延迟: ${DELAY}ms"
    fi
else
    MESSAGE=$(echo "$RESPONSE" | jq -r '.message' 2>/dev/null)
    echo "错误: $MESSAGE"
fi





