
# Contributing to QBitFlow Go SDK

Thank you for your interest in contributing to the QBitFlow Go SDK! We welcome contributions from the community.

## How to Contribute

### Reporting Issues

If you find a bug or have a feature request:

1. Check if the issue already exists in the [GitHub Issues](https://github.com/qbitflow/qbitflow-go-sdk/issues)
2. If not, create a new issue with:
   - Clear title and description
   - Steps to reproduce (for bugs)
   - Expected vs. actual behavior
   - Go version and SDK version
   - Code samples if applicable

### Submitting Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** following our coding standards
3. **Add tests** for any new functionality
4. **Update documentation** if needed
5. **Run tests** to ensure everything passes
6. **Commit your changes** with clear commit messages
7. **Submit a pull request**

## Development Setup

### Prerequisites

- Go 1.18 or higher
- Git

### Setting Up Your Development Environment

```bash
# Clone your fork
git clone https://github.com/qbitflow/qbitflow-go-sdk.git
cd qbitflow-go-sdk

# Add upstream remote
git remote add upstream https://github.com/qbitflow/qbitflow-go-sdk.git

# Install dependencies
go mod download

# Run tests
go test ./...
```

## Coding Standards

### Go Style Guide

Follow the [official Go style guide](https://go.dev/doc/effective_go) and these additional guidelines:

1. **Formatting**: Use `gofmt` or `goimports`
   ```bash
   gofmt -w .
   ```

2. **Linting**: Use `golangci-lint`
   ```bash
   golangci-lint run
   ```

3. **Naming Conventions**:
   - Use camelCase for variables and functions
   - Use PascalCase for exported types and functions
   - Use descriptive names

4. **Documentation**:
   - Add comments for all exported types, functions, and constants
   - Follow [Go Doc Comments](https://go.dev/doc/comment) conventions
   - Include examples in doc comments when helpful

5. **Error Handling**:
   - Always handle errors
   - Provide context in error messages
   - Use custom error types when appropriate

### Code Organization

```
pkg/
├── qbitflow/       # Main SDK package
├── models/         # Data models and types
└── errors/         # Custom error types

examples/           # Example code
tests/             # Integration tests
```

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test -v ./tests/... -run TestPaymentSession

# Run benchmarks
go test -bench=. ./tests/...
```

### Writing Tests

1. Place unit tests in the same package as the code
2. Place integration tests in the `tests/` directory
3. Use table-driven tests for multiple scenarios
4. Mock external dependencies when appropriate
5. Ensure tests are deterministic and can run in parallel

Example:

```go
func TestNewFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "result", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := NewFeature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("NewFeature() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("NewFeature() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Documentation

### Updating Documentation

- Update README.md for user-facing changes
- Update code comments for API changes
- Add examples for new features
- Update CHANGELOG.md with your changes

### Example Code

When adding examples:

1. Place them in the `examples/` directory
2. Ensure they compile and run
3. Include clear comments explaining each step
4. Reference them in the README

## Commit Messages

Write clear and meaningful commit messages:

```
feat: Add support for webhook signature verification

- Implement signature validation
- Add tests for signature verification
- Update documentation with usage examples

Closes #123
```

### Commit Message Format

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Adding or updating tests
- `refactor:` Code refactoring
- `style:` Code style changes (formatting)
- `chore:` Maintenance tasks

## Pull Request Process

1. **Update your branch** with the latest from `main`
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Ensure tests pass** and code is formatted
   ```bash
   go test ./...
   gofmt -w .
   ```

3. **Push to your fork** and create a pull request
   ```bash
   git push origin your-branch-name
   ```

4. **Fill out the PR template** with:
   - Description of changes
   - Related issues
   - Testing performed
   - Breaking changes (if any)

5. **Respond to feedback** and update as needed

## Code Review

All submissions require code review. We use GitHub pull requests for this purpose.

Reviewers will check:
- Code quality and style
- Test coverage
- Documentation
- Backward compatibility
- Performance implications

## Release Process

Releases are handled by maintainers:

1. Version bump in `go.mod`
2. Update CHANGELOG.md
3. Create release tag
4. Publish release notes

## Community

- Be respectful and inclusive
- Follow the [Code of Conduct](CODE_OF_CONDUCT.md)
- Help others in issues and discussions
- Share your use cases and feedback

## Questions?

If you have questions about contributing:

- Open a [Discussion](https://github.com/qbitflow/qbitflow-go-sdk/discussions)
- Check existing [Issues](https://github.com/qbitflow/qbitflow-go-sdk/issues)
- Contact the maintainers

Thank you for contributing to QBitFlow Go SDK! 🚀
