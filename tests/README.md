
# Integration Tests

This directory contains integration tests for the QBitFlow Go SDK.

## Running Tests

### Basic Test Run

```bash
go test -v ./tests/...
```

### With API Key

To run tests that require a valid API key:

```bash
export QBITFLOW_API_KEY="your-api-key-here"
go test -v ./tests/...
```

### Run Specific Test

```bash
go test -v ./tests/... -run TestPaymentSession
```

### Run with Coverage

```bash
go test -cover ./tests/...
```

### Verbose Coverage Report

```bash
go test -coverprofile=coverage.out ./tests/...
go tool cover -html=coverage.out
```

### Run Benchmarks

```bash
go test -bench=. ./tests/...
```

## Test Categories

### Unit Tests
- Client initialization
- Validation logic
- Model structures
- Constants

### Integration Tests
- Payment session creation
- Subscription management
- Transaction status checking
- Payment retrieval

**Note:** Integration tests require:
1. A valid QBitFlow API key
2. Access to the QBitFlow API
3. May create actual sessions (in test mode if configured)

## Environment Variables

- `QBITFLOW_API_KEY`: Your QBitFlow API key for integration tests

## Test Coverage

The test suite covers:
- ✅ Client initialization and configuration
- ✅ Payment session creation (product ID and custom)
- ✅ Subscription session creation
- ✅ Pay-as-you-go subscription creation
- ✅ Transaction status checking
- ✅ Payment retrieval and pagination
- ✅ Validation error handling
- ✅ All transaction types and statuses
- ✅ Duration units

## CI/CD Integration

For CI/CD pipelines, tests that require API access will be skipped if:
- No API key is provided
- The API is not accessible
- The test environment is not configured

These tests will log a skip message instead of failing.

## Best Practices

1. **Always use test mode** when running integration tests
2. **Clean up** any created resources after tests
3. **Mock external calls** when possible for faster unit tests
4. **Use table-driven tests** for multiple similar test cases
5. **Log important information** for debugging test failures

## Adding New Tests

When adding new tests:

1. Follow the existing naming convention: `Test<Feature>`
2. Add table-driven tests for multiple scenarios
3. Include error case testing
4. Add benchmark tests for performance-critical code
5. Document any special requirements or setup needed
6. Skip tests gracefully if requirements are not met

Example:

```go
func TestNewFeature(t *testing.T) {
    t.Run("success case", func(t *testing.T) {
        // Test implementation
    })
    
    t.Run("error case", func(t *testing.T) {
        // Test implementation
    })
}
```

## Troubleshooting

### Tests Skipping
If tests are being skipped, ensure:
- `QBITFLOW_API_KEY` environment variable is set
- API key is valid and has proper permissions
- You have network access to the QBitFlow API

### Tests Failing
If tests are failing:
1. Check the error messages in test output
2. Verify your API key is correct
3. Ensure the API endpoint is accessible
4. Check if you're in test mode vs. live mode
5. Review the API documentation for any changes

## Contact

For issues with tests or test infrastructure:
- Open an issue on GitHub
- Contact the development team
- Check the main README for support information
