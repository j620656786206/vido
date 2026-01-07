# Middleware Package

This package contains all HTTP middleware used in the Vido API server.

## Error Handling

The error handling middleware provides consistent error responses and panic recovery.

### Error Response Format

All errors are returned in a consistent JSON format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message"
  }
}
```

### Custom Error Types

The package provides several built-in error constructors:

- **ValidationError** (400 Bad Request): For invalid input
  ```go
  middleware.NewValidationError("Invalid email format")
  ```

- **NotFoundError** (404 Not Found): For missing resources
  ```go
  middleware.NewNotFoundError("User not found")
  ```

- **UnauthorizedError** (401 Unauthorized): For authentication failures
  ```go
  middleware.NewUnauthorizedError("Authentication required")
  ```

- **ForbiddenError** (403 Forbidden): For authorization failures
  ```go
  middleware.NewForbiddenError("Access denied")
  ```

- **InternalError** (500 Internal Server Error): For server errors
  ```go
  middleware.NewInternalError("Database error", err)
  ```

### Usage in Handlers

To use error handling in your route handlers:

```go
func (s *Server) getUser(c *gin.Context) {
    userID := c.Param("id")

    user, err := s.userService.GetByID(userID)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            c.Error(middleware.NewNotFoundError("User not found"))
            return
        }
        c.Error(middleware.NewInternalError("Failed to retrieve user", err))
        return
    }

    c.JSON(http.StatusOK, user)
}
```

### Panic Recovery

The Recovery middleware automatically catches panics, logs them with stack traces, and returns a consistent error response:

```go
router.Use(middleware.Recovery(cfg))
```

### Error Logging

Errors are automatically logged with:
- Request ID
- Request path and method
- Error details
- Stack trace (in development mode only)

### Testing

See `errors_test.go` for comprehensive test coverage of all error types and scenarios.
