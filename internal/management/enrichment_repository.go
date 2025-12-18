package management

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type EnrichmentRepository interface {
	CreateEnrichmentRule(ctx context.Context, rule *EnrichmentRule) error
	ListEnrichmentRules(ctx context.Context) ([]EnrichmentRule, error)
	GetEnrichmentRule(ctx context.Context, id string) (*EnrichmentRule, error)
	UpdateEnrichmentRule(ctx context.Context, rule *EnrichmentRule) error
	DeleteEnrichmentRule(ctx context.Context, id string) error
}

type mongoEnrichmentRepository struct {
	collection *mongo.Collection
}

func NewEnrichmentRepository(db *mongo.Database) EnrichmentRepository {
	return &mongoEnrichmentRepository{
		collection: db.Collection("enrichment_rules"),
	}
}

func (r *mongoEnrichmentRepository) CreateEnrichmentRule(ctx context.Context, rule *EnrichmentRule) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	_, err := r.collection.InsertOne(ctx, rule)
	if err != nil {
		return fmt.Errorf("failed to create enrichment rule: %w", err)
	}

	return nil
}

func (r *mongoEnrichmentRepository) GetEnrichmentRule(ctx context.Context, id string) (*EnrichmentRule, error) {
	filter := bson.M{"_id": id}

	var rule EnrichmentRule
	err := r.collection.FindOne(ctx, filter).Decode(&rule)
	if err == mongo.ErrNoDocuments {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get enrichment rule: %w", err)
	}

	return &rule, nil
}

func (r *mongoEnrichmentRepository) ListEnrichmentRules(ctx context.Context) ([]EnrichmentRule, error) {
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: -1}, {Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list enrichment rules: %w", err)
	}
	defer cursor.Close(ctx)

	var rules []EnrichmentRule
	if err := cursor.All(ctx, &rules); err != nil {
		return nil, fmt.Errorf("failed to decode enrichment rules: %w", err)
	}

	return rules, nil
}

func (r *mongoEnrichmentRepository) UpdateEnrichmentRule(ctx context.Context, rule *EnrichmentRule) error {
	rule.UpdatedAt = time.Now()

	filter := bson.M{"_id": rule.ID}
	update := bson.M{"$set": rule}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update enrichment rule: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("enrichment rule not found")
	}

	return nil
}

func (r *mongoEnrichmentRepository) DeleteEnrichmentRule(ctx context.Context, id string) error {
	filter := bson.M{"_id": id}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete enrichment rule: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("enrichment rule not found")
	}

	return nil
}
