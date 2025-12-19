#!/bin/bash

set -e

uv run generate_data.py
uv run load_mongodb.py
uv run load_redis.py

read -p "Run producer now? (y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo ""
    echo "============================================================"
    echo "Sending messages to Kafka"
    echo "============================================================"
    uv run producer.py
else
    echo ""
    echo "You can run producer later with: uv run producer.py"
fi
