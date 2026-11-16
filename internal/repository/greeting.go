// Package repository defines persistence contracts for the greet context.
package repository

import (
	"context"

	"github.com/pj-hoakari/go-service-template/internal/domain"
)

// GreetingRepository persists issued greetings.
type GreetingRepository interface {
	Record(context.Context, domain.Greeting) error
}
