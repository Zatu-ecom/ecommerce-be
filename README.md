# Datun E-commerce Backend

This project implements the backend API for the Datun e-commerce platform as specified in the PRD document.

## Completed Features (up to PRD section 3.3)

### Authentication APIs
- User Registration: `POST /api/auth/register`
- User Login: `POST /api/auth/login`
- Token Refresh: `POST /api/auth/refresh`
- User Logout: `POST /api/auth/logout`

### User Management APIs
- Get User Profile: `GET /api/users/profile`
- Update User Profile: `PUT /api/users/profile`
- Get User Addresses: `GET /api/users/addresses`
- Add Address: `POST /api/users/addresses`
- Update Address: `PUT /api/users/addresses/:id`
- Delete Address: `DELETE /api/users/addresses/:id`
- Set Default Address: `PATCH /api/users/addresses/:id/default`
- Change Password: `PATCH /api/users/password`

## Project Structure

The project follows a modular structure to support future microservices:

```
datun-site-db/
├── common/                 # Shared utilities and middleware
│   ├── db.go               # Database connection
│   ├── auth.go             # JWT authentication
│   ├── response.go         # API response helpers
│   └── middleware/         # HTTP middleware
│       └── middleware.go   # CORS, logging, auth middleware
├── user/                   # User module (microservice)
│   ├── module.go           # Module registration
│   ├── entity/             # Database models
│   │   ├── user.go
│   │   └── address.go
│   ├── model/              # API request/response models
│   │   ├── user_model.go
│   │   └── address_model.go
│   ├── repositories/       # Data access layer
│   │   ├── user_repository.go
│   │   ├── user_repository_impl.go
│   │   ├── address_repository.go
│   │   └── address_repository_impl.go
│   ├── service/            # Business logic layer
│   │   ├── user_service.go
│   │   ├── user_service_impl.go
│   │   ├── address_service.go
│   │   └── address_service_impl.go
│   ├── handlers/           # HTTP handlers
│   │   ├── user_handler.go
│   │   └── address_handler.go
│   └── routes/             # API route definitions
│       └── routes.go
├── main.go                 # Application entry point
└── go.mod                  # Go module definition
```

## Design Patterns Used

1. **Repository Pattern**: Separates data access logic from business logic
2. **Service Layer**: Contains business logic and orchestrates operations
3. **Dependency Injection**: Services and repositories are injected into handlers
4. **Interface-based Design**: Components interact through interfaces for loose coupling

## SOLID Principles Implementation

1. **Single Responsibility Principle**: Each component has a single responsibility
   - Repositories handle data access
   - Services handle business logic
   - Handlers handle HTTP requests/responses
   
2. **Open/Closed Principle**: Components can be extended without modification
   - New repository implementations can be created without changing existing code
   
3. **Liskov Substitution Principle**: Interface implementations are interchangeable
   - Repository implementations can be swapped without affecting services
   
4. **Interface Segregation Principle**: Small, focused interfaces
   - Each repository and service has its own specific interface
   
5. **Dependency Inversion Principle**: High-level modules depend on abstractions
   - Services depend on repository interfaces, not concrete implementations

## Running the Application

1. Ensure PostgreSQL is installed and running
2. Update the `.env` file with your database credentials
3. Run the application: `go run main.go`

## Next Steps (Phase 2)

The next phase will implement from section 3.3 of the PRD, including:
- Product APIs
- Cart APIs
- Order APIs
- Payment Integration
- And more as specified in the PRD
