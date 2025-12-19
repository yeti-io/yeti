#!/usr/bin/env python3

import json
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import requests
from kafka import KafkaProducer


class RuleManager:
    def __init__(self, base_url: str = "http://localhost:8084"):
        self.base_url = base_url

    def list_filtering_rules(self) -> list[dict]:
        url = f"{self.base_url}/api/v1/rules/filtering"
        try:
            response = requests.get(url)
            response.raise_for_status()
            result = response.json()
            return result if isinstance(result, list) else []
        except requests.RequestException:
            return []

    def get_filtering_rule_by_name(self, name: str) -> dict | None:
        rules = self.list_filtering_rules()
        if not rules:
            return None
        for rule in rules:
            if rule.get("name") == name:
                return rule
        return None

    def delete_filtering_rule(self, rule_id: str) -> None:
        url = f"{self.base_url}/api/v1/rules/filtering/{rule_id}"
        response = requests.delete(url)
        response.raise_for_status()

    def create_filtering_rule(self, name: str, expression: str, priority: int = 10) -> str:
        existing = self.get_filtering_rule_by_name(name)
        if existing:
            self.delete_filtering_rule(existing["id"])

        url = f"{self.base_url}/api/v1/rules/filtering"
        payload = {
            "name": name,
            "expression": expression,
            "priority": priority,
            "enabled": True,
        }
        response = requests.post(url, json=payload)
        response.raise_for_status()
        return response.json()["id"]

    def update_filtering_rule(self, rule_id: str, expression: str) -> None:
        url = f"{self.base_url}/api/v1/rules/filtering/{rule_id}"
        payload = {"expression": expression}
        response = requests.put(url, json=payload)
        response.raise_for_status()

    def list_enrichment_rules(self) -> list[dict]:
        url = f"{self.base_url}/api/v1/rules/enrichment"
        try:
            response = requests.get(url)
            response.raise_for_status()
            result = response.json()
            return result if isinstance(result, list) else []
        except requests.RequestException:
            return []

    def get_enrichment_rule_by_name(self, name: str) -> dict | None:
        rules = self.list_enrichment_rules()
        if not rules:
            return None
        for rule in rules:
            if rule.get("name") == name:
                return rule
        return None

    def delete_enrichment_rule(self, rule_id: str) -> None:
        url = f"{self.base_url}/api/v1/rules/enrichment/{rule_id}"
        response = requests.delete(url)
        response.raise_for_status()

    def create_enrichment_rule(self, rule: dict[str, Any]) -> str:
        existing = self.get_enrichment_rule_by_name(rule["name"])
        if existing:
            self.delete_enrichment_rule(existing["id"])

        url = f"{self.base_url}/api/v1/rules/enrichment"
        payload = {
            "name": rule["name"],
            "field_to_enrich": rule["field_to_enrich"],
            "source_type": rule["source_type"],
            "source_config": rule["source_config"],
            "transformations": rule.get("transformations", []),
            "cache_ttl_seconds": rule.get("cache_ttl_seconds", 0),
            "error_handling": rule.get("error_handling", "skip_rule"),
            "priority": rule.get("priority", 10),
            "enabled": rule.get("enabled", True),
        }
        response = requests.post(url, json=payload)
        if not response.ok:
            print(f"Error creating rule '{rule['name']}': {response.status_code}")
            try:
                error_detail = response.json()
                print(f"  Error response: {json.dumps(error_detail, indent=2)}")
                if "details" in error_detail and isinstance(error_detail["details"], dict):
                    if "message" in error_detail["details"]:
                        print(f"  Validation message: {error_detail['details']['message']}")
                if "message" in error_detail:
                    print(f"  Error message: {error_detail['message']}")
            except Exception as e:
                print(f"  Response text: {response.text}")
                print(f"  Failed to parse JSON: {e}")
            print(f"  Payload sent: {json.dumps(payload, indent=2)}")
            response.raise_for_status()
        return response.json()["id"]

    def update_deduplication_config(self, config: dict[str, Any]) -> None:
        url = f"{self.base_url}/api/v1/config/deduplication"
        payload = {
            "hash_algorithm": config.get("hash_algorithm"),
            "ttl_seconds": config.get("ttl_seconds"),
            "on_redis_error": config.get("on_redis_error"),
            "fields_to_hash": config.get("fields_to_hash", []),
        }
        response = requests.put(url, json=payload)
        response.raise_for_status()


class MessageProducer:
    def __init__(self, broker: str = "localhost:29092", topic: str = "input_events"):
        self.producer = KafkaProducer(
            bootstrap_servers=[broker],
            value_serializer=lambda v: json.dumps(v).encode("utf-8"),
            key_serializer=lambda k: k.encode("utf-8") if k else None,
        )
        self.topic = topic

    def send_envelope(self, envelope: dict[str, Any]) -> None:
        if "timestamp" not in envelope:
            envelope["timestamp"] = datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")
        if "metadata" not in envelope:
            envelope["metadata"] = {}
        self.producer.send(self.topic, key=envelope["id"], value=envelope)
        self.producer.flush()

    def close(self) -> None:
        self.producer.close()


def load_fixtures() -> tuple[list[dict], list[dict], dict, list[dict]]:
    fixtures_dir = Path(__file__).parent / "fixtures"
    generated_dir = Path(__file__).parent / "generated"

    with open(fixtures_dir / "filtering.json") as f:
        filtering_rules = json.load(f)
    with open(fixtures_dir / "enrichment.json") as f:
        enrichment_rules = json.load(f)
    with open(fixtures_dir / "deduplication.json") as f:
        dedup_config = json.load(f)

    messages_file = generated_dir / "messages.json"
    if messages_file.exists():
        with open(messages_file) as f:
            messages = json.load(f)
    else:
        with open(fixtures_dir / "messages.json") as f:
            messages = json.load(f)

    return filtering_rules, enrichment_rules, dedup_config, messages


def main():
    filtering_rules, enrichment_rules, dedup_config, messages = load_fixtures()
    rule_manager = RuleManager()
    message_producer = MessageProducer()

    print("=== Setting up filtering rules ===")
    rule_ids = []
    for rule in filtering_rules:
        rule_id = rule_manager.create_filtering_rule(
            name=rule["name"],
            expression=rule["expression"],
            priority=rule["priority"],
        )
        rule_ids.append(rule_id)
        print(f"  Created: {rule['name']} (ID: {rule_id})")

    if enrichment_rules:
        print("\n=== Setting up enrichment rules ===")
        for rule in enrichment_rules:
            rule_id = rule_manager.create_enrichment_rule(rule)
            print(f"  Created: {rule['name']} (ID: {rule_id})")

    print("\n=== Setting up deduplication config ===")
    rule_manager.update_deduplication_config(dedup_config)
    print(f"  Configured: hash={dedup_config['hash_algorithm']}, TTL={dedup_config['ttl_seconds']}s")

    print("\n=== Waiting for rules to propagate ===")
    time.sleep(5)

    print("\n=== Sending payment-service messages ===")
    print(f"Sending {len(messages)} generated messages...")
    for i, msg in enumerate(messages, 1):
        message_producer.send_envelope(msg)
        payload = msg["payload"]
        product_info = f", product_id: {payload.get('product_id')}" if payload.get("product_id") else ""
        print(f"  [{i}/{len(messages)}] Sent: {msg['id']} (order_id: {payload['order_id']}, amount: {payload['amount']}, status: {payload['status']}{product_info})")

    message_producer.close()
    print("\n=== Demo completed ===")
    print("Check consumer output for processed messages.")


if __name__ == "__main__":
    main()
