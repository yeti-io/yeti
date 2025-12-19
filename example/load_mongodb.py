#!/usr/bin/env python3

import json
from pathlib import Path

from pymongo import MongoClient


def load_mongodb_data(mongodb_uri: str = "mongodb://admin:password@localhost:27017", database: str = "test_db"):
    generated_dir = Path(__file__).parent / "generated"
    mongodb_file = generated_dir / "mongodb_data.json"

    if not mongodb_file.exists():
        print(f"MongoDB data file not found: {mongodb_file}")
        print("   Run 'uv run generate_data.py' first to generate data.")
        return

    with open(mongodb_file) as f:
        data = json.load(f)

    try:
        client = MongoClient(mongodb_uri)
        db = client[database]

        print(f"Loading data into MongoDB: {mongodb_uri}/{database}")

        if "users" in data and data["users"]:
            users_collection = db["user_profiles"]
            for user in data["users"]:
                users_collection.replace_one({"_id": user["_id"]}, user, upsert=True)
            print(f"  Loaded {len(data['users'])} users into user_profiles collection")

        if "products" in data and data["products"]:
            products_collection = db["products"]
            for product in data["products"]:
                products_collection.replace_one({"_id": product["_id"]}, product, upsert=True)
            print(f"  Loaded {len(data['products'])} products into products collection")

        client.close()
        print("MongoDB data loaded successfully")
    except Exception as e:
        print(f"Error loading MongoDB data: {e}")
        print("  Make sure MongoDB is running and accessible")
        print("  If MongoDB requires authentication, update mongodb_uri in load_mongodb.py")


if __name__ == "__main__":
    load_mongodb_data()
