// Package service implements business logic for the Agent Task Center.
package service

import "errors"

// Sentinel errors used across the service layer.
// Handlers use errors.Is() to match these and map them to HTTP status codes.
var (
	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrNoAvailableTask indicates no unclaimed task with met dependencies is available.
	ErrNoAvailableTask = errors.New("no available task")

	// ErrVersionConflict indicates an optimistic locking CAS failure.
	ErrVersionConflict = errors.New("version conflict")

	// ErrInvalidFile indicates the uploaded file has an unsupported format or invalid content.
	ErrInvalidFile = errors.New("invalid file format")

	// ErrUnauthorizedAgent indicates an agent attempted to modify a task claimed by a different agent.
	ErrUnauthorizedAgent = errors.New("task claimed by different agent")

	// ErrInvalidStatus indicates an invalid status value was provided.
	ErrInvalidStatus = errors.New("invalid status")
)
