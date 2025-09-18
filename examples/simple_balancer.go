package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	pb "github.com/FlooooowY/SteelMount-Captcha-Service/proto/balancer/v1"
	captchaPb "github.com/FlooooowY/SteelMount-Captcha-Service/proto/captcha/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SimpleBalancer demonstrates how to integrate with captcha service
type SimpleBalancer struct {
	pb.UnimplementedBalancerServiceServer

	instances map[string]*CaptchaInstance
	mutex     sync.RWMutex
}

// CaptchaInstance represents a registered captcha service instance
type CaptchaInstance struct {
	ID            string
	Host          string
	Port          int32
	ChallengeType string
	LastHeartbeat time.Time
	Connection    *grpc.ClientConn
	Client        captchaPb.CaptchaServiceClient
}

// NewSimpleBalancer creates a new balancer instance
func NewSimpleBalancer() *SimpleBalancer {
	return &SimpleBalancer{
		instances: make(map[string]*CaptchaInstance),
	}
}

// RegisterInstance handles captcha service registration
func (b *SimpleBalancer) RegisterInstance(stream pb.BalancerService_RegisterInstanceServer) error {
	log.Println("New registration stream opened")

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			log.Println("Registration stream closed")
			return nil
		}
		if err != nil {
			log.Printf("Error receiving registration: %v", err)
			return err
		}

		b.mutex.Lock()

		switch req.EventType {
		case pb.RegisterInstanceRequest_READY:
			log.Printf("Registering instance: %s at %s:%d", req.InstanceId, req.Host, req.PortNumber)

			// Create connection to captcha service
			conn, err := grpc.NewClient(
				fmt.Sprintf("%s:%d", req.Host, req.PortNumber),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				log.Printf("Failed to connect to instance %s: %v", req.InstanceId, err)
				b.mutex.Unlock()
				continue
			}

			instance := &CaptchaInstance{
				ID:            req.InstanceId,
				Host:          req.Host,
				Port:          req.PortNumber,
				ChallengeType: req.ChallengeType,
				LastHeartbeat: time.Now(),
				Connection:    conn,
				Client:        captchaPb.NewCaptchaServiceClient(conn),
			}

			b.instances[req.InstanceId] = instance

			// Send success response
			err = stream.Send(&pb.RegisterInstanceResponse{
				Status:  pb.RegisterInstanceResponse_SUCCESS,
				Message: "Instance registered successfully",
			})
			if err != nil {
				log.Printf("Failed to send registration response: %v", err)
			}

		case pb.RegisterInstanceRequest_STOPPED:
			log.Printf("Instance stopping: %s", req.InstanceId)

			if instance, exists := b.instances[req.InstanceId]; exists {
				if instance.Connection != nil {
					instance.Connection.Close()
				}
				delete(b.instances, req.InstanceId)
			}

		default:
			// Update heartbeat for existing instances
			if instance, exists := b.instances[req.InstanceId]; exists {
				instance.LastHeartbeat = time.Now()
			}
		}

		b.mutex.Unlock()
	}
}

// GetChallenge requests a new challenge from available instances
func (b *SimpleBalancer) GetChallenge(ctx context.Context, complexity int32) (*captchaPb.ChallengeResponse, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	// Simple round-robin selection (in real implementation, use better load balancing)
	for _, instance := range b.instances {
		if time.Since(instance.LastHeartbeat) > 60*time.Second {
			continue // Skip stale instances
		}

		// Request challenge from instance
		req := &captchaPb.ChallengeRequest{
			Complexity: complexity,
		}

		resp, err := instance.Client.NewChallenge(ctx, req)
		if err != nil {
			log.Printf("Failed to get challenge from instance %s: %v", instance.ID, err)
			continue
		}

		log.Printf("Challenge created: %s from instance %s", resp.ChallengeId, instance.ID)
		return resp, nil
	}

	return nil, fmt.Errorf("no available captcha instances")
}

// GetInstanceStats returns statistics about registered instances
func (b *SimpleBalancer) GetInstanceStats() map[string]interface{} {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_instances": len(b.instances),
		"instances":       make([]map[string]interface{}, 0),
	}

	for _, instance := range b.instances {
		instanceInfo := map[string]interface{}{
			"id":             instance.ID,
			"host":           instance.Host,
			"port":           instance.Port,
			"challenge_type": instance.ChallengeType,
			"last_heartbeat": instance.LastHeartbeat,
			"healthy":        time.Since(instance.LastHeartbeat) < 60*time.Second,
		}
		stats["instances"] = append(stats["instances"].([]map[string]interface{}), instanceInfo)
	}

	return stats
}

func main() {
	// Create balancer
	balancer := NewSimpleBalancer()

	// Start gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterBalancerServiceServer(s, balancer)

	log.Println("Simple Balancer started on :50051")

	// Start cleanup routine for stale instances
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			balancer.mutex.Lock()

			for id, instance := range balancer.instances {
				if time.Since(instance.LastHeartbeat) > 120*time.Second {
					log.Printf("Removing stale instance: %s", id)
					if instance.Connection != nil {
						instance.Connection.Close()
					}
					delete(balancer.instances, id)
				}
			}

			balancer.mutex.Unlock()
		}
	}()

	// Example: periodically request challenges to test integration
	go func() {
		time.Sleep(10 * time.Second) // Wait for instances to register

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			challenge, err := balancer.GetChallenge(ctx, 50)
			if err != nil {
				log.Printf("Failed to get challenge: %v", err)
			} else {
				log.Printf("Successfully got challenge: %s (HTML length: %d bytes)",
					challenge.ChallengeId, len(challenge.Html))
			}

			cancel()
		}
	}()

	// Print stats periodically
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := balancer.GetInstanceStats()
			log.Printf("Balancer stats: %+v", stats)
		}
	}()

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
