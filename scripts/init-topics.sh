#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

readonly BOOTSTRAP_SERVER="${BOOTSTRAP_SERVER:-kafka:9092}"
readonly DEFAULT_PARTITIONS="${KAFKA_DEFAULT_PARTITIONS:-1}"
readonly DEFAULT_REPLICATION="${KAFKA_DEFAULT_REPLICATION:-1}"

create_topic() {
  local name=$1
  local partitions=${2:-$DEFAULT_PARTITIONS}
  local replication=${3:-$DEFAULT_REPLICATION}

  if [ -z "$name" ]; then
    echo "Error: Topic name is required"
    return 1
  fi

  echo "Creating topic: $name (partitions: $partitions, replication: $replication)"
  kafka-topics --bootstrap-server "$BOOTSTRAP_SERVER" \
    --create --if-not-exists \
    --topic "$name" \
    --partitions "$partitions" \
    --replication-factor "$replication" || {
    echo "Warning: Failed to create topic $name, it may already exist"
  }
}

if [ -n "${KAFKA_TOPICS:-}" ]; then
  echo "Creating topics from KAFKA_TOPICS environment variable..."
  IFS=',' read -ra TOPIC_LIST <<< "$KAFKA_TOPICS"
  for topic_spec in "${TOPIC_LIST[@]}"; do
    topic_spec=$(echo "$topic_spec" | xargs)

    if [ -z "$topic_spec" ]; then
      continue
    fi

    IFS=':' read -ra TOPIC_PARTS <<< "$topic_spec"
    topic_name="${TOPIC_PARTS[0]}"
    partitions="${TOPIC_PARTS[1]:-$DEFAULT_PARTITIONS}"
    replication="${TOPIC_PARTS[2]:-$DEFAULT_REPLICATION}"

    create_topic "$topic_name" "$partitions" "$replication"
  done
else
  echo "Warning: KAFKA_TOPICS environment variable is not set. No topics will be created."
fi

echo "Done"
