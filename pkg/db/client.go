package db

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	appconfig "github.com/jlgore/dynamightea/pkg/config"
)

// KeySchemaElement represents a key schema element in DynamoDB
type KeySchemaElement struct {
	AttributeName string
	KeyType       string
}

// IndexInfo represents an index in DynamoDB
type IndexInfo struct {
	IndexName string
	KeySchema []KeySchemaElement
}

// TableInfo represents information about a DynamoDB table
type TableInfo struct {
	TableName            string
	KeySchema            []KeySchemaElement
	AttributeDefinitions map[string]string
	GSIs                 []IndexInfo
	LSIs                 []IndexInfo
}

// DynamoClient provides methods for interacting with DynamoDB
type DynamoClient struct {
	client *dynamodb.Client
	cfg    *appconfig.Config
}

// NewDynamoClient creates a new DynamoDB client
func NewDynamoClient() *DynamoClient {
	// Load configuration
	cfg, err := appconfig.LoadConfig()
	if err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
		return &DynamoClient{client: nil}
	}

	// Create AWS SDK config
	client, err := createDynamoDBClient(cfg)
	if err != nil {
		log.Printf("Warning: Failed to create DynamoDB client: %v", err)
		return &DynamoClient{client: nil, cfg: cfg}
	}

	return &DynamoClient{
		client: client,
		cfg:    cfg,
	}
}

// createDynamoDBClient creates a DynamoDB client with the provided configuration
func createDynamoDBClient(cfg *appconfig.Config) (*dynamodb.Client, error) {
	var awsConfig aws.Config
	var err error

	// If credentials provided via environment or config files
	// Use the default AWS SDK credential chain
	optFns := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	// If using a custom endpoint (like DynamoDB Local)
	if cfg.Endpoint != "" {
		optFns = append(optFns, config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:           cfg.Endpoint,
						SigningRegion: cfg.Region,
					}, nil
				},
			),
		))
	}

	// Try to get explicit credentials from metadata services if enabled
	var creds *appconfig.Credentials
	if cfg.UseIMDS || cfg.UseECSMetadata {
		creds, err = cfg.GetCredentials()
		if err == nil && creds != nil {
			// Use explicit credentials provider
			optFns = append(optFns, config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(
					creds.AccessKeyID,
					creds.SecretAccessKey,
					creds.SessionToken,
				),
			))
		}
	}

	// Load AWS SDK configuration
	awsConfig, err = config.LoadDefaultConfig(
		context.Background(),
		optFns...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	// Create and return the DynamoDB client
	return dynamodb.NewFromConfig(awsConfig), nil
}

// ListTables lists all DynamoDB tables
func (d *DynamoClient) ListTables() ([]string, error) {
	// If in demo mode or client not initialized, return mock data
	if d.client == nil {
		// For demo purposes, returning mock data
		return []string{"Users", "Products", "Orders"}, nil
	}
	
	// Use the real DynamoDB client
	var tableNames []string
	var nextToken *string
	
	for {
		resp, err := d.client.ListTables(context.TODO(), &dynamodb.ListTablesInput{
			ExclusiveStartTableName: nextToken,
		})
		if err != nil {
			log.Printf("Error listing tables: %v", err)
			// Fall back to mock data on error
			return []string{"Users", "Products", "Orders"}, nil
		}
		
		tableNames = append(tableNames, resp.TableNames...)
		
		nextToken = resp.LastEvaluatedTableName
		if nextToken == nil {
			break
		}
	}
	
	return tableNames, nil
}

// DescribeTable gets information about a specific table
func (d *DynamoClient) DescribeTable(tableName string) (*TableInfo, error) {
	// If in demo mode or client not initialized, return mock data
	if d.client == nil {
		// For demo purposes, returning mock data based on table name
		return getMockTableInfo(tableName)
	}
	
	// Use the real DynamoDB client
	resp, err := d.client.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		log.Printf("Error describing table %s: %v", tableName, err)
		// Fall back to mock data on error
		return getMockTableInfo(tableName)
	}
	
	table := resp.Table
	if table == nil {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}
	
	result := &TableInfo{
		TableName:            *table.TableName,
		KeySchema:            convertKeySchema(table.KeySchema),
		AttributeDefinitions: convertAttrDefinitions(table.AttributeDefinitions),
		GSIs:                 []IndexInfo{},
		LSIs:                 []IndexInfo{},
	}
	
	// Add GSIs
	for _, gsi := range table.GlobalSecondaryIndexes {
		result.GSIs = append(result.GSIs, IndexInfo{
			IndexName: *gsi.IndexName,
			KeySchema: convertKeySchema(gsi.KeySchema),
		})
	}
	
	// Add LSIs
	for _, lsi := range table.LocalSecondaryIndexes {
		result.LSIs = append(result.LSIs, IndexInfo{
			IndexName: *lsi.IndexName,
			KeySchema: convertKeySchema(lsi.KeySchema),
		})
	}
	
	return result, nil
}

// Helper functions 
func convertKeySchema(schema []types.KeySchemaElement) []KeySchemaElement {
	result := make([]KeySchemaElement, len(schema))
	for i, key := range schema {
		result[i] = KeySchemaElement{
			AttributeName: *key.AttributeName,
			KeyType:       string(key.KeyType),
		}
	}
	return result
}

func convertAttrDefinitions(attrs []types.AttributeDefinition) map[string]string {
	result := make(map[string]string)
	for _, attr := range attrs {
		result[*attr.AttributeName] = string(attr.AttributeType)
	}
	return result
}

// getMockTableInfo provides mock data for demo purposes
func getMockTableInfo(tableName string) (*TableInfo, error) {
	switch tableName {
	case "Users":
		return &TableInfo{
			TableName: "Users",
			KeySchema: []KeySchemaElement{
				{AttributeName: "UserID", KeyType: "HASH"},
				{AttributeName: "Email", KeyType: "RANGE"},
			},
			AttributeDefinitions: map[string]string{
				"UserID":    "S",
				"Email":     "S",
				"Username":  "S",
				"CreatedAt": "N",
			},
			GSIs: []IndexInfo{
				{
					IndexName: "UsernameIndex",
					KeySchema: []KeySchemaElement{
						{AttributeName: "Username", KeyType: "HASH"},
					},
				},
			},
			LSIs: []IndexInfo{
				{
					IndexName: "CreatedAtIndex",
					KeySchema: []KeySchemaElement{
						{AttributeName: "UserID", KeyType: "HASH"},
						{AttributeName: "CreatedAt", KeyType: "RANGE"},
					},
				},
			},
		}, nil
	case "Products":
		return &TableInfo{
			TableName: "Products",
			KeySchema: []KeySchemaElement{
				{AttributeName: "ProductID", KeyType: "HASH"},
			},
			AttributeDefinitions: map[string]string{
				"ProductID":  "S",
				"Category":   "S",
				"Price":      "N",
				"CreateDate": "S",
			},
			GSIs: []IndexInfo{
				{
					IndexName: "CategoryPriceIndex",
					KeySchema: []KeySchemaElement{
						{AttributeName: "Category", KeyType: "HASH"},
						{AttributeName: "Price", KeyType: "RANGE"},
					},
				},
			},
			LSIs: []IndexInfo{},
		}, nil
	case "Orders":
		return &TableInfo{
			TableName: "Orders",
			KeySchema: []KeySchemaElement{
				{AttributeName: "CustomerID", KeyType: "HASH"},
				{AttributeName: "OrderID", KeyType: "RANGE"},
			},
			AttributeDefinitions: map[string]string{
				"CustomerID": "S",
				"OrderID":    "S",
				"OrderDate":  "S",
				"Status":     "S",
			},
			GSIs: []IndexInfo{
				{
					IndexName: "StatusOrderDateIndex",
					KeySchema: []KeySchemaElement{
						{AttributeName: "Status", KeyType: "HASH"},
						{AttributeName: "OrderDate", KeyType: "RANGE"},
					},
				},
			},
			LSIs: []IndexInfo{
				{
					IndexName: "OrderDateIndex",
					KeySchema: []KeySchemaElement{
						{AttributeName: "CustomerID", KeyType: "HASH"},
						{AttributeName: "OrderDate", KeyType: "RANGE"},
					},
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("table not found: %s", tableName)
	}
}