package grpc

import (
	"context"
	"net"
	"strings"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/security"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// SecurityMiddleware provides security checks for gRPC requests
type SecurityMiddleware struct {
	securityService *security.SecurityService
}

// NewSecurityMiddleware creates a new security middleware
func NewSecurityMiddleware(securityService *security.SecurityService) *SecurityMiddleware {
	return &SecurityMiddleware{
		securityService: securityService,
	}
}

// UnaryInterceptor creates a unary interceptor for security checks
func (sm *SecurityMiddleware) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract client information
		ip, userAgent := sm.extractClientInfo(ctx)

		// Perform security checks
		result, err := sm.securityService.CheckRequest(ctx, ip, userAgent, info.FullMethod, 0, false)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "security check failed: %v", err)
		}

		if !result.Allowed {
			return nil, status.Errorf(codes.PermissionDenied, "request blocked: %s", strings.Join(result.Reasons, ", "))
		}

		// Call the actual handler
		return handler(ctx, req)
	}
}

// StreamInterceptor creates a stream interceptor for security checks
func (sm *SecurityMiddleware) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Extract client information
		ip, userAgent := sm.extractClientInfo(ss.Context())

		// Perform security checks
		result, err := sm.securityService.CheckRequest(ss.Context(), ip, userAgent, info.FullMethod, 0, false)
		if err != nil {
			return status.Errorf(codes.Internal, "security check failed: %v", err)
		}

		if !result.Allowed {
			return status.Errorf(codes.PermissionDenied, "request blocked: %s", strings.Join(result.Reasons, ", "))
		}

		// Call the actual handler
		return handler(srv, ss)
	}
}

// extractClientInfo extracts IP address and user agent from gRPC context
func (sm *SecurityMiddleware) extractClientInfo(ctx context.Context) (string, string) {
	// Extract IP address
	ip := "127.0.0.1" // Default fallback
	if p, ok := peer.FromContext(ctx); ok {
		if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
			ip = tcpAddr.IP.String()
		}
	}

	// Extract user agent from metadata
	userAgent := "grpc-client" // Default fallback
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ua := md.Get("user-agent"); len(ua) > 0 {
			userAgent = ua[0]
		}
	}

	return ip, userAgent
}

// BlockIP blocks an IP address
func (sm *SecurityMiddleware) BlockIP(ctx context.Context, ip string, reason string) error {
	return sm.securityService.BlockIP(ctx, ip, reason, 0) // Use default duration
}

// UnblockIP unblocks an IP address
func (sm *SecurityMiddleware) UnblockIP(ctx context.Context, ip string) error {
	return sm.securityService.UnblockIP(ctx, ip)
}

// GetSecurityStats returns security statistics
func (sm *SecurityMiddleware) GetSecurityStats() map[string]interface{} {
	return sm.securityService.GetStats()
}
