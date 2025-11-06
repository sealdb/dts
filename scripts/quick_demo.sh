#!/usr/bin/env bash

################################################################################
# Quick Demo Script - Simple demonstration of key features
################################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANAGER="$SCRIPT_DIR/pg_cluster_manager.sh"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}PostgreSQL Cluster Quick Demo${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Demo 1: Simplest usage - use all defaults
echo -e "${GREEN}Demo 1: Simplest Usage - All Defaults${NC}"
echo -e "${YELLOW}No config file, no parameters needed!${NC}"
echo ""
echo -e "${YELLOW}Command: $MANAGER init --force${NC}"
"$MANAGER" init --force
echo ""

echo -e "${YELLOW}Command: $MANAGER start${NC}"
"$MANAGER" start
sleep 2
echo ""

echo -e "${YELLOW}Command: $MANAGER status${NC}"
"$MANAGER" status
echo ""

echo -e "${YELLOW}Created cluster with defaults:${NC}"
echo "  - Base Port: 5432"
echo "  - WAL Level: replica"
echo "  - Cluster Prefixes: s1_, s2_"
echo "  - Data Dir: /tmp/pg_clusters"
echo ""

echo -e "${YELLOW}Connect to cluster 1 primary:${NC}"
echo "  psql -h localhost -p 5432 -U postgres"
echo ""

read -p "Press Enter to continue to next demo..."
echo ""

echo -e "${YELLOW}Command: $MANAGER stop${NC}"
"$MANAGER" stop
sleep 2
echo ""

# Demo 2: Command line overrides
echo -e "${GREEN}Demo 2: Command Line Parameter Overrides${NC}"
echo ""
echo -e "${YELLOW}Command: $MANAGER init --base-port 6000 --wal-level logical --cluster-prefix \"app_,db_\" --force${NC}"
"$MANAGER" init --base-port 6000 --wal-level logical --cluster-prefix "app_,db_" --force
echo ""

echo -e "${YELLOW}Command: $MANAGER start --base-port 6000${NC}"
"$MANAGER" start --base-port 6000
sleep 2
echo ""

echo -e "${YELLOW}Command: $MANAGER status --base-port 6000${NC}"
"$MANAGER" status --base-port 6000
echo ""

echo -e "${YELLOW}Created cluster with custom settings:${NC}"
echo "  - Base Port: 6000 (overridden)"
echo "  - WAL Level: logical (overridden)"
echo "  - Cluster Prefixes: app_, db_ (overridden)"
echo ""

echo -e "${YELLOW}Directory structure:${NC}"
ls -ld /tmp/pg_clusters/* 2>/dev/null | head -6
echo ""

echo -e "${YELLOW}Connect to cluster 1 primary:${NC}"
echo "  psql -h localhost -p 6000 -U postgres"
echo ""

read -p "Press Enter to clean up..."
echo ""

echo -e "${YELLOW}Command: $MANAGER stop --base-port 6000${NC}"
"$MANAGER" stop --base-port 6000
sleep 2
echo ""

# Cleanup
echo -e "${YELLOW}Cleaning up test data...${NC}"
rm -rf /tmp/pg_clusters
echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Demo Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${YELLOW}Key Takeaways:${NC}"
echo "1. ✅ No config file needed - use built-in defaults"
echo "2. ✅ Easy to override settings via command line"
echo "3. ✅ Same parameters for init/start/stop/status"
echo "4. ✅ --force for clean reinitialization"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo "- Run: $MANAGER --help"
echo "- Read: $SCRIPT_DIR/README.md"
echo "- Test: $SCRIPT_DIR/test_all_features.sh"
echo ""

