#!/usr/bin/env python3

import json
from typing import Any

from kafka import KafkaConsumer


class MessageConsumer:
    def __init__(
        self,
        broker: str = "localhost:29092",
        topic: str = "processed_events",
        group_id: str = "demo-consumer",
    ):
        self.consumer = KafkaConsumer(
            topic,
            bootstrap_servers=[broker],
            group_id=group_id,
            value_deserializer=lambda m: json.loads(m.decode("utf-8")),
            key_deserializer=lambda k: k.decode("utf-8") if k else None,
            auto_offset_reset="latest",
            enable_auto_commit=True,
        )
        self.topic = topic

    def consume(self, timeout_ms: int = 1000) -> None:
        print(f"Consuming from topic: {self.topic}")
        print("Waiting for messages...\n")

        try:
            for message in self.consumer:
                self._print_message(message.value)
        except KeyboardInterrupt:
            print("\nStopping consumer...")
        finally:
            self.consumer.close()

    def _print_message(self, envelope: dict[str, Any]) -> None:
        print(f"Message ID: {envelope['id']}")
        print(f"Source: {envelope['source']}")
        print(f"Payload: {envelope['payload']}")

        metadata = envelope.get("metadata", {})
        if filters_applied := metadata.get("filters_applied"):
            rule_ids = filters_applied.get("rule_ids", [])
            print(f"Applied rules: {rule_ids}")

        if dedup := metadata.get("deduplication"):
            is_unique = dedup.get("is_unique", False)
            print(f"Deduplication: {'unique' if is_unique else 'duplicate'}")

        if enrichment := metadata.get("enrichment"):
            print(f"Enrichment: {enrichment}")

        print("-" * 50)


if __name__ == "__main__":
    consumer = MessageConsumer()
    consumer.consume()