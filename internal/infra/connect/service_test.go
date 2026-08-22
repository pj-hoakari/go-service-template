package connect

import (
	"context"
	"errors"
	"strings"
	"testing"

	connectrpc "connectrpc.com/connect"

	greetv1 "github.com/pj-hoakari/go-service-template/gen/greet/v1"
	"github.com/pj-hoakari/go-service-template/internal/application"
	"github.com/pj-hoakari/go-service-template/internal/domain"
)

func TestServiceGreet(t *testing.T) {
	t.Parallel()

	service := NewService(application.NewGreetService())

	res, err := service.Greet(context.Background(), connectrpc.NewRequest(&greetv1.GreetRequest{Name: "Ada"}))
	if err != nil {
		t.Fatalf("Greet() error = %v", err)
	}

	if got, want := res.Msg.GetGreeting(), "Hello, Ada!"; got != want {
		t.Errorf("Greeting = %q, want %q", got, want)
	}
}

func TestServiceGreetRejectsMissingName(t *testing.T) {
	t.Parallel()

	service := NewService(application.NewGreetService())

	_, err := service.Greet(context.Background(), connectrpc.NewRequest(&greetv1.GreetRequest{}))
	if got, want := connectrpc.CodeOf(err), connectrpc.CodeInvalidArgument; got != want {
		t.Fatalf("Greet() error code = %v, want %v", got, want)
	}
}

// TestInternalErrorHidesDetail keeps the cause of an internal failure out of
// the response: the client only ever sees the fixed message.
func TestInternalErrorHidesDetail(t *testing.T) {
	t.Parallel()

	service := NewService(failingGreetService{err: errors.New("secret detail")})

	_, err := service.Greet(context.Background(), connectrpc.NewRequest(&greetv1.GreetRequest{Name: "Ada"}))
	if got, want := connectrpc.CodeOf(err), connectrpc.CodeInternal; got != want {
		t.Fatalf("Greet() error code = %v, want %v", got, want)
	}

	var connectErr *connectrpc.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("Greet() error = %v, want a Connect error", err)
	}

	if got, want := connectErr.Message(), "internal error"; got != want {
		t.Errorf("Message() = %q, want %q", got, want)
	}

	if strings.Contains(err.Error(), "secret detail") {
		t.Errorf("error = %q, want it to omit the underlying failure", err)
	}
}

// failingGreetService is a GreetUseCases whose every call fails with err.
type failingGreetService struct {
	err error
}

func (s failingGreetService) Greet(context.Context, application.GreetInput) (domain.Greeting, error) {
	return domain.Greeting{}, s.err
}
