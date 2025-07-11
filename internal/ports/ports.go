package ports

import (
	"context"
	"tezos-delegation/internal/model"
)

// Repository Ports

// DelegationRepositoryPort defines the contract for delegation data persistence
type DelegationRepositoryPort interface {
	InsertDelegations(delegations []*model.Delegation) error
	GetLatestTzktID(ctx context.Context) (int64, error)
	ListDelegations(ctx context.Context, limit, offset int, year *int) ([]model.Delegation, error)
}

// Service Ports

// DelegationServicePort defines the contract for delegation business logic
type DelegationServicePort interface {
	GetDelegations(ctx context.Context, pageNo, pageSize int, year *int) ([]model.Delegation, error)
}

// PollerServicePort defines the contract for the data polling service
type PollerServicePort interface {
	Start(ctx context.Context)
	Wait()
}

// Handler Ports

// DelegationHandlerPort defines the contract for delegation HTTP handlers
type DelegationHandlerPort interface {
	GetDelegations(ctx interface{}) // Using interface{} to be framework-agnostic
}

// Infrastructure Ports

// DatabasePort defines the contract for database connections
type DatabasePort interface {
	Close() error
}

// HTTPClientPort defines the contract for HTTP clients
type HTTPClientPort interface {
	Do(req interface{}) (interface{}, error)
}
