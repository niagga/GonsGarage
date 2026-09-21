# GonsGarage Development Guide

## 🎯 Project Overview

**GonsGarage** is a comprehensive auto repair shop management system built with Clean Architecture principles.

### Technology Stack
- **Backend**: Go 1.27+ (see `backend/go.mod`) with Gin framework and middleware
- **Frontend**: Next.js 16 (App Router), React 19, TypeScript, Zustand
- **Database**: PostgreSQL 16+ (local Compose image) with Redis 8+ for caching
- **API Documentation**: Swagger/OpenAPI 3.0 with automated generation
- **Development**: Docker-based local environment
- **Testing**: Test-Driven Development (TDD) — **mandatory for product code** (see [docs/testing-tdd.md](docs/testing-tdd.md) and [CONTRIBUTING.md](CONTRIBUTING.md))
- **Frontend package manager**: **pnpm** (not npm)

---

## 1️⃣ Coding Standards

### Naming Conventions
```go
// Go - Correct (Backend Services - Exported Structs)
type UserService struct {}        // PascalCase for exported structs
type carService struct {}         // camelCase for unexported implementations
func (s *UserService) CreateUser() // PascalCase for exported methods
var userName string               // camelCase for variables
const maxRetries = 3             // camelCase for constants

// Go - Incorrect
type userService struct {}        // ❌ exported should be PascalCase
func (s *UserService) create_user() // ❌ should be camelCase
var UserName string               // ❌ unexported should be camelCase
```

```typescript
// TypeScript - Correct (Frontend with Zustand)
interface UserProps {             // PascalCase for interfaces
  firstName: string;              // camelCase for properties
}

// Zustand Store
interface UserStore {
  users: User[];
  createUser: (user: User) => void;
  fetchUsers: () => Promise<void>;
}

const useUserStore = create<UserStore>((set, get) => ({
  users: [],
  createUser: (user) => set(state => ({ users: [...state.users, user] })),
  fetchUsers: async () => { /* implementation */ }
}));
```

### JSON API Conventions
```go
// Go Structs - camelCase JSON tags
type CreateUserRequest struct {
    FirstName string `json:"firstName"`  // ✅ camelCase
    LastName  string `json:"lastName"`   // ✅ camelCase
    Email     string `json:"email"`
}

// ❌ Incorrect - snake_case
type CreateUserRequest struct {
    FirstName string `json:"first_name"` // ❌ should be camelCase
    LastName  string `json:"last_name"`  // ❌ should be camelCase
}
```

---

## 2️⃣ Architecture Patterns

### Backend Structure (Clean Architecture)
```
backend/
├── cmd/
│   └── api/main.go                 # Application entry point
├── internal/
│   ├── domain/                     # Business entities (plantilla §1)
│   ├── service/                    # Casos de uso (auth, car, appointment, …)
│   ├── handler/                    # HTTP (Gin)
│   ├── middleware/                 # Auth, CORS, etc.
│   ├── repository/                 # postgres | redis | mock
│   ├── core/
│   │   └── ports/                  # Interfaces / contratos
│   └── platform/
│       └── sqlxdb/                 # sqlx sobre el pool compartido (Fase 2)
├── docs/                           # Swagger (swag): docs.go, swagger.json/yaml
├── pkg/                            # Utilidades compartidas
└── tests/                          # Integration tests
```

### Frontend Structure (Next.js + TypeScript + Zustand)
```
frontend/
├── src/
│   ├── app/                       # Next.js App Router (TypeScript)
│   ├── components/                # Reusable UI components
│   ├── stores/                    # ✅ Zustand state management
│   │   ├── auth.store.ts         # Authentication state
│   │   ├── user.store.ts         # User management state  
│   │   ├── car.store.ts          # Car management state
│   │   └── index.ts              # Store exports
│   ├── lib/
│   │   └── api/                  # API clients (TypeScript)
│   ├── types/                    # TypeScript type definitions
│   └── hooks/                    # Custom React hooks
├── __tests__/                    # Jest + React Testing Library
└── package.json
```

---

## 3️⃣ Unified Domain Model

### ✅ User Entity (Single Source of Truth)
```go
// domain/user.go - Unified user entity
type User struct {
    ID        uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Email     string     `json:"email" gorm:"uniqueIndex;not null"`
    Password  string     `json:"-" gorm:"not null"`
    FirstName string     `json:"firstName" gorm:"not null"`
    LastName  string     `json:"lastName" gorm:"not null"`
    Role      UserRole   `json:"role" gorm:"not null;default:'client'"`
    CreatedAt time.Time  `json:"createdAt"`
    UpdatedAt time.Time  `json:"updatedAt"`
    DeletedAt *time.Time `json:"-" gorm:"index"`
}

type UserRole string
const (
    RoleClient      UserRole = "client"
    RoleEmployee    UserRole = "employee"
    RoleAdmin       UserRole = "admin"
)

// Helper methods for role checking
func (u *User) IsClient() bool { return u.Role == RoleClient }
func (u *User) IsEmployee() bool { return u.Role == RoleEmployee }
func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }
func (u *User) CanManageUsers() bool { return u.Role == RoleAdmin }
```

### ✅ Employee Profile (Extension, not separate entity)
```go
// domain/employee.go - Profile for employee users only
type Employee struct {
    ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID     uuid.UUID `json:"userId" gorm:"type:uuid;not null"`
    User       User      `json:"user" gorm:"foreignKey:UserID"`
    Position   string    `json:"position" gorm:"not null"`
    HourlyRate *float64  `json:"hourlyRate"`
    HireDate   time.Time `json:"hireDate" gorm:"not null"`
    CreatedAt  time.Time `json:"createdAt"`
    UpdatedAt  time.Time `json:"updatedAt"`
    DeletedAt  *time.Time `json:"-" gorm:"index"`
}
```

---

## 4️⃣ Service Layer Patterns

### ✅ Service Implementation (Gin-ready)
```go
// services/user/user_service.go
type userService struct {  // ✅ unexported implementation
    userRepo  ports.UserRepository
    logger    *slog.Logger
}

func NewUserService(userRepo ports.UserRepository, logger *slog.Logger) ports.UserService {
    return &userService{
        userRepo: userRepo,
        logger:   logger,
    }
}

func (s *userService) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
    s.logger.Info("creating user", "email", user.Email, "role", user.Role)
    
    // Business logic here
    if err := user.Validate(); err != nil {
        return nil, fmt.Errorf("invalid user data: %w", err)
    }
    
    return s.userRepo.CreateUser(ctx, user)
}
```

### ✅ Interface Definitions
```go
// ports/services.go
type UserService interface {
    CreateUser(ctx context.Context, user *domain.User) (*domain.User, error)
    GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error)
    UpdateUser(ctx context.Context, user *domain.User) (*domain.User, error)
    DeleteUser(ctx context.Context, id uuid.UUID) error
    ListUsersByRole(ctx context.Context, role string) ([]*domain.User, error)
}

type AuthService interface {
    Login(ctx context.Context, email, password string) (*domain.User, string, error)
    Register(ctx context.Context, user *domain.User) (*domain.User, error)
    ValidateToken(ctx context.Context, token string) (*domain.User, error)
}
```

---

## 5️⃣ Gin Framework Patterns

### ✅ Middleware Implementation
```go
// middleware/auth_middleware.go
type AuthMiddleware struct {
    jwtSecret string
}

func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
    return &AuthMiddleware{jwtSecret: jwtSecret}
}

// Gin-native middleware
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
            c.Abort()
            return
        }
        
        // JWT validation logic...
        c.Set("userID", userID)  // ✅ Standard context key
        c.Next()
    }
}
```

### ✅ Handler Implementation with Swagger Documentation
```go
// handlers/user_handler.go
type UserHandler struct {
    userService ports.UserService
}

func NewUserHandler(userService ports.UserService) *UserHandler {
    return &UserHandler{userService: userService}
}

// CreateUser creates a new user
// @Summary Create a new user
// @Description Create a new user in the system (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user body CreateUserRequest true "User data"
// @Success 201 {object} domain.User "User created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }
    
    userID := c.GetString("userID")  // ✅ Get from Gin context
    
    user, err := h.userService.CreateUser(c.Request.Context(), &domain.User{
        FirstName: req.FirstName,
        LastName:  req.LastName,
        Email:     req.Email,
        Role:      domain.UserRole(req.Role),
    })
    
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusCreated, user)
}
```

---

## 6️⃣ Swagger/OpenAPI Documentation Standards

### ✅ API Documentation Structure
```go
// handlers/swagger_types.go - Swagger response types

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
    FirstName string `json:"firstName" binding:"required" example:"John"`
    LastName  string `json:"lastName" binding:"required" example:"Doe"`
    Email     string `json:"email" binding:"required,email" example:"john.doe@example.com"`
    Password  string `json:"password" binding:"required,min=8" example:"securepassword123"`
    Role      string `json:"role" binding:"required" enums:"client,employee,admin" example:"client"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
    Error   string            `json:"error" example:"validation_failed"`
    Message string            `json:"message" example:"Invalid input data"`
    Details map[string]string `json:"details,omitempty"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
    Message string `json:"message" example:"Operation completed successfully"`
}
```

### ✅ Swagger Annotations Guidelines
```go
// ✅ Complete handler documentation
// @Summary Short description of the endpoint
// @Description Detailed description of what the endpoint does
// @Tags group-name
// @Accept json
// @Produce json
// @Security BearerAuth (for protected endpoints)
// @Param paramName path/query/body type required "Description" 
// @Success 200 {object} ResponseType "Success description"
// @Failure 400 {object} ErrorResponse "Error description"
// @Router /api/v1/endpoint [method]

// Example for car endpoints:
// GetCars retrieves user's cars
// @Summary Get user's cars
// @Description Retrieve all cars owned by the authenticated user
// @Tags cars
// @Produce json
// @Security BearerAuth
// @Success 200 {array} domain.Car "List of user's cars"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/cars [get]
func (h *CarHandler) GetCars(c *gin.Context) {
    // Implementation
}
```

### ✅ OpenAPI Configuration
```go
// cmd/api/main.go - Swagger setup
import (
    "github.com/gin-gonic/gin"
    swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
    _ "github.com/your-org/gonsgarage/docs" // Import generated docs
)

// @title GonsGarage API
// @version 1.0
// @description Auto repair shop management system API
// @termsOfService https://gonsgarage.com/terms
// @contact.name API Support
// @contact.url https://gonsgarage.com/support
// @contact.email support@gonsgarage.com
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token
func main() {
    r := gin.Default()
    
    // Swagger endpoint
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
    
    // API routes
    setupRoutes(r)
    
    r.Run(":8080")
}
```

---

## 7️⃣ Frontend Patterns (Next.js + Zustand)

### ✅ Zustand Store Definition
```typescript
// stores/user.store.ts
interface UserState {
  users: User[];
  currentUser: User | null;
  loading: boolean;
  error: string | null;
}

interface UserActions {
  fetchUsers: () => Promise<void>;
  createUser: (userData: CreateUserRequest) => Promise<void>;
  updateUser: (id: string, userData: UpdateUserRequest) => Promise<void>;
  deleteUser: (id: string) => Promise<void>;
  setCurrentUser: (user: User | null) => void;
  clearError: () => void;
}

type UserStore = UserState & UserActions;

export const useUserStore = create<UserStore>((set, get) => ({
  // State
  users: [],
  currentUser: null,
  loading: false,
  error: null,

  // Actions
  fetchUsers: async () => {
    set({ loading: true, error: null });
    try {
      const users = await userApi.getUsers();
      set({ users, loading: false });
    } catch (error) {
      set({ error: error.message, loading: false });
    }
  },

  createUser: async (userData) => {
    set({ loading: true, error: null });
    try {
      const newUser = await userApi.createUser(userData);
      set(state => ({ 
        users: [...state.users, newUser], 
        loading: false 
      }));
    } catch (error) {
      set({ error: error.message, loading: false });
    }
  },

  // ... other actions
}));
```

### ✅ Component Integration
```typescript
// components/UserList.tsx
import { useUserStore } from '@/stores/user.store';

export default function UserList() {
  const { 
    users, 
    loading, 
    error, 
    fetchUsers, 
    createUser,
    clearError 
  } = useUserStore();

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorAlert message={error} onClose={clearError} />;

  return (
    <div>
      {users.map(user => (
        <UserCard key={user.id} user={user} />
      ))}
    </div>
  );
}
```

---

## 8️⃣ Testing Patterns

### ✅ Backend Testing (TDD)
```go
// services/user/user_service_test.go
func TestUserService_CreateUser(t *testing.T) {
    // Arrange
    mockRepo := mocks.NewMockUserRepository(t)
    logger := slog.Default()
    service := NewUserService(mockRepo, logger)
    
    user := &domain.User{
        FirstName: "John",
        LastName:  "Doe", 
        Email:     "john@example.com",
        Role:      domain.RoleClient,
    }
    
    mockRepo.EXPECT().CreateUser(mock.Anything, user).Return(user, nil)
    
    // Act
    result, err := service.CreateUser(context.Background(), user)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, user.Email, result.Email)
    mockRepo.AssertExpectations(t)
}
```

### ✅ Frontend Testing (Jest + RTL)
```typescript
// __tests__/stores/user.store.test.ts
describe('UserStore', () => {
  beforeEach(() => {
    useUserStore.getState().users = [];
  });

  it('should create user successfully', async () => {
    // Arrange
    const mockUser = { id: '1', firstName: 'John', lastName: 'Doe' };
    jest.spyOn(userApi, 'createUser').mockResolvedValue(mockUser);
    
    // Act
    await useUserStore.getState().createUser({
      firstName: 'John',
      lastName: 'Doe',
      email: 'john@example.com'
    });
    
    // Assert
    expect(useUserStore.getState().users).toContain(mockUser);
    expect(useUserStore.getState().loading).toBe(false);
  });
});
```

---

## 9️⃣ API Design Standards

### ✅ RESTful Endpoints
```
POST   /api/v1/auth/login           # Authentication
POST   /api/v1/auth/register        # User registration
GET    /api/v1/users               # List users (admin only)
POST   /api/v1/users               # Create user (admin only)
GET    /api/v1/users/:id           # Get user details
PUT    /api/v1/users/:id           # Update user
DELETE /api/v1/users/:id           # Delete user
GET    /api/v1/cars                # List user's cars
POST   /api/v1/cars               # Create car
GET    /api/v1/employees           # List employees
POST   /api/v1/employees          # Create employee profile
GET    /swagger/index.html         # Swagger UI documentation
GET    /swagger/doc.json          # OpenAPI JSON specification
```

### ✅ Standard Error Responses
```json
{
  "error": "validation_failed",
  "message": "Invalid input data",
  "details": {
    "firstName": "required field",
    "email": "invalid format"
  }
}
```

---

## 🔟 Development Guidelines

### Context Management
- **Backend**: Use `"userID"` as standard context key (camelCase)
- **Frontend**: Use Zustand for global state, React Context for component-scoped state

### Database Patterns
- **Entities**: Unified User entity with roles, avoid redundant entities
- **Migrations**: Use descriptive names, always include rollback
- **Indexing**: Index all foreign keys and frequently queried columns

### Error Handling
- **Backend**: Use structured logging with slog
- **Frontend**: Centralized error handling in Zustand stores
- **API**: Consistent error response format

### Security
- **JWT**: Use secure secrets, implement token refresh
- **Validation**: Validate all inputs at multiple layers
- **Authorization**: Role-based access control with middleware

### API Documentation
- **Swagger**: Use swaggo for automatic documentation generation
- **Annotations**: Document all public endpoints with comprehensive annotations
- **Examples**: Provide request/response examples for all endpoints
- **Security**: Document authentication requirements clearly

---

## 1️⃣1️⃣ Key Architectural Decisions

1. **✅ Unified User Entity**: Single User table with roles instead of separate Client entity
2. **✅ Services over UseCases**: Renamed usecases to services for clarity
3. **✅ Gin Middleware**: Native Gin middleware for authentication and CORS
4. **✅ Zustand over Context**: Zustand for complex state, React Context for simple cases
5. **✅ TypeScript First**: Full TypeScript adoption in frontend
6. **✅ camelCase JSON**: Consistent camelCase for all API JSON fields
7. **✅ Structured Logging**: slog for backend, console.error for frontend development
8. **✅ Swagger/OpenAPI**: Automated API documentation with comprehensive annotations

This guide ensures consistency, maintainability, and scalability across the entire GonsGarage project.