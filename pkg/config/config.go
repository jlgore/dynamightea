package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Config holds application configuration
type Config struct {
	Region          string
	Profile         string
	Endpoint        string
	ConfigFile      string
	CredentialsFile string
	UseIMDS         bool
	IMDSVersion     string // "v1", "v2"
	UseECSMetadata  bool
}

// Credentials represents AWS credentials
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

// LoadConfig loads the application configuration
func LoadConfig() (*Config, error) {
	// Default to environment variables or AWS config file
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region == "" {
		region = "us-east-1" // Default region
	}

	// Get AWS profile
	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "default"
	}

	// Endpoint for local development (e.g., DynamoDB Local)
	endpoint := os.Getenv("AWS_DYNAMODB_ENDPOINT")

	// AWS config file locations
	homeDir, _ := os.UserHomeDir()
	awsConfigDir := filepath.Join(homeDir, ".aws")
	configFile := filepath.Join(awsConfigDir, "config")
	credentialsFile := filepath.Join(awsConfigDir, "credentials")

	// IMDS configuration
	useIMDS := os.Getenv("AWS_USE_IMDS") != "false" // Use IMDS by default in EC2 environment
	imdsVersion := os.Getenv("AWS_IMDS_VERSION")
	if imdsVersion == "" {
		imdsVersion = "v2" // Default to IMDSv2 which is more secure
	}

	// ECS metadata configuration
	useECSMetadata := os.Getenv("AWS_ECS_METADATA_ENDPOINT") != ""

	return &Config{
		Region:          region,
		Profile:         profile,
		Endpoint:        endpoint,
		ConfigFile:      configFile,
		CredentialsFile: credentialsFile,
		UseIMDS:         useIMDS,
		IMDSVersion:     imdsVersion,
		UseECSMetadata:  useECSMetadata,
	}, nil
}

// GetCredentials attempts to retrieve AWS credentials from various sources
func (c *Config) GetCredentials() (*Credentials, error) {
	// First check environment variables (highest precedence)
	creds := &Credentials{
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		SessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
	}

	if creds.AccessKeyID != "" && creds.SecretAccessKey != "" {
		return creds, nil
	}

	// Try ECS metadata service if configured
	if c.UseECSMetadata {
		if ecsCreds, err := getECSCredentials(); err == nil {
			return ecsCreds, nil
		}
	}

	// Try IMDS if configured
	if c.UseIMDS {
		var imdsCreds *Credentials
		var err error

		switch c.IMDSVersion {
		case "v1":
			imdsCreds, err = getIMDSv1Credentials()
		case "v2":
			imdsCreds, err = getIMDSv2Credentials()
		default:
			// Try v2 first, fall back to v1
			imdsCreds, err = getIMDSv2Credentials()
			if err != nil {
				imdsCreds, err = getIMDSv1Credentials()
			}
		}

		if err == nil && imdsCreds != nil {
			return imdsCreds, nil
		}
	}

	// Could add logic to parse AWS config/credentials files here

	return nil, fmt.Errorf("unable to locate AWS credentials")
}

// getIMDSv1Credentials retrieves credentials from EC2 Instance Metadata Service (IMDSv1)
func getIMDSv1Credentials() (*Credentials, error) {
	// Get the role name from the instance metadata
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://169.254.169.254/latest/meta-data/iam/security-credentials/")
	if err != nil {
		return nil, fmt.Errorf("failed to get IAM role from IMDS: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get IAM role from IMDS: %s", resp.Status)
	}

	roleName, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read role name from IMDS: %v", err)
	}

	// Get the credentials using the role name
	resp, err = client.Get(fmt.Sprintf("http://169.254.169.254/latest/meta-data/iam/security-credentials/%s", roleName))
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials from IMDS: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get credentials from IMDS: %s", resp.Status)
	}

	var credResponse struct {
		AccessKeyID     string `json:"AccessKeyId"`
		SecretAccessKey string `json:"SecretAccessKey"`
		Token           string `json:"Token"`
		Expiration      string `json:"Expiration"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&credResponse); err != nil {
		return nil, fmt.Errorf("failed to decode credentials from IMDS: %v", err)
	}

	expiration, err := time.Parse(time.RFC3339, credResponse.Expiration)
	if err != nil {
		expiration = time.Now().Add(1 * time.Hour) // Default expiration
	}

	return &Credentials{
		AccessKeyID:     credResponse.AccessKeyID,
		SecretAccessKey: credResponse.SecretAccessKey,
		SessionToken:    credResponse.Token,
		Expiration:      expiration,
	}, nil
}

// getIMDSv2Credentials retrieves credentials from EC2 Instance Metadata Service (IMDSv2)
func getIMDSv2Credentials() (*Credentials, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Step 1: Get a session token
	tokenReq, err := http.NewRequest("PUT", "http://169.254.169.254/latest/api/token", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %v", err)
	}
	tokenReq.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600") // 6 hours

	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get token from IMDSv2: %v", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get token from IMDSv2: %s", tokenResp.Status)
	}

	token, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token from IMDSv2: %v", err)
	}

	// Step 2: Get the role name using the token
	roleReq, err := http.NewRequest("GET", "http://169.254.169.254/latest/meta-data/iam/security-credentials/", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create role request: %v", err)
	}
	roleReq.Header.Set("X-aws-ec2-metadata-token", string(token))

	roleResp, err := client.Do(roleReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get IAM role from IMDSv2: %v", err)
	}
	defer roleResp.Body.Close()

	if roleResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get IAM role from IMDSv2: %s", roleResp.Status)
	}

	roleName, err := io.ReadAll(roleResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read role name from IMDSv2: %v", err)
	}

	// Step 3: Get the credentials using the role name and token
	credsReq, err := http.NewRequest("GET", fmt.Sprintf("http://169.254.169.254/latest/meta-data/iam/security-credentials/%s", roleName), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create credentials request: %v", err)
	}
	credsReq.Header.Set("X-aws-ec2-metadata-token", string(token))

	credsResp, err := client.Do(credsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials from IMDSv2: %v", err)
	}
	defer credsResp.Body.Close()

	if credsResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get credentials from IMDSv2: %s", credsResp.Status)
	}

	var credResponse struct {
		AccessKeyID     string `json:"AccessKeyId"`
		SecretAccessKey string `json:"SecretAccessKey"`
		Token           string `json:"Token"`
		Expiration      string `json:"Expiration"`
	}

	if err := json.NewDecoder(credsResp.Body).Decode(&credResponse); err != nil {
		return nil, fmt.Errorf("failed to decode credentials from IMDSv2: %v", err)
	}

	expiration, err := time.Parse(time.RFC3339, credResponse.Expiration)
	if err != nil {
		expiration = time.Now().Add(1 * time.Hour) // Default expiration
	}

	return &Credentials{
		AccessKeyID:     credResponse.AccessKeyID,
		SecretAccessKey: credResponse.SecretAccessKey,
		SessionToken:    credResponse.Token,
		Expiration:      expiration,
	}, nil
}

// getECSCredentials retrieves credentials from ECS Task Metadata Endpoint
func getECSCredentials() (*Credentials, error) {
	// Get the credentials endpoint from environment
	metadataEndpoint := os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI")
	if metadataEndpoint == "" {
		return nil, fmt.Errorf("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI not set")
	}

	ecsEndpoint := fmt.Sprintf("http://169.254.170.2%s", metadataEndpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ecsEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for ECS credentials: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials from ECS metadata: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get credentials from ECS metadata: %s", resp.Status)
	}

	var credResponse struct {
		AccessKeyID     string `json:"AccessKeyId"`
		SecretAccessKey string `json:"SecretAccessKey"`
		Token           string `json:"Token"`
		Expiration      string `json:"Expiration"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&credResponse); err != nil {
		return nil, fmt.Errorf("failed to decode credentials from ECS metadata: %v", err)
	}

	expiration, err := time.Parse(time.RFC3339, credResponse.Expiration)
	if err != nil {
		expiration = time.Now().Add(1 * time.Hour) // Default expiration
	}

	return &Credentials{
		AccessKeyID:     credResponse.AccessKeyID,
		SecretAccessKey: credResponse.SecretAccessKey,
		SessionToken:    credResponse.Token,
		Expiration:      expiration,
	}, nil
}