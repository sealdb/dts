#!/bin/bash

# DTS API Test Script
# Usage: ./scripts/test_api.sh [BASE_URL]
# Example: ./scripts/test_api.sh http://localhost:8080

BASE_URL="${1:-http://localhost:8080}"
API_BASE="${BASE_URL}/dts/api/tasks"

echo "=========================================="
echo "DTS API Test Script"
echo "Base URL: ${BASE_URL}"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test task ID (will be set after creating a task)
TASK_ID="test-task-$(date +%s)"

echo -e "${BLUE}1. Create Task (POST /dts/api/tasks)${NC}"
echo "----------------------------------------"
CREATE_RESPONSE=$(curl -s -X POST "${API_BASE}" \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "'${TASK_ID}'",
    "source": {
      "domin": "localhost",
      "port": "5432",
      "username": "postgres",
      "password": "postgres",
      "database": "source_db"
    },
    "dest": {
      "domin": "localhost",
      "port": "5433",
      "username": "postgres",
      "password": "postgres",
      "database": "target_db"
    },
    "tables": ["users", "orders"]
  }')

echo "$CREATE_RESPONSE" | jq '.' 2>/dev/null || echo "$CREATE_RESPONSE"
echo ""

# Extract task_id from response if needed (assuming it's returned)
# For now, we'll use the one we set
echo -e "${GREEN}Task ID: ${TASK_ID}${NC}"
echo ""

echo -e "${BLUE}2. Get Task Status (GET /dts/api/tasks/:task_id/status)${NC}"
echo "----------------------------------------"
curl -s -X GET "${API_BASE}/${TASK_ID}/status" | jq '.' 2>/dev/null || curl -s -X GET "${API_BASE}/${TASK_ID}/status"
echo ""
echo ""

echo -e "${BLUE}3. Start Task (POST /dts/api/tasks/:task_id/start)${NC}"
echo "----------------------------------------"
curl -s -X POST "${API_BASE}/${TASK_ID}/start" | jq '.' 2>/dev/null || curl -s -X POST "${API_BASE}/${TASK_ID}/start"
echo ""
echo ""

echo -e "${BLUE}4. Get Task Status Again${NC}"
echo "----------------------------------------"
curl -s -X GET "${API_BASE}/${TASK_ID}/status" | jq '.' 2>/dev/null || curl -s -X GET "${API_BASE}/${TASK_ID}/status"
echo ""
echo ""

echo -e "${BLUE}5. Pause Task (POST /dts/api/tasks/:task_id/pause)${NC}"
echo "----------------------------------------"
curl -s -X POST "${API_BASE}/${TASK_ID}/pause" | jq '.' 2>/dev/null || curl -s -X POST "${API_BASE}/${TASK_ID}/pause"
echo ""
echo ""

echo -e "${BLUE}6. Resume Task (POST /dts/api/tasks/:task_id/resume)${NC}"
echo "----------------------------------------"
curl -s -X POST "${API_BASE}/${TASK_ID}/resume" | jq '.' 2>/dev/null || curl -s -X POST "${API_BASE}/${TASK_ID}/resume"
echo ""
echo ""

echo -e "${BLUE}7. Stop Task (POST /dts/api/tasks/:task_id/stop)${NC}"
echo "----------------------------------------"
curl -s -X POST "${API_BASE}/${TASK_ID}/stop" | jq '.' 2>/dev/null || curl -s -X POST "${API_BASE}/${TASK_ID}/stop"
echo ""
echo ""

echo -e "${BLUE}8. Switch Task (POST /dts/api/tasks/:task_id/switch)${NC}"
echo "----------------------------------------"
curl -s -X POST "${API_BASE}/${TASK_ID}/switch" | jq '.' 2>/dev/null || curl -s -X POST "${API_BASE}/${TASK_ID}/switch"
echo ""
echo ""

echo -e "${BLUE}9. Delete Task (DELETE /dts/api/tasks/:task_id)${NC}"
echo "----------------------------------------"
curl -s -X DELETE "${API_BASE}/${TASK_ID}" | jq '.' 2>/dev/null || curl -s -X DELETE "${API_BASE}/${TASK_ID}"
echo ""
echo ""

echo -e "${GREEN}All API tests completed!${NC}"

