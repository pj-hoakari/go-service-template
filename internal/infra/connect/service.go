package connect

import (
	"context"
	"errors"

	connectrpc "connectrpc.com/connect"

	greetv1 "github.com/pj-hoakari/go-service-template/gen/greet/v1"
	"github.com/pj-hoakari/go-service-template/gen/greet/v1/greetv1connect"
	"github.com/pj-hoakari/go-service-template/internal/application"
	"github.com/pj-hoakari/go-service-template/internal/domain"
)

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

		return nil, connectrpc.NewError(connectrpc.CodeInternal, err)
	}

	return connectrpc.NewResponse(&greetv1.GreetResponse{
		Greeting: greeting.Message(),
	}), nil
}
