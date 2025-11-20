# GitHub Copilot Instructions

## 📚 Project Documentation

Before making any code changes, please review these documents:

- **[Architecture Documentation](../ARCHITECTURE.md)** - Understand the project structure, design patterns, and service organization
- **[Coding Standards](../CODING_STANDARDS.md)** - Follow coding best practices, conventions, and quality guidelines

## 🎯 Key Guidelines

When writing code for this project:

1. **Follow the established architecture** - Services → Handlers → Repositories pattern
2. **Never bypass service layers** - Always call services, not repositories directly
3. **Use dependency injection** - All dependencies through constructors
4. **Add validation in models** - Use `binding` tags and `dive` for nested structs
5. **Use pointers for nullable fields** - In update models to distinguish null from empty
6. **Always create new migrations** - Never modify existing migration files
7. **Maintain quality seed data** - Follow API rules (e.g., products need variants)
8. **Write integration tests** - Full coverage for all API endpoints
9. **Use structured logging** - Include correlation ID and context
10. **Return AppError** - Use application-level errors, not generic strings

## 📖 Quick Links

- Architecture: [ARCHITECTURE.md](../ARCHITECTURE.md)
- Coding Standards: [CODING_STANDARDS.md](../CODING_STANDARDS.md)
- Migrations: `migrations/`
- Tests: `test/integration/`
