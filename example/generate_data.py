#!/usr/bin/env python3

import json
import random
import string
from datetime import datetime, timezone, timedelta
from pathlib import Path


def generate_user_id() -> str:
    return f"user-{random.randint(1, 9999)}"


def generate_product_id() -> str:
    letters = ''.join(random.choices(string.ascii_lowercase, k=random.randint(3, 6)))
    return f"product-{letters}"


def generate_order_id() -> str:
    return f"order-{random.randint(1, 9999):03d}"


def generate_message_id() -> str:
    return f"payment-order-{random.randint(1, 9999):03d}"


def generate_amount() -> float:
    return round(random.uniform(10.0, 10000.0), 2)


def generate_status() -> str:
    return random.choice(["completed", "active", "pending", "inactive", "processing"])


def generate_currency() -> str:
    return "USD"


def generate_created_at() -> str:
    base_date = datetime.now(timezone.utc)

    days_offset = random.randint(-90, 90)
    hours_offset = random.randint(0, 23)
    minutes_offset = random.randint(0, 59)
    dt = base_date + timedelta(days=days_offset, hours=hours_offset, minutes=minutes_offset)
    return dt.isoformat().replace("+00:00", "Z")


def generate_payment_message() -> dict:
    has_product = random.choice([True, False])
    payload = {
        "order_id": generate_order_id(),
        "user_id": generate_user_id(),
        "amount": generate_amount(),
        "currency": generate_currency(),
        "status": generate_status(),
        "created_at": generate_created_at(),
    }
    if has_product:
        payload["product_id"] = generate_product_id()

    return {
        "id": generate_message_id(),
        "source": "payment-service",
        "payload": payload,
    }


def generate_user_data(user_id: str) -> dict:
    name_length = random.randint(5, 20)
    name = ''.join(random.choices(string.ascii_lowercase + ' ', k=name_length)).title().strip()
    email_prefix_length = random.randint(5, 10)
    email_prefix = ''.join(random.choices(string.ascii_lowercase, k=email_prefix_length))
    tier = random.choice(["basic", "premium", "enterprise"])

    return {
        "_id": user_id,
        "name": name,
        "email": f"{email_prefix}@example.com",
        "tier": tier,
    }


def generate_product_data(product_id: str) -> dict:
    name_length = random.randint(5, 30)
    name = ''.join(random.choices(string.ascii_lowercase + ' ', k=name_length)).title().strip()
    price = round(random.uniform(10.0, 1000.0), 2)
    category = random.choice(["electronics", "clothing", "food", "books", "general"])

    return {
        "_id": product_id,
        "name": name,
        "price": price,
        "category": category,
    }


def generate_order_data(order_id: str) -> dict:
    discount_tier = random.choice([None, "standard", "gold", "premium"])
    shipping_method = random.choice([None, "standard", "express", "premium"])
    shipping_cost = round(random.uniform(0.0, 50.0), 2) if shipping_method == "express" else None
    estimated_delivery = random.randint(1, 14)

    return {
        "discount_tier": discount_tier,
        "shipping_method": shipping_method,
        "shipping_cost": shipping_cost,
        "estimated_delivery_days": estimated_delivery,
    }


def generate_messages(count: int = 20) -> list[dict]:
    messages = []
    for _ in range(count):
        messages.append(generate_payment_message())
    return messages


def generate_mongodb_data(messages: list[dict]) -> dict:
    user_ids = set()
    product_ids = set()

    for msg in messages:
        payload = msg["payload"]
        if "user_id" in payload:
            user_ids.add(payload["user_id"])
        if "product_id" in payload:
            product_ids.add(payload["product_id"])

    users = [generate_user_data(uid) for uid in user_ids]
    products = [generate_product_data(pid) for pid in product_ids]

    return {
        "users": users,
        "products": products,
    }


def generate_redis_order_data(messages: list[dict]) -> dict:
    order_ids = set()
    for msg in messages:
        payload = msg["payload"]
        if "order_id" in payload:
            order_ids.add(payload["order_id"])

    order_data = {}
    for order_id in order_ids:
        order_data[f"order:{order_id}"] = generate_order_data(order_id)

    return order_data


def main():
    generated_dir = Path(__file__).parent / "generated"
    generated_dir.mkdir(exist_ok=True)

    print("Generating messages...")
    messages = generate_messages(count=100_000)
    messages_file = generated_dir / "messages.json"
    with open(messages_file, "w") as f:
        json.dump(messages, f, indent=2)
    print(f"Generated {len(messages)} messages -> {messages_file}")

    print("\nGenerating MongoDB data...")
    mongodb_data = generate_mongodb_data(messages)
    mongodb_file = generated_dir / "mongodb_data.json"
    with open(mongodb_file, "w") as f:
        json.dump(mongodb_data, f, indent=2)
    print(f"Generated {len(mongodb_data['users'])} users and {len(mongodb_data['products'])} products -> {mongodb_file}")

    print("\nGenerating Redis data...")
    redis_data = {}
    for user in mongodb_data["users"]:
        redis_data[f"user:{user['_id']}"] = {
            "name": user["name"],
            "email": user["email"],
            "tier": user["tier"],
        }
    for product in mongodb_data["products"]:
        redis_data[f"product:{product['_id']}"] = {
            "name": product["name"],
            "price": product["price"],
            "category": product["category"],
        }
    order_redis_data = generate_redis_order_data(messages)
    redis_data.update(order_redis_data)
    redis_file = generated_dir / "redis_data.json"
    with open(redis_file, "w") as f:
        json.dump(redis_data, f, indent=2)
    print(f"Generated {len(redis_data)} Redis keys ({len(order_redis_data)} orders) -> {redis_file}")

    print("\nData generation completed!")


if __name__ == "__main__":
    main()
