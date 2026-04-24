#!/bin/bash

PID=$(pgrep -f "rerank_server_bge.py")

if [ -n "$PID" ]; then
    echo "rerank 服务已在运行 (PID: $PID)，跳过启动。"
    exit 0
fi

echo "启动 rerank 服务..."
uv run python-scripts/rerank_server_bge.py > mac-rerank.log 2>&1 &
echo "rerank 服务已启动 (PID: $!)"
