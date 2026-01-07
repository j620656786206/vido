# API Documentation

This directory contains the OpenAPI/Swagger documentation for the Vido API.

## Files

- `swagger.json` - OpenAPI 2.0 specification in JSON format
- `swagger.yaml` - OpenAPI 2.0 specification in YAML format
- `docs.go` - Generated Go package for embedding the spec in the application

## Accessing the Documentation

Once the server is running, you can access the Swagger UI at:

```
http://localhost:8080/swagger/index.html
```

The raw OpenAPI spec is available at:

```
http://localhost:8080/swagger/doc.json
```

## Regenerating the Documentation

After modifying API endpoint annotations in your Go code, regenerate the spec:

```bash
swag init -g cmd/api/main.go -o docs
```

Or use the Makefile (if available):

```bash
make swagger
```

## Adding Documentation to New Endpoints

To document a new endpoint, add swaggo annotations above the handler function:

```go
// GetUser retrieves a user by ID
// @Summary      Get user by ID
// @Description  Retrieve detailed information about a specific user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  User
// @Failure      404  {object}  middleware.ErrorResponse
// @Failure      500  {object}  middleware.ErrorResponse
// @Router       /api/v1/users/{id} [get]
// @Security     BearerAuth
func (s *Server) getUser(c *gin.Context) {
    // Handler implementation
}
```

## Annotation Reference

Common annotations:

- `@Summary` - Short description (1 line)
- `@Description` - Detailed description
- `@Tags` - Group endpoints together
- `@Accept` - Accepted content types (json, xml, etc.)
- `@Produce` - Response content types
- `@Param` - Request parameters (path, query, header, body)
- `@Success` - Success response
- `@Failure` - Error responses
- `@Router` - Route path and HTTP method
- `@Security` - Authentication requirements

For more information, see the [swaggo documentation](https://github.com/swaggo/swag).
