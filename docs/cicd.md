# CI/CD Pipeline Documentation

## Overview

The Verus Gateway uses GitHub Actions for continuous integration and deployment. Our CI/CD pipeline ensures code quality, reliability, and automated releases.

## Workflows

### 1. Test Workflow (`.github/workflows/test.yml`)

**Triggers:**
- Push to `main`, `master`, or `develop` branches
- Pull requests to these branches

**Jobs:**

#### Unit Tests
- **Go Version:** 1.23
- **Steps:**
  1. Checkout code
  2. Set up Go with caching
  3. Download and verify dependencies
  4. Run `go vet` for static analysis
  5. Run tests with race detector
  6. Generate coverage report
  7. Upload to Codecov (optional)
  8. Display total coverage percentage

#### Integration Tests
- **Dependencies:** Redis service
- **Steps:**
  1. Check if integration tests exist
  2. Run integration tests if available
  3. Skip gracefully if not present

**Best Practices Implemented:**
- ✅ Race detector enabled (`-race`)
- ✅ Proper timeouts (5m for unit tests, 10m for integration)
- ✅ Coverage reporting
- ✅ Dependency verification
- ✅ Service dependencies with health checks

### 2. Lint Workflow (`.github/workflows/lint.yml`)

**Triggers:**
- Push to any branch
- Pull requests

**Tools:**
- golangci-lint (latest version)
- Timeout: 5 minutes

**Best Practices Implemented:**
- ✅ Automatic caching
- ✅ Comprehensive linting rules
- ✅ Timeout protection

### 3. Release Workflow (`.github/workflows/release.yml`)

**Triggers:**
- Push of semver tags (e.g., `v1.0.0`, `v0.5.0-beta`)

**Jobs:**

#### 1. Validate Tag
- Validates semver format: `vX.Y.Z` or `vX.Y.Z-prerelease`
- Fails fast if tag format is invalid

#### 2. Test Before Release
- **Runs all tests with race detector**
- **Checks coverage threshold (minimum 50%)**
- Verifies dependencies
- Runs static analysis with `go vet`

**Coverage Threshold:**
```bash
Coverage must be >= 50% to proceed with release
Current coverage: 54.8%
```

#### 3. Lint Before Release
- Runs full linting suite
- Ensures code quality standards

#### 4. Build and Release
- **Depends on:** validate, test, lint (all must pass)
- Builds binaries for all platforms
- Creates GitHub release with:
  - Binary attachments
  - Auto-generated release notes
  - Changelog

#### 5. Docker Build and Push
- **Depends on:** validate, test, lint, build (all must pass)
- Builds multi-platform images (amd64, arm64)
- Pushes to GitHub Container Registry (GHCR)
- Tags: `latest`, `vX.Y.Z`, `vX.Y`, `vX`

**Best Practices Implemented:**
- ✅ Tag validation before any work
- ✅ Tests run before building
- ✅ Linting enforced before release
- ✅ Coverage threshold enforcement
- ✅ Multi-platform Docker builds
- ✅ Proper dependency chain (test → build → docker)
- ✅ Build caching for faster builds

## Pipeline Stages

```
Release Tag Created (vX.Y.Z)
         |
         ↓
   [1] Validate Tag
         |
    ┌────┴────┐
    ↓         ↓
[2] Test   [3] Lint
    |         |
    └────┬────┘
         ↓
    [4] Build & Release
         |
         ↓
    [5] Docker Build & Push
         |
         ↓
    Release Complete! 🎉
```

## Coverage Requirements

### Current Coverage
- **Overall:** 54.8%
- **Threshold:** 50% (enforced in CI)

### Package Coverage
| Package | Coverage |
|---------|----------|
| internal/crypto | 100% |
| internal/observability/logger | 100% |
| internal/storage | 93.4% |
| internal/config | 87.5% |
| internal/domain | 81.0% |
| pkg/verusrpc | 80.2% |
| internal/chain | 62.9% |
| internal/http/handler | 56.9% |
| internal/cache | 55.7% |

## Creating a Release

### Manual Release Process

1. **Ensure all tests pass locally:**
   ```bash
   make test
   make lint
   ```

2. **Check coverage:**
   ```bash
   make test-coverage
   # Must be >= 50%
   ```

3. **Update version and changelog:**
   ```bash
   # Update README.md, CHANGELOG.md if needed
   git add .
   git commit -m "chore: prepare for release vX.Y.Z"
   ```

4. **Create and push tag:**
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z - Brief description"
   git push && git push --tags
   ```

5. **Monitor CI/CD:**
   ```bash
   gh run list --limit 5
   gh run watch
   ```

### Release Versioning

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** (vX.0.0): Breaking API changes
- **MINOR** (v0.X.0): New features, backward compatible
- **PATCH** (v0.0.X): Bug fixes, backward compatible

Examples:
- `v0.5.0` - New test coverage improvements
- `v0.5.1` - Bug fix in handler
- `v1.0.0` - First stable release

### Pre-release Tags

For beta/alpha releases:
```bash
git tag -a v0.6.0-beta.1 -m "Beta release"
git tag -a v1.0.0-rc.1 -m "Release candidate"
```

## Troubleshooting CI/CD

### Tests Failing in CI but Pass Locally

1. **Check Go version:**
   ```bash
   go version
   # CI uses Go 1.23
   ```

2. **Check race conditions:**
   ```bash
   go test -race ./...
   ```

3. **Check for environment differences:**
   - Timezone differences
   - File path separators
   - Environment variables

### Coverage Below Threshold

If coverage drops below 50%:

1. **Identify uncovered code:**
   ```bash
   go test -coverprofile=coverage.out ./...
   go tool cover -html=coverage.out
   ```

2. **Add missing tests**

3. **Re-run locally:**
   ```bash
   make test-coverage
   ```

### Docker Build Fails

1. **Test locally:**
   ```bash
   docker build -t test .
   ```

2. **Check Dockerfile:**
   - Verify base image exists
   - Check COPY paths
   - Verify build context

### Tag Format Invalid

Error: "Tag does not follow semver format"

**Valid formats:**
- ✅ `v1.0.0`
- ✅ `v0.5.0`
- ✅ `v1.0.0-beta.1`
- ✅ `v2.1.0-rc.2`

**Invalid formats:**
- ❌ `1.0.0` (missing 'v' prefix)
- ❌ `v1.0` (incomplete version)
- ❌ `release-1.0.0` (wrong format)

## Security Considerations

### Secrets Management

**Required Secrets:**
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions
- `CODECOV_TOKEN` - Optional, for coverage reporting

**Never commit:**
- API keys
- Passwords
- Private keys
- Environment files with secrets

### Docker Image Security

- Images built from specific Go version
- Non-root user in containers
- Minimal attack surface
- Regular dependency updates

## Performance Optimization

### Caching Strategy

1. **Go module cache:**
   ```yaml
   - uses: actions/setup-go@v5
     with:
       cache: true
   ```

2. **Docker layer cache:**
   ```yaml
   cache-from: type=gha
   cache-to: type=gha,mode=max
   ```

### Build Time Improvements

- Use Go module proxy
- Parallel test execution
- Cached dependencies
- Incremental Docker builds

**Average CI times:**
- Test job: ~30 seconds
- Lint job: ~40 seconds
- Build job: ~1-2 minutes
- Docker job: ~3-5 minutes

## Monitoring CI/CD

### View Recent Runs

```bash
# List recent workflow runs
gh run list

# Watch current run
gh run watch

# View specific run
gh run view RUN_ID
```

### Workflow Status Badges

Add to README.md:
```markdown
![Tests](https://github.com/devdudeio/verus-gateway/workflows/Tests/badge.svg)
![Lint](https://github.com/devdudeio/verus-gateway/workflows/Lint/badge.svg)
```

## Best Practices Summary

✅ **Testing**
- All tests must pass before release
- Race detector enabled
- Coverage threshold enforced (50%)
- Integration tests with service dependencies

✅ **Code Quality**
- Linting enforced before release
- Static analysis with go vet
- Dependency verification

✅ **Release Process**
- Tag format validation
- Automated version detection
- Multi-platform builds
- Automated release notes

✅ **Docker**
- Multi-architecture support (amd64, arm64)
- Build caching
- Proper tagging strategy
- GHCR integration

✅ **Security**
- No secrets in code
- Minimal permissions
- Dependency scanning
- Regular updates

## References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Semantic Versioning](https://semver.org/)
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [golangci-lint](https://golangci-lint.run/)
- [Docker Multi-platform Builds](https://docs.docker.com/build/building/multi-platform/)

---

**Last Updated:** 2025-10-11
**Current Version:** v0.5.0
