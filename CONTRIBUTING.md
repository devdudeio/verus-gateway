# Contributing to Verus Gateway

First off, thank you for considering contributing to Verus Gateway! It's people like you that make this project a great tool for the Verus community.

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When you create a bug report, include as many details as possible:

- **Use a clear and descriptive title**
- **Describe the exact steps to reproduce the problem**
- **Provide specific examples** (code snippets, config files, etc.)
- **Describe the behavior you observed** and what you expected
- **Include logs and error messages**
- **Specify your environment** (OS, Go version, Verus version)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- **Use a clear and descriptive title**
- **Provide a detailed description** of the proposed functionality
- **Explain why this enhancement would be useful** to most users
- **Provide examples** of how the feature would be used
- **Consider the scope** - is this a core feature or better as a plugin?

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Follow the code style** used in the project
3. **Write or update tests** for your changes
4. **Update documentation** as needed
5. **Ensure tests pass** (`go test ./...`)
6. **Write clear commit messages**
7. **Submit your pull request**

## Development Setup

### Prerequisites

- Go 1.23 or higher
- Git
- A Verus node (testnet is fine for development)

### Setting Up Your Development Environment

```bash
# Fork and clone the repository
git clone https://github.com/YOUR-USERNAME/verus-gateway.git
cd verus-gateway

# Install dependencies
go mod download

# Copy example config
cp config.example.yaml config.yaml

# Edit config with your Verus node details
nano config.yaml

# Run tests
go test ./...

# Run the gateway locally
go run ./cmd/gateway
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestSpecificFunction ./internal/server

# Run integration tests (requires Verus node)
go test -tags=integration ./tests/...
```

### Code Style

- Follow standard Go conventions (use `gofmt` and `golint`)
- Write clear, descriptive variable and function names
- Add comments for exported functions and complex logic
- Keep functions focused and testable
- Use meaningful package names

### Commit Message Guidelines

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, missing semicolons, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**

```
feat(api): add support for batch file retrieval

Implement new endpoint /c/{chain}/files/batch that accepts multiple
TXIDs and returns files in a single response.

Closes #123
```

```
fix(crypto): handle edge case in decryption

Fix panic when EVK is malformed by adding proper validation
before attempting decryption.

Fixes #456
```

### Branch Naming

- `feature/description` - New features
- `fix/description` - Bug fixes
- `docs/description` - Documentation updates
- `refactor/description` - Code refactoring

## Project Structure

Understanding the project layout:

```
verus-gateway/
├── cmd/
│   └── gateway/          # Main application
├── internal/             # Private application code
│   ├── config/          # Configuration
│   ├── server/          # HTTP server
│   ├── verus/           # Verus RPC client
│   └── crypto/          # Encryption/decryption
├── pkg/                 # Public libraries
│   └── types/           # Shared types
├── tests/               # Integration tests
└── docs/                # Documentation
```

### Adding a New Feature

1. **Create an issue** describing the feature
2. **Discuss the design** with maintainers
3. **Create a feature branch** (`feature/your-feature-name`)
4. **Implement the feature** with tests
5. **Update documentation** (README, API docs, etc.)
6. **Submit a pull request**

### Testing Guidelines

- Write unit tests for all new code
- Aim for high test coverage (>80%)
- Include edge cases and error scenarios
- Mock external dependencies (RPC calls, etc.)
- Add integration tests for critical paths

Example test structure:

```go
func TestFileRetrieval(t *testing.T) {
    tests := []struct {
        name    string
        txid    string
        evk     string
        want    []byte
        wantErr bool
    }{
        {
            name: "valid file",
            txid: "abc123...",
            evk:  "zxviews...",
            want: []byte("file contents"),
            wantErr: false,
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Documentation

- Update README.md for user-facing changes
- Add inline code comments for complex logic
- Update API documentation for endpoint changes
- Add examples for new features
- Keep the changelog updated

## Community

- Be respectful and inclusive
- Help others in issues and discussions
- Share your use cases and experiences
- Provide constructive feedback
- Celebrate contributions from others

## Questions?

Don't hesitate to ask questions! You can:

- Open an issue with the `question` label
- Join the Verus Discord and ask in the development channel
- Reach out to maintainers directly

## Recognition

Contributors are recognized in:

- The README.md contributors section
- Release notes for significant contributions
- The GitHub contributors page

Thank you for contributing to Verus Gateway!
