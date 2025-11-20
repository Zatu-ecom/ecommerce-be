# 🛍️ Zatu E-commerce Backend

**Zatu** is a modern, scalable e-commerce backend API built to power next-generation online shopping experiences. Whether you're building a mobile app, web application, or admin dashboard, Zatu provides a robust REST API with comprehensive features for managing products, orders, payments, and user accounts.

---

## 🎯 What is Zatu?

Zatu is a complete e-commerce backend solution that handles:

- **User Management** - Customer registration, authentication, profile management, and address handling
- **Product Catalog** - Advanced product management with categories, variants, attributes, and dynamic options
- **Multi-Tenant Architecture** - Support for multiple sellers with complete data isolation
- **Order Processing** - Full order lifecycle from cart to checkout to fulfillment
- **Payment Integration** - Ready for payment gateway integration
- **Search & Discovery** - Powerful product search and filtering capabilities
- **Admin Tools** - Complete administrative control over the platform

### Key Features

✅ **Role-Based Access Control** - Admin, Seller, and Customer roles with fine-grained permissions  
✅ **Multi-Seller Support** - Each seller operates independently with isolated data  
✅ **Product Variants** - Handle products with multiple options (size, color, storage, etc.)  
✅ **Category Hierarchy** - Nested categories with attribute inheritance  
✅ **Distributed Tracing** - Correlation IDs for tracking requests across services  
✅ **Caching Layer** - Redis caching for improved performance  
✅ **Comprehensive Testing** - Full integration test coverage for reliability  
✅ **API Documentation** - Postman collection included for easy testing

---

## 🛠️ Technology Stack

| Layer         | Technology                  | Version |
| ------------- | --------------------------- | ------- |
| **Language**  | Go (Golang)                 | 1.21+   |
| **Framework** | Gin                         | Latest  |
| **Database**  | PostgreSQL                  | 14+     |
| **Cache**     | Redis                       | 7+      |
| **ORM**       | GORM                        | Latest  |
| **Auth**      | JWT (JSON Web Tokens)       | -       |
| **Testing**   | Go testing + Testcontainers | -       |

---

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 14 or higher
- Redis 7 or higher
- Git

### Installation

1. **Clone the repository**

   ```bash
   git clone https://github.com/Zatu-ecom/ecommerce-be.git
   cd ecommerce-be
   ```

2. **Install dependencies**

   ```bash
   go mod download
   ```

3. **Set up environment variables**

   ```bash
   cp .env.example .env
   # Edit .env with your database credentials
   ```

4. **Run database migrations**

   ```bash
   cd migrations
   ./run_migrations.sh
   cd ..
   ```

5. **Start the application**
   ```bash
   go run main.go
   ```

The API will be available at `http://localhost:8080`

---

## 📚 Documentation

Comprehensive documentation is available to help you understand and contribute to the project:

- **[ARCHITECTURE.md](./ARCHITECTURE.md)** - High-level architecture overview, design patterns, and system structure
- **[CODING_STANDARDS.md](./CODING_STANDARDS.md)** - Coding best practices, conventions, and quality guidelines

### API Testing

Import the Postman collection for easy API testing:

- **[Ecommerce API Collection](./Ecommerce%20API%20-%20Complete%20Collection%202025.postman_collection.json)**

---

## 🔑 Environment Configuration

Create a `.env` file in the root directory with the following variables:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=ecommerce

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# Application Configuration
PORT=8080
GIN_MODE=debug

# JWT Configuration
JWT_SECRET=your-secret-key-here
JWT_EXPIRY_HOURS=24
```

---

## 🧪 Running Tests

Run the comprehensive integration test suite:

```bash
# Run all tests
go test ./test/integration/... -v

# Run tests with coverage
go test ./test/integration/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific service tests
go test ./test/integration/product/... -v
go test ./test/integration/user/... -v
```

---

## 🏗️ Project Architecture

Zatu follows a **modular monolithic architecture** designed for easy transition to microservices. The codebase is organized into self-contained service modules:

```
ecommerce-be/
├── main.go              # Application entry point
├── common/              # Shared utilities (auth, cache, logging, etc.)
├── user/                # User service
├── product/             # Product catalog service
├── order/               # Order management service
├── payment/             # Payment processing service
├── notification/        # Notification service
├── migrations/          # Database migrations & seed data
└── test/                # Integration tests
```

Each service follows **Clean Architecture** principles with clear separation of concerns:

- **Entity** - Database models
- **Repository** - Data access layer
- **Service** - Business logic layer
- **Handler** - HTTP request handlers
- **Routes** - API route definitions

For detailed architecture information, see [ARCHITECTURE.md](./ARCHITECTURE.md).

---

## 👥 User Roles

Zatu supports three primary user roles:

| Role         | Description             | Capabilities                                  |
| ------------ | ----------------------- | --------------------------------------------- |
| **Admin**    | System administrator    | Full system access, manage all resources      |
| **Seller**   | Product vendor/merchant | Manage own products, inventory, and orders    |
| **Customer** | End user/shopper        | Browse products, place orders, manage profile |

---

## 🔐 Authentication

The API uses **JWT (JSON Web Tokens)** for authentication. Include the token in the `Authorization` header:

```http
Authorization: Bearer <your-jwt-token>
```

### Required Headers

- `X-Correlation-ID` - Unique request identifier (mandatory for all requests)
- `X-Seller-ID` - Seller identifier (required for public product browsing)
- `Authorization` - JWT token (required for authenticated endpoints)

---

## 🤝 Contributing

We welcome contributions! To maintain code quality, please:

1. Read [CODING_STANDARDS.md](./CODING_STANDARDS.md) before making changes
2. Follow the established architecture patterns in [ARCHITECTURE.md](./ARCHITECTURE.md)
3. Write integration tests for all new endpoints
4. Ensure all tests pass before submitting PR
5. Use conventional commits for your commit messages

### Development Tools

Install recommended code formatting tools:

```bash
go install mvdan.cc/gofumpt@latest
go install github.com/segmentio/golines@latest
```

---

## 📄 License

This project is proprietary and confidential. All rights reserved.

---

## 📞 Support

For questions or issues:

- Create an issue in the GitHub repository
- Review the documentation in `ARCHITECTURE.md` and `CODING_STANDARDS.md`
- Check the Postman collection for API examples

---

**Built with ❤️ for modern e-commerce**
