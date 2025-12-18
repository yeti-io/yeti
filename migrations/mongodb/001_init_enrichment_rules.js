// MongoDB Migration: Initialize enrichment_rules collection
// Run with: mongosh <database> migrations/mongodb/001_init_enrichment_rules.js

db.createCollection("enrichment_rules", {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["name", "field_to_enrich", "source_type", "enabled"],
      properties: {
        _id: {
          bsonType: "string",
          description: "Rule ID (UUID string)"
        },
        name: {
          bsonType: "string",
          description: "Rule name"
        },
        field_to_enrich: {
          bsonType: "string",
          description: "Field name in message to enrich"
        },
        source_type: {
          enum: ["api", "database", "mongodb", "postgresql", "cache", "redis"],
          description: "Type of data source"
        },
        source_config: {
          bsonType: "object",
          description: "Source configuration (varies by source_type)"
        },
        transformations: {
          bsonType: "array",
          description: "Array of transformation rules",
          items: {
            bsonType: "object",
            required: ["source_path", "target_field"],
            properties: {
              source_path: { bsonType: "string" },
              target_field: { bsonType: "string" },
              transform: { bsonType: "string" },
              default: {}
            }
          }
        },
        cache_ttl_seconds: {
          bsonType: "int",
          minimum: 0,
          description: "Cache TTL in seconds"
        },
        error_handling: {
          enum: ["skip_field", "skip_rule", "fail"],
          description: "Error handling strategy"
        },
        fallback_value: {
          description: "Fallback value (any type)"
        },
        priority: {
          bsonType: "int",
          description: "Rule priority (higher = first)"
        },
        enabled: {
          bsonType: "bool",
          description: "Whether rule is enabled"
        },
        created_at: {
          bsonType: "date",
          description: "Creation timestamp"
        },
        updated_at: {
          bsonType: "date",
          description: "Last update timestamp"
        }
      }
    }
  },
  validationLevel: "moderate",
  validationAction: "error"
});

db.enrichment_rules.createIndex(
  { enabled: 1, priority: -1 },
  { name: "idx_enrichment_rules_enabled_priority" }
);

db.enrichment_rules.createIndex(
  { priority: -1 },
  { name: "idx_enrichment_rules_priority" }
);

db.enrichment_rules.createIndex(
  { updated_at: -1 },
  { name: "idx_enrichment_rules_updated_at" }
);

db.enrichment_rules.createIndex(
  { field_to_enrich: 1 },
  { name: "idx_enrichment_rules_field_to_enrich" }
);

db.enrichment_rules.createIndex(
  { enabled: 1, field_to_enrich: 1, priority: -1 },
  { name: "idx_enrichment_rules_enabled_field_priority" }
);
