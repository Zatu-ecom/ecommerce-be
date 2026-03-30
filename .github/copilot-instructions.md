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
11. **Use singular naming for API endpoints** - Follow RESTful singular resource naming

## 📝 API Naming Convention

We follow **singular naming** for all API endpoints:

| ✅ Correct (Singular)      | ❌ Incorrect (Plural)        |
| -------------------------- | ---------------------------- |
| `/api/product`             | `/api/products`              |
| `/api/product/:id`         | `/api/products/:id`          |
| `/api/product/:id/variant` | `/api/products/:id/variants` |
| `/api/category`            | `/api/product/category/`     |
| `/api/user`                | `/api/users`                 |
| `/api/order`               | `/api/orders`                |

**Examples:**

```
GET    /api/product           # Get all products
GET    /api/product/:id       # Get product by ID
POST   /api/product           # Create product
PUT    /api/product/:id       # Update product
DELETE /api/product/:id       # Delete product
GET    /api/product/:id/variant/:variantId  # Get variant
```

## 📛 Singular Naming Convention (Project-Wide)

We use **singular naming** consistently across the entire codebase:

### Database Tables

| ✅ Correct        | ❌ Incorrect       |
| ----------------- | ------------------ |
| `product`         | `products`         |
| `category`        | `categories`       |
| `user`            | `users`            |
| `order`           | `orders`           |
| `product_variant` | `product_variants` |

### Package Names

| ✅ Correct  | ❌ Incorrect  |
| ----------- | ------------- |
| `product/`  | `products/`   |
| `user/`     | `users/`      |
| `order/`    | `orders/`     |
| `category/` | `categories/` |

### Entity/Model Names

| ✅ Correct             | ❌ Incorrect             |
| ---------------------- | ------------------------ |
| `type Product struct`  | `type Products struct`   |
| `type Category struct` | `type Categories struct` |
| `ProductRepository`    | `ProductsRepository`     |
| `ProductService`       | `ProductsService`        |

### File Names

| ✅ Correct            | ❌ Incorrect            |
| --------------------- | ----------------------- |
| `product_handler.go`  | `products_handler.go`   |
| `category_service.go` | `categories_service.go` |
| `user_repository.go`  | `users_repository.go`   |

### Constants & Variables

| ✅ Correct       | ❌ Incorrect      |
| ---------------- | ----------------- |
| `APIBaseProduct` | `APIBaseProducts` |
| `productList`    | `productsList`    |
| `categoryMap`    | `categoriesMap`   |

## 📖 Quick Links

- Architecture: [ARCHITECTURE.md](../ARCHITECTURE.md)
- Coding Standards: [CODING_STANDARDS.md](../CODING_STANDARDS.md)
- Migrations: `migrations/`
- Tests: `test/integration/`
