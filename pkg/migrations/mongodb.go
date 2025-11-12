package migrations

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func EnsureMongoCollection(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("enrichment_rules")

	collections, err := db.ListCollectionNames(ctx, map[string]interface{}{"name": "enrichment_rules"})
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	collectionExists := false
	for _, name := range collections {
		if name == "enrichment_rules" {
			collectionExists = true
			break
		}
	}

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "enabled", Value: 1}, {Key: "priority", Value: -1}},
			Options: options.Index().SetName("idx_enrichment_rules_enabled_priority"),
		},
		{
			Keys:    bson.D{{Key: "priority", Value: -1}},
			Options: options.Index().SetName("idx_enrichment_rules_priority"),
		},
		{
			Keys:    bson.D{{Key: "updated_at", Value: -1}},
			Options: options.Index().SetName("idx_enrichment_rules_updated_at"),
		},
		{
			Keys:    bson.D{{Key: "field_to_enrich", Value: 1}},
			Options: options.Index().SetName("idx_enrichment_rules_field_to_enrich"),
		},
		{
			Keys:    bson.D{{Key: "enabled", Value: 1}, {Key: "field_to_enrich", Value: 1}, {Key: "priority", Value: -1}},
			Options: options.Index().SetName("idx_enrichment_rules_enabled_field_priority"),
		},
	}

	_, err = collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create indexes: %w", err)
		}
	}

	if !collectionExists {
		// Collection will be created automatically on first insert
		// But we can create it explicitly if needed
		// For now, just log that indexes are created
	}

	return nil
}
