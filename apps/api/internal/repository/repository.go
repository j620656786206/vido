package repository

import (
	"context"
	"database/sql"
)

// Repository defines the base interface for data access operations
// All specific repositories should implement this interface with their entity type
type Repository[T any] interface {
	// Create inserts a new entity into the database
	Create(ctx context.Context, entity *T) error

	// FindByID retrieves an entity by its primary key
	FindByID(ctx context.Context, id string) (*T, error)

	// Update modifies an existing entity in the database
	Update(ctx context.Context, entity *T) error

	// Delete removes an entity from the database by ID
	Delete(ctx context.Context, id string) error

	// List retrieves entities with pagination support
	List(ctx context.Context, params ListParams) ([]T, *PaginationResult, error)
}

// ListParams contains parameters for listing/querying entities
type ListParams struct {
	// Pagination
	Page     int // 1-indexed page number
	PageSize int // Number of items per page

	// Sorting
	SortBy    string // Field to sort by
	SortOrder string // "asc" or "desc"

	// Filtering (implementation-specific)
	Filters map[string]interface{}
}

// DefaultPageSize is the default number of items per page
const DefaultPageSize = 20

// MaxPageSize is the maximum allowed page size to prevent excessive queries
const MaxPageSize = 100

// NewListParams creates a new ListParams with sensible defaults
func NewListParams() ListParams {
	return ListParams{
		Page:      1,
		PageSize:  DefaultPageSize,
		SortOrder: "desc",
		Filters:   make(map[string]interface{}),
	}
}

// Validate ensures ListParams have valid values
func (p *ListParams) Validate() {
	if p.Page < 1 {
		p.Page = 1
	}

	if p.PageSize < 1 {
		p.PageSize = DefaultPageSize
	}

	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}

	if p.SortOrder != "asc" && p.SortOrder != "desc" {
		p.SortOrder = "desc"
	}

	if p.Filters == nil {
		p.Filters = make(map[string]interface{})
	}
}

// Offset calculates the SQL OFFSET value for pagination
func (p *ListParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the SQL LIMIT value for pagination
func (p *ListParams) Limit() int {
	return p.PageSize
}

// PaginationResult contains pagination metadata returned with list queries
type PaginationResult struct {
	Page         int `json:"page"`         // Current page number (1-indexed)
	PageSize     int `json:"pageSize"`     // Number of items per page
	TotalResults int `json:"totalResults"` // Total number of items across all pages
	TotalPages   int `json:"totalPages"`   // Total number of pages
}

// NewPaginationResult creates a PaginationResult from params and total count
func NewPaginationResult(params ListParams, totalResults int) *PaginationResult {
	totalPages := totalResults / params.PageSize
	if totalResults%params.PageSize > 0 {
		totalPages++
	}

	return &PaginationResult{
		Page:         params.Page,
		PageSize:     params.PageSize,
		TotalResults: totalResults,
		TotalPages:   totalPages,
	}
}

// Transactor defines the interface for types that support database transactions
type Transactor interface {
	// BeginTx starts a new database transaction
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// TransactionFunc is a function that executes within a transaction
type TransactionFunc func(tx *sql.Tx) error

// WithTransaction executes a function within a database transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func WithTransaction(ctx context.Context, db Transactor, fn TransactionFunc) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
