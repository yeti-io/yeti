#!/usr/bin/env python3

import json
from pathlib import Path

import redis


def load_redis_data(redis_host: str = "localhost", redis_port: int = 6379, redis_db: int = 0):
    generated_dir = Path(__file__).parent / "generated"
    redis_file = generated_dir / "redis_data.json"

    if not redis_file.exists():
        print(f"Redis data file not found: {redis_file}")
        print("   Run 'uv run generate_data.py' first to generate data.")
        return

    with open(redis_file) as f:
        data = json.load(f)

    r = redis.Redis(host=redis_host, port=redis_port, db=redis_db, decode_responses=False)

    print(f"Loading data into Redis: {redis_host}:{redis_port}/{redis_db}")

    count = 0
    for key, value in data.items():
        r.set(key, json.dumps(value))
        count += 1

    print(f"Loaded {count} keys into Redis")

    r.close()
    print("Redis data loaded successfully")


if __name__ == "__main__":
    load_redis_data()
