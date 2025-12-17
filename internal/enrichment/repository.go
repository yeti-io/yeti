package enrichment

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository interface {
	GetActiveRules(ctx context.Context) ([]Rule, error)
}

type MongoDBRepository struct {
	collection *mongo.Collection
}

func NewRepository(db *mongo.Database) Repository {
	return &MongoDBRepository{
		collection: db.Collection("enrichment_rules"),
	}
}

func (r *MongoDBRepository) GetActiveRules(ctx context.Context) ([]Rule, error) {
	filter := bson.M{"enabled": true}
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find rules: %w", err)
	}
	defer cursor.Close(ctx)

	var rules []Rule
	if err := cursor.All(ctx, &rules); err != nil {
		return nil, fmt.Errorf("failed to decode rules: %w", err)
	}

	return rules, nil
}
