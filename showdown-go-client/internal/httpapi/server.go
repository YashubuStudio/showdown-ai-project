package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ysu/showdown-go-client/internal/studio"
	"github.com/ysu/showdown-go-client/pkg/showdown"
)

const maxRequestBodyBytes = 1 << 20 // 1 MB

type Server struct {
	mux *http.ServeMux
}

func New() *Server {
	s := &Server{mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	s.mux.HandleFunc("/api/local-format", s.handleLocalFormat)
	s.mux.HandleFunc("/api/ping", s.handlePing)
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/validate-team", s.handleValidateTeam)
	s.mux.HandleFunc("/api/mock-battle", s.handleMockBattle)
}

func (s *Server) handleLocalFormat(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		state, err := studio.LoadLocalFormatState()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, state)
	case http.MethodPost:
		limitBody(r)
		req := struct {
			Config        showdown.LocalFormatConfig `json:"config"`
			RestartServer *bool                      `json:"restart_server"`
		}{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		restartServer := true
		if req.RestartServer != nil {
			restartServer = *req.RestartServer
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		state, err := studio.SaveLocalFormatConfig(ctx, req.Config, restartServer)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, state)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
	}
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	limitBody(r)
	req := struct {
		ServerURL string `json:"server_url"`
		Username  string `json:"username"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateServerURL(req.ServerURL); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	info, err := showdown.Ping(ctx, req.ServerURL, defaultUsername(req.Username, "ping"))
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	limitBody(r)
	req := struct {
		ServerURL string `json:"server_url"`
		Username  string `json:"username"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateServerURL(req.ServerURL); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	status, err := showdown.FetchStatus(ctx, req.ServerURL, defaultUsername(req.Username, "status"))
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleMockBattle(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	limitBody(r)
	req := showdown.MockBattleRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateServerURL(req.ServerURL); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	formatID := req.Format
	if strings.TrimSpace(formatID) == "" {
		if strings.TrimSpace(req.TeamA) != "" || strings.TrimSpace(req.TeamB) != "" {
			formatID = showdown.LocalStudioFormatID
		} else {
			formatID = "gen9randombattle"
		}
	}
	req.Format = formatID

	if strings.TrimSpace(req.TeamA) != "" || strings.TrimSpace(req.TeamB) != "" {
		if strings.TrimSpace(req.TeamA) == "" || strings.TrimSpace(req.TeamB) == "" {
			writeError(w, http.StatusBadRequest, errTeamsMustBeProvidedTogether)
			return
		}

		validationA, err := studio.ValidateTeam(r.Context(), formatID, req.TeamA)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		validationB, err := studio.ValidateTeam(r.Context(), formatID, req.TeamB)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		if !validationA.Valid || !validationB.Valid {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"ok":            false,
				"error":         "team validation failed",
				"team_a_errors": validationA.Errors,
				"team_b_errors": validationB.Errors,
			})
			return
		}
		req.PackedTeamA = validationA.PackedTeam
		req.PackedTeamB = validationB.PackedTeam
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout+15*time.Second)
	defer cancel()

	result, err := showdown.RunMockBattle(ctx, req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleValidateTeam(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	limitBody(r)
	req := struct {
		FormatID string `json:"format_id"`
		Team     string `json:"team"`
	}{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	result, err := studio.ValidateTeam(r.Context(), req.FormatID, req.Team)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func defaultUsername(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	writeError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
	return false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	msg := "internal server error"
	var ae *apiError
	if errors.As(err, &ae) {
		msg = ae.message
	} else if status == http.StatusBadRequest || status == http.StatusBadGateway || status == http.StatusUnprocessableEntity {
		msg = err.Error()
	}
	writeJSON(w, status, map[string]any{
		"ok":    false,
		"error": msg,
	})
}

func limitBody(r *http.Request) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxRequestBodyBytes)
}

func validateServerURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return &apiError{message: "server_url is required"}
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return &apiError{message: "server_url is not a valid URL"}
	}
	switch parsed.Scheme {
	case "http", "https":
	default:
		return &apiError{message: "server_url must use http or https scheme"}
	}
	if parsed.Host == "" {
		return &apiError{message: "server_url must include a host"}
	}
	return nil
}

var (
	errMethodNotAllowed            = &apiError{message: "method not allowed"}
	errTeamsMustBeProvidedTogether = &apiError{message: "team_a and team_b must be provided together"}
)

type apiError struct {
	message string
}

func (e *apiError) Error() string {
	return e.message
}
