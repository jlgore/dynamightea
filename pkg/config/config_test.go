package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Save original environment
	originalRegion := os.Getenv("AWS_REGION")
	originalProfile := os.Getenv("AWS_PROFILE")
	
	defer func() {
		// Restore original environment
		os.Setenv("AWS_REGION", originalRegion)
		os.Setenv("AWS_PROFILE", originalProfile)
	}()
	
	// Test with environment variables
	os.Setenv("AWS_REGION", "us-west-2")
	os.Setenv("AWS_PROFILE", "testprofile")
	
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	if cfg.Region != "us-west-2" {
		t.Errorf("Expected region to be us-west-2, got %s", cfg.Region)
	}
	
	if cfg.Profile != "testprofile" {
		t.Errorf("Expected profile to be testprofile, got %s", cfg.Profile)
	}
}

func TestGetCredentials(t *testing.T) {
	// Save original environment
	originalAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	originalSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	originalToken := os.Getenv("AWS_SESSION_TOKEN")
	
	defer func() {
		// Restore original environment
		os.Setenv("AWS_ACCESS_KEY_ID", originalAccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", originalSecretKey)
		os.Setenv("AWS_SESSION_TOKEN", originalToken)
	}()
	
	// Test with environment variables
	os.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	os.Setenv("AWS_SESSION_TOKEN", "test-session-token")
	
	cfg, _ := LoadConfig()
	creds, err := cfg.GetCredentials()
	
	if err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}
	
	if creds.AccessKeyID != "test-access-key" {
		t.Errorf("Expected access key to be test-access-key, got %s", creds.AccessKeyID)
	}
	
	if creds.SecretAccessKey != "test-secret-key" {
		t.Errorf("Expected secret key to be test-secret-key, got %s", creds.SecretAccessKey)
	}
	
	if creds.SessionToken != "test-session-token" {
		t.Errorf("Expected session token to be test-session-token, got %s", creds.SessionToken)
	}
}