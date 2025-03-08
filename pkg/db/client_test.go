package db

import (
	"testing"
)

func TestMockListTables(t *testing.T) {
	client := NewDynamoClient()
	tables, err := client.ListTables()
	
	if err != nil {
		t.Fatalf("Error listing tables: %v", err)
	}
	
	// Verify that we get the three mock tables
	expectedTables := map[string]bool{
		"Users":    true,
		"Products": true,
		"Orders":   true,
	}
	
	if len(tables) != 3 {
		t.Errorf("Expected 3 tables, got %d", len(tables))
	}
	
	for _, table := range tables {
		if !expectedTables[table] {
			t.Errorf("Unexpected table: %s", table)
		}
	}
}

func TestMockDescribeTable(t *testing.T) {
	client := NewDynamoClient()
	
	// Test Users table
	userTable, err := client.DescribeTable("Users")
	if err != nil {
		t.Fatalf("Error describing Users table: %v", err)
	}
	
	if userTable.TableName != "Users" {
		t.Errorf("Expected table name to be Users, got %s", userTable.TableName)
	}
	
	if len(userTable.KeySchema) != 2 {
		t.Errorf("Expected 2 key schema elements for Users, got %d", len(userTable.KeySchema))
	}
	
	if len(userTable.GSIs) != 1 {
		t.Errorf("Expected 1 GSI for Users, got %d", len(userTable.GSIs))
	}
	
	if len(userTable.LSIs) != 1 {
		t.Errorf("Expected 1 LSI for Users, got %d", len(userTable.LSIs))
	}
	
	// Test non-existent table
	_, err = client.DescribeTable("NonExistentTable")
	if err == nil {
		t.Error("Expected error for non-existent table, got nil")
	}
}