package connect

import (
	"context"
	"testing"

	connectrpc "connectrpc.com/connect"

	greetv1 "github.com/pj-hoakari/go-service-template/gen/greet/v1"
	"github.com/pj-hoakari/go-service-template/internal/application"
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
