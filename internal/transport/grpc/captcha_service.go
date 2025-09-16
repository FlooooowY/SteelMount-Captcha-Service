package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/domain"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/usecase"
	pb "github.com/FlooooowY/SteelMount-Captcha-Service/pb/proto/captcha/v1"
)

// CaptchaService implements the gRPC captcha service
type CaptchaService struct {
	pb.UnimplementedCaptchaServiceServer
	captchaUsecase usecase.CaptchaUsecase
}

// NewCaptchaService creates a new captcha service
func NewCaptchaService(captchaUsecase usecase.CaptchaUsecase) *CaptchaService {
	return &CaptchaService{
		captchaUsecase: captchaUsecase,
	}
}

// NewChallenge creates a new captcha challenge
func (s *CaptchaService) NewChallenge(ctx context.Context, req *pb.ChallengeRequest) (*pb.ChallengeResponse, error) {
	// Create challenge using usecase
	challenge, err := s.captchaUsecase.CreateChallenge(ctx, req.Complexity)
	if err != nil {
		return nil, fmt.Errorf("failed to create challenge: %w", err)
	}

	// Return response
	return &pb.ChallengeResponse{
		ChallengeId: challenge.ID,
		Html:        challenge.HTML,
	}, nil
}

// MakeEventStream handles bidirectional event streaming
func (s *CaptchaService) MakeEventStream(stream pb.CaptchaService_MakeEventStreamServer) error {
	ctx := stream.Context()

	for {
		// Receive client event
		clientEvent, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("failed to receive client event: %w", err)
		}

		// Convert to domain event
		domainEvent := &domain.Event{
			Type:        s.convertEventType(clientEvent.EventType),
			ChallengeID: clientEvent.ChallengeId,
			Data:        clientEvent.Data,
			Timestamp:   time.Now(),
		}

		// Process event
		serverEvent, err := s.captchaUsecase.ProcessEvent(ctx, domainEvent)
		if err != nil {
			return fmt.Errorf("failed to process event: %w", err)
		}

		// Send server event
		if err := s.sendServerEvent(stream, serverEvent); err != nil {
			return fmt.Errorf("failed to send server event: %w", err)
		}
	}
}

// convertEventType converts protobuf event type to domain event type
func (s *CaptchaService) convertEventType(eventType pb.ClientEvent_EventType) domain.EventType {
	switch eventType {
	case pb.ClientEvent_FRONTEND_EVENT:
		return domain.EventTypeFrontendEvent
	case pb.ClientEvent_CONNECTION_CLOSED:
		return domain.EventTypeConnectionClosed
	case pb.ClientEvent_BALANCER_EVENT:
		return domain.EventTypeBalancerEvent
	default:
		return domain.EventTypeFrontendEvent
	}
}

// sendServerEvent sends a server event to the client
func (s *CaptchaService) sendServerEvent(stream pb.CaptchaService_MakeEventStreamServer, event *domain.ServerEvent) error {
	serverEvent := &pb.ServerEvent{}

	switch event.Type {
	case domain.ServerEventTypeChallengeResult:
		serverEvent.Event = &pb.ServerEvent_Result{
			Result: &pb.ServerEvent_ChallengeResult{
				ChallengeId:      event.ChallengeID,
				ConfidencePercent: 100, // TODO: Get from event data
			},
		}
	case domain.ServerEventTypeRunClientJS:
		serverEvent.Event = &pb.ServerEvent_ClientJs{
			ClientJs: &pb.ServerEvent_RunClientJS{
				ChallengeId: event.ChallengeID,
				JsCode:      event.JSCode,
			},
		}
	case domain.ServerEventTypeSendClientData:
		serverEvent.Event = &pb.ServerEvent_ClientData{
			ClientData: &pb.ServerEvent_SendClientData{
				ChallengeId: event.ChallengeID,
				Data:        event.Data,
			},
		}
	default:
		return fmt.Errorf("unknown server event type: %s", event.Type)
	}

	return stream.Send(serverEvent)
}
