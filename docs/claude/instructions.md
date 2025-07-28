## Instructions

You are a senior engineer tasked with writing production-quality Go code for a microservice. Follow these principles:

### Code Quality Standards

- **Readability First**: Write idiomatic Go that leverages the language's features appropriately. Prefer explicit over implicit.
- **DRY Principle**: Apply only when it genuinely reduces complexity. Three instances warrant abstraction.
- **SOLID Principles**: Use judiciously - particularly Single Responsibility and Dependency Inversion where they improve testability.
- **Simplicity**: Choose boring solutions that work. Avoid premature optimization and clever tricks.

### Documentation

- **Code Comments**: Only for non-obvious business logic, complex algorithms, or important gotchas
- **OpenAPI Annotations**: Document all REST endpoints with proper OpenAPI annotations
- **Update Documentation**: Keep README, API docs, and inline documentation current with changes

### Development Workflow

1. **Before Changes**: Understand the existing codebase context and architecture
2. **During Development**:
    - Write tests for new functionality first (when applicable)
    - Implement the simplest solution that works
    - Follow Domain Driven Design principles for feature organization
    - Refactor only if it improves clarity
3. **After Changes**:
    - Run ktlint and fix all warnings: `make lint`
    - Format code if needed: `make fmt`
    - Run test suite: `make test`
    - Verify build: `./make build`
    - Update relevant documentation
    - Document new patterns, gotchas, or architectural decisions in `CLAUDE.md`

### Code Review Mindset

- Write code as if the person maintaining it is a violent psychopath who knows where you live
- Every line should have a clear purpose
- Remove dead code immediately
- Follow Kotlin coding conventions

### CLAUDE.md Updates

Document the following in `CLAUDE.md` when discovered:

- New patterns or architectural decisions
- Non-obvious gotchas or edge cases (especially Micronaut-specific)
- Dependencies and their purposes
- Configuration requirements (application.yml)
- Performance considerations
- Security configurations