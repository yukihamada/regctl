# Contributing to regctl Cloud

Thank you for your interest in contributing to regctl Cloud! This document provides guidelines for contributing to the project.

## Development Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/your-username/regctl.git
   cd regctl
   ```

3. Run the setup script:
   ```bash
   ./scripts/setup.sh
   ```

4. Copy `.env.example` to `.env` and fill in your API credentials

5. Start the development server:
   ```bash
   make dev
   ```

## Code Style

### Go Code
- Follow standard Go conventions
- Run `go fmt` before committing
- Use meaningful variable and function names
- Add comments for exported functions

### TypeScript Code
- Use ESLint configuration provided
- Prefer functional components
- Use TypeScript strict mode
- Document complex logic

## Testing

### Running Tests
```bash
# Run all tests
make test

# Run Go tests only
cd cmd/regctl && go test ./...

# Run TypeScript tests only
cd edge/workers && npm test

# Run e2e tests
make e2e
```

### Writing Tests
- Write unit tests for all new functionality
- Aim for >80% code coverage
- Use table-driven tests in Go
- Mock external API calls

## Pull Request Process

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and commit:
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

3. Follow [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` - New feature
   - `fix:` - Bug fix
   - `docs:` - Documentation changes
   - `test:` - Test additions/changes
   - `refactor:` - Code refactoring
   - `chore:` - Maintenance tasks

4. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

5. Create a Pull Request with:
   - Clear description of changes
   - Link to related issues
   - Screenshots (if UI changes)
   - Test results

## API Design Guidelines

1. Follow RESTful principles
2. Use proper HTTP status codes
3. Version APIs (e.g., `/api/v1/`)
4. Include pagination for list endpoints
5. Validate all inputs
6. Return consistent error formats

## Security

- Never commit secrets or API keys
- Use environment variables for configuration
- Validate and sanitize all user inputs
- Follow OWASP security guidelines
- Report security issues privately

## Documentation

- Update API documentation for endpoint changes
- Add JSDoc comments for TypeScript
- Update README for significant features
- Include examples in documentation

## Release Process

1. Ensure all tests pass
2. Update version numbers
3. Update CHANGELOG.md
4. Create a tagged release
5. Deploy to staging first
6. Monitor for issues
7. Deploy to production

## Questions?

- Open an issue for bugs or features
- Join our Discord community
- Email: dev@regctl.cloud

Thank you for contributing!