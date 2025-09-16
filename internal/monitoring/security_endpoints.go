package monitoring

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/security"
)

// SecurityEndpoints provides HTTP endpoints for security monitoring
type SecurityEndpoints struct {
	securityService *security.SecurityService
}

// NewSecurityEndpoints creates new security endpoints
func NewSecurityEndpoints(securityService *security.SecurityService) *SecurityEndpoints {
	return &SecurityEndpoints{
		securityService: securityService,
	}
}

// SecurityStatsHandler handles security statistics requests
func (se *SecurityEndpoints) SecurityStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	stats := se.securityService.GetStats()
	json.NewEncoder(w).Encode(stats)
}

// BlockedIPsHandler handles blocked IPs requests
func (se *SecurityEndpoints) BlockedIPsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	blockedIPs := se.securityService.GetBlockedIPs()
	json.NewEncoder(w).Encode(blockedIPs)
}

// BlockIPHandler handles IP blocking requests
func (se *SecurityEndpoints) BlockIPHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		IP     string `json:"ip"`
		Reason string `json:"reason"`
		Hours  int    `json:"hours,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.IP == "" {
		http.Error(w, "IP address required", http.StatusBadRequest)
		return
	}

	if request.Reason == "" {
		request.Reason = "Manual block"
	}

	duration := time.Duration(request.Hours) * time.Hour
	if duration == 0 {
		duration = time.Hour // Default 1 hour
	}

	if err := se.securityService.BlockIP(r.Context(), request.IP, request.Reason, duration); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "IP blocked successfully",
	})
}

// UnblockIPHandler handles IP unblocking requests
func (se *SecurityEndpoints) UnblockIPHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		IP string `json:"ip"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.IP == "" {
		http.Error(w, "IP address required", http.StatusBadRequest)
		return
	}

	if err := se.securityService.UnblockIP(r.Context(), request.IP); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "IP unblocked successfully",
	})
}

// RegisterRoutes registers security monitoring routes
func (se *SecurityEndpoints) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/security/stats", se.SecurityStatsHandler)
	mux.HandleFunc("/security/blocked-ips", se.BlockedIPsHandler)
	mux.HandleFunc("/security/block-ip", se.BlockIPHandler)
	mux.HandleFunc("/security/unblock-ip", se.UnblockIPHandler)
}
