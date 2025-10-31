package provider

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBProvider struct {
	client *mongo.Client
}

func NewMongoDBProvider(client *mongo.Client) *MongoDBProvider {
	return &MongoDBProvider{
		client: client,
	}
}

func (p *MongoDBProvider) Fetch(ctx context.Context, config SourceConfig, fieldValue interface{}) (map[string]interface{}, error) {
	if config.Database == "" || config.Collection == "" {
		return nil, fmt.Errorf("database and collection are required for MongoDB provider")
	}

	db := p.client.Database(config.Database)
	collection := db.Collection(config.Collection)

	filter := bson.M{}

	queryMap := getQueryMap(config)

	if queryMap != nil && len(queryMap) > 0 {
		for k, v := range queryMap {
			if strVal, ok := v.(string); ok {
				strVal = strings.ReplaceAll(strVal, "{field_value}", fmt.Sprintf("%v", fieldValue))
				strVal = strings.ReplaceAll(strVal, "{value}", fmt.Sprintf("%v", fieldValue))
				filter[k] = strVal
			} else {
				filter[k] = v
			}
		}
	} else if config.Field != "" {
		filter[config.Field] = fieldValue
	} else {
		filter["_id"] = fieldValue
	}

	var result bson.M
	err := collection.FindOne(ctx, filter, options.FindOne()).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("document not found")
	}
	if err != nil {
		return nil, fmt.Errorf("mongodb query failed: %w", err)
	}

	resultMap := make(map[string]interface{}, len(result))
	for key, value := range result {
		resultMap[key] = value
	}

	return resultMap, nil
}

func getQueryMap(config SourceConfig) map[string]interface{} {
	if config.Query != nil {
		return config.Query.ToMap()
	}
	return nil
}

type PostgreSQLProvider struct {
	db *sql.DB
}

func NewPostgreSQLProvider(db *sql.DB) *PostgreSQLProvider {
	return &PostgreSQLProvider{
		db: db,
	}
}

func (p *PostgreSQLProvider) Fetch(ctx context.Context, config SourceConfig, fieldValue interface{}) (map[string]interface{}, error) {
	if config.Collection == "" {
		return nil, fmt.Errorf("collection (table name) is required for PostgreSQL provider")
	}

	tableName := config.Collection

	var whereClause string
	var args []interface{}

	queryMap := getQueryMap(config)

	if queryMap != nil && len(queryMap) > 0 {
		var conditions []string
		argIndex := 1
		for k, v := range queryMap {
			valStr := fmt.Sprintf("%v", v)
			valStr = strings.ReplaceAll(valStr, "{field_value}", fmt.Sprintf("%v", fieldValue))
			valStr = strings.ReplaceAll(valStr, "{value}", fmt.Sprintf("%v", fieldValue))

			conditions = append(conditions, fmt.Sprintf("%s = $%d", k, argIndex))
			args = append(args, valStr)
			argIndex++
		}
		whereClause = strings.Join(conditions, " AND ")
	} else if config.Field != "" {
		whereClause = fmt.Sprintf("%s = $1", config.Field)
		args = []interface{}{fieldValue}
	} else {
		return nil, fmt.Errorf("either query or field must be specified for PostgreSQL provider")
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT 1", tableName, whereClause)

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgresql query failed: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("row not found")
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, fmt.Errorf("postgresql scan failed: %w", err)
	}

	result := make(map[string]interface{})
	for i, col := range columns {
		val := values[i]

		if bytes, ok := val.([]byte); ok {
			var jsonVal interface{}
			if err := json.Unmarshal(bytes, &jsonVal); err == nil {
				result[col] = jsonVal
			} else {
				result[col] = string(bytes)
			}
		} else {
			result[col] = val
		}
	}

	return result, nil
}
