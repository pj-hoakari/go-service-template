package connect

import (
	"context"
	"errors"
	"log"

	connectrpc "connectrpc.com/connect"

	greetv1 "github.com/pj-hoakari/go-service-template/gen/greet/v1"
	"github.com/pj-hoakari/go-service-template/gen/greet/v1/greetv1connect"
	"github.com/pj-hoakari/go-service-template/internal/application"
	"github.com/pj-hoakari/go-service-template/internal/domain"
)

// errInternal is the only detail a client learns about an internal failure.
var errInternal = errors.New("internal error")

// internalError reports a failure the client can do nothing about. The cause
// is written to the server log and replaced by a fixed message, so that no
// internal detail leaves the service.
func internalError(err error) *connectrpc.Error {
	log.Printf("go-service-template: internal error: %v", err)

	return connectrpc.NewError(connectrpc.CodeInternal, errInternal)
}

// InternalError exposes internalError to the other transports of this process,
// so that every service answers an internal failure the same way.
func InternalError(err error) *connectrpc.Error {
	return internalError(err)
}

// Service is the Connect transport implementation of GreetService.
type Service struct {
	greetv1connect.UnimplementedGreetServiceHandler
	greetService application.GreetUseCases
}

func NewService(greetService application.GreetUseCases) *Service {
	return &Service{
		UnimplementedGreetServiceHandler: greetv1connect.UnimplementedGreetServiceHandler{},
		greetService:                     greetService,
	}
}

func (s *Service) Greet(ctx context.Context, req *connectrpc.Request[greetv1.GreetRequest]) (*connectrpc.Response[greetv1.GreetResponse], error) {
	greeting, err := s.greetService.Greet(ctx, application.GreetInput{
		Name: req.Msg.GetName(),
	})
	if err != nil {
		if errors.Is(err, domain.ErrGreetingNameRequired) {
			return nil, connectrpc.NewError(connectrpc.CodeInvalidArgument, err)
		}

		return nil, internalError(err)
	}

	return connectrpc.NewResponse(&greetv1.GreetResponse{
		Greeting: greeting.Message(),
	}), nil
}
