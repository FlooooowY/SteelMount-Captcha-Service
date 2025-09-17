# SteelMount Captcha Service - Tests

This directory contains comprehensive tests for the SteelMount Captcha Service.

## Test Structure

```
tests/
├── unit/           # Unit tests for individual components
├── integration/    # Integration tests for component interactions
├── performance/    # Performance and load tests
├── security/       # Security-specific tests
├── Makefile        # Test execution commands
└── README.md       # This file
```

## Test Categories

### Unit Tests (`tests/unit/`)

- **captcha_engine_test.go** - Tests for captcha generation engine
- **rate_limiter_test.go** - Tests for rate limiting functionality
- **bot_detector_test.go** - Tests for bot detection algorithms

### Integration Tests (`tests/integration/`)

- **server_test.go** - Tests for server startup, shutdown, and port discovery

### Performance Tests (`tests/performance/`)

- **load_test.go** - Load testing for RPS targets and memory usage

### Security Tests (`tests/security/`)

- **security_test.go** - Tests for IP blocking, rate limiting, and bot detection

## Running Tests

### Prerequisites

- Go 1.21+
- Redis (optional, for integration tests)

### Quick Start

```bash
# Run all unit tests
make test

# Run all tests
make test-all

# Run with coverage
make test-coverage
```

### Individual Test Categories

```bash
# Unit tests only
make test-unit

# Integration tests only
make test-integration

# Performance tests only
make test-performance

# Security tests only
make test-security
```

### Specific Tests

```bash
# Run specific test
make test-specific TEST=TestCaptchaGenerationRPS

# Run benchmarks
make test-bench
```

## Test Requirements

### Performance Targets

- **Captcha Generation**: ≥100 RPS per type
- **Security Checks**: ≥500 RPS
- **Concurrent Processing**: ≥200 RPS with 50 goroutines
- **Memory Usage**: <1KB per captcha

### Security Requirements

- **Rate Limiting**: Blocks requests exceeding limits
- **IP Blocking**: Blocks IPs after failed attempts
- **Bot Detection**: Detects suspicious patterns
- **Manual Blocking**: Supports manual IP management

## Test Configuration

Tests use minimal configuration to avoid external dependencies:

- Redis is optional (local-only mode when unavailable)
- Port ranges are limited to avoid conflicts
- Timeouts are set appropriately for CI/CD

## Continuous Integration

Tests are designed to run in CI/CD environments:

- No external dependencies required
- Deterministic results
- Appropriate timeouts
- Race condition detection

## Coverage

Run tests with coverage to see code coverage:

```bash
make test-coverage
```

This generates:

- `coverage.out` - Coverage data file
- `coverage.html` - HTML coverage report

## Troubleshooting

### Common Issues

1. **Port conflicts**: Tests use port ranges 38000-40000
2. **Redis connection**: Tests work without Redis (local-only mode)
3. **Timeout issues**: Increase timeout for slow environments
4. **Race conditions**: Use `-race` flag for detection

### Debug Mode

```bash
# Run with verbose output
go test -v -race ./tests/unit/...

# Run specific test with debug
go test -v -race -run TestSpecific ./tests/...
```

## Adding New Tests

1. **Unit Tests**: Add to `tests/unit/` for individual components
2. **Integration Tests**: Add to `tests/integration/` for component interactions
3. **Performance Tests**: Add to `tests/performance/` for load testing
4. **Security Tests**: Add to `tests/security/` for security features

### Test Naming Convention

- `TestComponent_Function` - Unit tests
- `TestIntegration_Feature` - Integration tests
- `TestPerformance_Metric` - Performance tests
- `TestSecurity_Feature` - Security tests

### Best Practices

- Use table-driven tests for multiple scenarios
- Include performance assertions for critical paths
- Test both success and failure cases
- Use appropriate timeouts
- Clean up resources in tests
