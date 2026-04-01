#!/bin/bash

export DATABASE_DSN="host=127.0.0.1 port=54321 user=system password=123456 dbname=test sslmode=disable"
export KINGBASE_DSN="$DATABASE_DSN"

echo "=== Inside script ==="
echo "DATABASE_DSN = [$DATABASE_DSN]"
echo "KINGBASE_DSN = [$KINGBASE_DSN]"

# 额外打印给用户提示
echo ""
echo "请使用 source ./scripts/dev/export.sh 来加载这些变量"