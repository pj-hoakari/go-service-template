package connect

import (
	"bytes"
	"context"
	"errors"
	"log"
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
// the response and puts it in the server log instead. It cannot run in
// parallel: it reads the log back through the standard library's process-wide
// logger.
func TestInternalErrorHidesDetail(t *testing.T) {
	logs := captureLog(t)
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

	if got, want := logs.String(), "go-service-template: internal error: secret detail"; !strings.Contains(got, want) {
		t.Errorf("log = %q, want it to contain %q", got, want)
	}
}

// TestCanceledRequestIsNotLogged pins a client that goes away to canceled: it
// is not a server fault, so nothing is logged. It shares the process-wide
// logger with TestInternalErrorHidesDetail and so cannot run in parallel
// either.
func TestCanceledRequestIsNotLogged(t *testing.T) {
	logs := captureLog(t)
	service := NewService(failingGreetService{err: context.Canceled})

	_, err := service.Greet(context.Background(), connectrpc.NewRequest(&greetv1.GreetRequest{Name: "Ada"}))
	if got, want := connectrpc.CodeOf(err), connectrpc.CodeCanceled; got != want {
		t.Fatalf("Greet() error code = %v, want %v", got, want)
	}

	if logs.Len() != 0 {
		t.Errorf("log = %q, want nothing logged", logs.String())
	}
}

// captureLog redirects the standard library's global logger into a buffer for
// the duration of the test. The logger is process-wide, so a test that calls
// this one must not call t.Parallel().
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()

	var logs bytes.Buffer

	previous := log.Writer()

	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previous) })

	return &logs
}

// failingGreetService is a GreetUseCases whose every call fails with err.
type failingGreetService struct {
	err error
}

func (s failingGreetService) Greet(context.Context, application.GreetInput) (domain.Greeting, error) {
	return domain.Greeting{}, s.err
}
