package ecs

import "errors"

var (
	// ErrInstanceNotFoundByID defines an error for when the instance with the provided provider ID is missing.
	ErrInstanceNotFoundByID = errors.New("failed to find instance by id")

	// ErrShowInstance defines an error for when ECS SDK returns error when showing instances.
	ErrShowInstance = errors.New("failed to show instance by id")
)
