#!/bin/bash

# 启动 DTS 服务器

echo "=== 启动 DTS 服务器 ==="

# 设置日志级别（可选）
LOG_LEVEL=${DTS_LOG_LEVEL:-info}
PORT=${DTS_PORT:-8080}

# 元数据库配置（可选，会覆盖配置文件）
DB_HOST=${DTS_DB_HOST:-}
DB_PORT=${DTS_DB_PORT:-}
DB_USER=${DTS_DB_USER:-}
DB_PASSWORD=${DTS_DB_PASSWORD:-}
DB_NAME=${DTS_DB_NAME:-}
DB_SSLMODE=${DTS_DB_SSLMODE:-}

echo "启动参数:"
echo "  Log Level: $LOG_LEVEL"
echo "  Port: $PORT"
if [ -n "$DB_HOST" ]; then
    echo "  元数据库 Host: $DB_HOST"
fi
if [ -n "$DB_PORT" ]; then
    echo "  元数据库 Port: $DB_PORT"
fi
if [ -n "$DB_NAME" ]; then
    echo "  元数据库 Name: $DB_NAME"
fi
echo ""

# 构建启动命令
CMD="./bin/dts --log-level $LOG_LEVEL --port $PORT"

if [ -n "$DB_HOST" ]; then
    CMD="$CMD --db-host $DB_HOST"
fi
if [ -n "$DB_PORT" ]; then
    CMD="$CMD --db-port $DB_PORT"
fi
if [ -n "$DB_USER" ]; then
    CMD="$CMD --db-user $DB_USER"
fi
if [ -n "$DB_PASSWORD" ]; then
    CMD="$CMD --db-password $DB_PASSWORD"
fi
if [ -n "$DB_NAME" ]; then
    CMD="$CMD --db-name $DB_NAME"
fi
if [ -n "$DB_SSLMODE" ]; then
    CMD="$CMD --db-sslmode $DB_SSLMODE"
fi

echo "执行命令: $CMD"
echo ""

# 启动服务器
$CMD

