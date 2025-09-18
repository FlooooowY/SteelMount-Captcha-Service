package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/logger"
	pb "github.com/FlooooowY/SteelMount-Captcha-Service/proto/balancer/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// BalancerClient handles communication with the balancer
type BalancerClient struct {
	client        pb.BalancerServiceClient
	conn          *grpc.ClientConn
	instanceID    string
	host          string
	port          int
	challengeType string
	logger        *logrus.Logger
}

// NewBalancerClient creates a new balancer client
func NewBalancerClient(balancerURL, instanceID, host, challengeType string, port int) (*BalancerClient, error) {
	// Create gRPC connection
	conn, err := grpc.NewClient(balancerURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to balancer: %w", err)
	}

	client := pb.NewBalancerServiceClient(conn)
	log := logger.GetLogger()

	return &BalancerClient{
		client:        client,
		conn:          conn,
		instanceID:    instanceID,
		host:          host,
		port:          port,
		challengeType: challengeType,
		logger:        log,
	}, nil
}

// StartRegistration starts the registration process with the balancer
func (c *BalancerClient) StartRegistration(ctx context.Context) error {
	// Create stream
	stream, err := c.client.RegisterInstance(ctx)
	if err != nil {
		return fmt.Errorf("failed to create registration stream: %w", err)
	}

	// Send initial registration
	if err := c.sendRegistration(stream, pb.RegisterInstanceRequest_READY); err != nil {
		return fmt.Errorf("failed to send initial registration: %w", err)
	}

	// Start heartbeat loop
	go c.heartbeatLoop(ctx, stream)

	// Start response handler
	go c.handleResponses(ctx, stream)

	return nil
}

// StopRegistration stops the registration process
func (c *BalancerClient) StopRegistration(ctx context.Context) error {
	// Create stream for final registration
	stream, err := c.client.RegisterInstance(ctx)
	if err != nil {
		return fmt.Errorf("failed to create stop stream: %w", err)
	}

	// Send STOPPED event
	if err := c.sendRegistration(stream, pb.RegisterInstanceRequest_STOPPED); err != nil {
		return fmt.Errorf("failed to send stop registration: %w", err)
	}

	// Close stream
	return stream.CloseSend()
}

// Close closes the client connection
func (c *BalancerClient) Close() error {
	return c.conn.Close()
}

// sendRegistration sends a registration request
func (c *BalancerClient) sendRegistration(stream pb.BalancerService_RegisterInstanceClient, eventType pb.RegisterInstanceRequest_EventType) error {
	req := &pb.RegisterInstanceRequest{
		EventType:     eventType,
		InstanceId:    c.instanceID,
		ChallengeType: c.challengeType,
		Host:          c.host,
		PortNumber:    int32(c.port),
		Timestamp:     time.Now().Unix(),
	}

	return stream.Send(req)
}

// heartbeatLoop sends periodic heartbeat messages
func (c *BalancerClient) heartbeatLoop(ctx context.Context, stream pb.BalancerService_RegisterInstanceClient) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Heartbeat loop stopped")
			return
		case <-ticker.C:
			if err := c.sendRegistration(stream, pb.RegisterInstanceRequest_READY); err != nil {
				c.logger.Errorf("Failed to send heartbeat: %v", err)
			}
		}
	}
}

// handleResponses handles responses from the balancer
func (c *BalancerClient) handleResponses(ctx context.Context, stream pb.BalancerService_RegisterInstanceClient) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Response handler stopped")
			return
		default:
			resp, err := stream.Recv()
			if err != nil {
				c.logger.Errorf("Failed to receive response: %v", err)
				return
			}

			c.handleResponse(resp)
		}
	}
}

// handleResponse handles a single response from the balancer
func (c *BalancerClient) handleResponse(resp *pb.RegisterInstanceResponse) {
	switch resp.Status {
	case pb.RegisterInstanceResponse_SUCCESS:
		c.logger.Debug("Registration successful")
	case pb.RegisterInstanceResponse_ERROR:
		c.logger.Errorf("Registration error: %s", resp.Message)
	default:
		c.logger.Warnf("Unknown registration status: %v", resp.Status)
	}
}
