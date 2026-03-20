package httpapi

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/arsen/fleet-reservation/reservation/internal/core/domain"
	"github.com/arsen/fleet-reservation/reservation/internal/core/ports"
	"github.com/google/uuid"
)

//go:embed ui.html
var uiHTML []byte

// HttpAdapter serves the web UI and a REST API that bridges to the core application.
type HttpAdapter struct {
	app  ports.CoreApplicationPort
	port int
}

func NewHttpAdapter(app ports.CoreApplicationPort, port int) *HttpAdapter {
	return &HttpAdapter{app: app, port: port}
}

func (h *HttpAdapter) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.route)
	addr := fmt.Sprintf(":%d", h.port)
	log.Printf("HTTP UI server listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}

func (h *HttpAdapter) route(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case (path == "/" || path == "/index.html") && r.Method == http.MethodGet:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(uiHTML) //nolint:errcheck
	case path == "/api/resources" && r.Method == http.MethodGet:
		h.listResources(w, r)
	case path == "/api/reservations" && r.Method == http.MethodGet:
		h.listReservations(w, r)
	case path == "/api/reservations" && r.Method == http.MethodPost:
		h.createReservation(w, r)
	case strings.HasPrefix(path, "/api/reservations/") &&
		strings.HasSuffix(path, "/release") &&
		r.Method == http.MethodPost:
		h.releaseReservation(w, r)
	default:
		http.NotFound(w, r)
	}
}

// ── Request / Response types ──────────────────────────────────────────────────

type createReservationRequest struct {
	Resources     []resourceCountRequest `json:"resources"`
	DurationHours int                    `json:"duration_hours"`
}

type resourceCountRequest struct {
	ResourceID string `json:"resource_id"`
	Count      int    `json:"count"`
}

type resourceResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	OperatingSystem string `json:"operating_system"`
}

type reservationResourceResponse struct {
	ResourceID    string `json:"resource_id"`
	ResourceName  string `json:"resource_name"`
	InstanceID    string `json:"instance_id"`
	InstanceState string `json:"instance_state"`
	IPAddress     string `json:"ip_address"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

type reservationResponse struct {
	ID        string                        `json:"id"`
	Status    string                        `json:"status"`
	Duration  int64                         `json:"duration"`
	CreatedAt int64                         `json:"created_at"`
	Resources []reservationResourceResponse `json:"resources"`
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func (h *HttpAdapter) listResources(w http.ResponseWriter, r *http.Request) {
	resources, err := h.app.ListResources(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := make([]resourceResponse, len(resources))
	for i, res := range resources {
		resp[i] = resourceResponse{ID: res.ID.String(), Name: res.Name, OperatingSystem: res.OperatingSystem}
	}
	jsonOK(w, resp)
}

func (h *HttpAdapter) listReservations(w http.ResponseWriter, r *http.Request) {
	reservations, err := h.app.ListAllReservations(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build resource name lookup for enriched responses
	resources, _ := h.app.ListResources(r.Context())
	nameByID := make(map[string]string, len(resources))
	for _, res := range resources {
		nameByID[res.ID.String()] = res.Name
	}

	resp := make([]reservationResponse, len(reservations))
	for i, rv := range reservations {
		resResps := make([]reservationResourceResponse, len(rv.ReservationResources))
		for j, rr := range rv.ReservationResources {
			ip := rr.IPAddress

			// TODO ArsenP : remove hardcoded username and password.
			// Get username and password from the Deployment Response. Password should be a secret (AWS secret Manager)
			username, password := "", ""
			if ip != "" {
				username = "ubuntu"
				password = "fleet-" + rr.InstanceID.String()[:8]
			}
			resResps[j] = reservationResourceResponse{
				ResourceID:    rr.ResourceID.String(),
				ResourceName:  nameByID[rr.ResourceID.String()],
				InstanceID:    rr.InstanceID.String(),
				InstanceState: string(rr.InstanceState),
				IPAddress:     ip,
				Username:      username,
				Password:      password,
			}
		}
		resp[i] = reservationResponse{
			ID:        rv.ID.String(),
			Status:    string(rv.Status),
			Duration:  rv.Duration,
			CreatedAt: rv.CreatedAt,
			Resources: resResps,
		}
	}
	jsonOK(w, resp)
}

func (h *HttpAdapter) createReservation(w http.ResponseWriter, r *http.Request) {
	var req createReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(req.Resources) == 0 {
		jsonError(w, "no resources specified", http.StatusBadRequest)
		return
	}
	if req.DurationHours < 1 {
		jsonError(w, "duration_hours must be >= 1", http.StatusBadRequest)
		return
	}

	var resources []domain.ReservationResource
	for _, rc := range req.Resources {
		if rc.Count < 1 {
			continue
		}
		resourceID, err := uuid.Parse(rc.ResourceID)
		if err != nil {
			jsonError(w, fmt.Sprintf("invalid resource_id %q", rc.ResourceID), http.StatusBadRequest)
			return
		}
		for i := 0; i < rc.Count; i++ {
			resources = append(resources, domain.NewReservationResource(resourceID, nil))
		}
	}
	if len(resources) == 0 {
		jsonError(w, "all resources have count 0", http.StatusBadRequest)
		return
	}

	reservation := domain.NewReservation(int64(req.DurationHours)*3600, resources)
	created, err := h.app.CreateReservation(r.Context(), reservation)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"id": created.ID.String(), "status": string(created.Status)})
}

func (h *HttpAdapter) releaseReservation(w http.ResponseWriter, r *http.Request) {
	// path: /api/reservations/{id}/release  →  parts: ["api","reservations","{id}","release"]
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 {
		jsonError(w, "invalid path", http.StatusBadRequest)
		return
	}
	reservationID, err := uuid.Parse(parts[2])
	if err != nil {
		jsonError(w, "invalid reservation id", http.StatusBadRequest)
		return
	}
	if err := h.app.ReleaseReservation(r.Context(), reservationID); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]bool{"ok": true})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}
