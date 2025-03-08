# DynamighTea

A terminal-based UI (TUI) frontend for Amazon DynamoDB, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- Browse DynamoDB tables
- View table schema and metadata
- Explore Global Secondary Indexes (GSIs) and Local Secondary Indexes (LSIs)
- Navigate with keyboard shortcuts
- Simple, intuitive interface

## Installation

```bash
# Clone the repository
git clone https://github.com/jlgore/dynamightea.git
cd dynamightea

# Build the binary
go build -o dynamightea

# Run the application
./dynamightea
```

## Configuration

DynamighTea uses AWS SDK for Go to connect to DynamoDB. It supports multiple authentication methods:

### AWS Credentials

1. Environment variables:
   - `AWS_REGION` or `AWS_DEFAULT_REGION`: The AWS region to use
   - `AWS_PROFILE`: The AWS profile to use from your AWS config
   - `AWS_DYNAMODB_ENDPOINT`: Custom endpoint for connecting to DynamoDB Local or other endpoints
   - `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`: Direct credential values

2. AWS config files:
   - `~/.aws/config`
   - `~/.aws/credentials`

### EC2 Instance Metadata Service (IMDS)

DynamighTea supports both IMDSv1 and IMDSv2 for retrieving credentials from EC2 instances:

- `AWS_USE_IMDS`: Set to "false" to disable IMDS usage (enabled by default on EC2)
- `AWS_IMDS_VERSION`: Set to "v1" or "v2" to specify the IMDS version (defaults to "v2")

### ECS Container Credentials

For applications running in Amazon ECS, DynamighTea will automatically use the ECS Task Metadata Endpoint when the following environment variable is set:

- `AWS_CONTAINER_CREDENTIALS_RELATIVE_URI`: This is automatically set by ECS task daemon

## Usage

```bash
# Start DynamighTea
./dynamightea
```

### Keyboard Navigation

- `↑/↓` or `k/j`: Navigate through tables and options
- `Enter`: Select a table to view its details
- `Tab`: Switch between different views (Tables, Table Details, Indexes)
- `q` or `Ctrl+C`: Quit the application

## Development

### Prerequisites

- Go 1.21 or higher
- AWS account with DynamoDB access (or DynamoDB Local for development)

### Build from source

```bash
# Get dependencies
go mod tidy

# Build
go build -o dynamightea
```

### Continuous Integration

This project uses GitHub Actions for continuous integration and delivery:

- Automatically builds on Linux, macOS, and Windows
- Cross-compiles for ARM64, ARM, and other architectures
- Runs unit tests on all platforms
- Creates release artifacts (zip for Windows, tar.gz for others)
- Automatically creates GitHub Releases when version tags are pushed

To create a new release:

1. Tag the commit with a version:
   ```bash
   git tag v1.0.0
   ```

2. Push the tag to GitHub:
   ```bash
   git push origin v1.0.0
   ```

3. GitHub Actions will automatically build all binaries and create a release

## License

MIT