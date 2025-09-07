package configserver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestMeta carries identifying details gleaned from the /config request.
type RequestMeta struct {
	Hostname  string
	MAC       string
	Serial    string
	UUID      string
	IP        string
	UserAgent string
	Time      time.Time
}

// Role denotes which type of machineconfig to return.
type Role string

const (
	RoleControlPlane Role = "controlplane"
	RoleWorker       Role = "worker"
)

// Generator renders machineconfig YAML for the given role & request.
type Generator interface {
	GenerateControlPlane(ctx context.Context, meta RequestMeta) ([]byte, error)
	GenerateWorker(ctx context.Context, meta RequestMeta) ([]byte, error)
}

// defaultGenerator is a stub that returns placeholder YAML.
// Replace this with real generation via Talos machinery/config packages.
type defaultGenerator struct{}

func DefaultGenerator() Generator { return &defaultGenerator{} }

func (g *defaultGenerator) GenerateControlPlane(_ context.Context, meta RequestMeta) ([]byte, error) {
	yaml := fmt.Sprintf(`# placeholder controlplane.machineconfig.yaml
# generated: %s
# from: IP=%s hostname=%s mac=%s serial=%s uuid=%s

kind: MachineConfig
machine:
  type: controlplane
  token: CHANGEME
cluster:
  name: CHANGEME
  controlPlane:
    endpoint: https://%s:6443
`, time.Now().UTC().Format(time.RFC3339), meta.IP, meta.Hostname, meta.MAC, meta.Serial, meta.UUID, meta.IP)
	return []byte(yaml), nil
}

func (g *defaultGenerator) GenerateWorker(_ context.Context, meta RequestMeta) ([]byte, error) {
	yaml := fmt.Sprintf(`# placeholder worker.machineconfig.yaml
# generated: %s
# from: IP=%s hostname=%s mac=%s serial=%s uuid=%s

kind: MachineConfig
machine:
  type: worker
  token: CHANGEME
`, time.Now().UTC().Format(time.RFC3339), meta.IP, meta.Hostname, meta.MAC, meta.Serial, meta.UUID)
	return []byte(yaml), nil
}

// Server holds the HTTP server & state.
type Server struct {
	router *gin.Engine
	srv    *http.Server

	mu                 sync.Mutex
	firstAssigned      bool
	gen                Generator
	assignControlPlane AssignControlPlaneFunc
}

// AssignControlPlaneFunc decides whether this request should receive a controlplane config.
// The default implementation assigns the *first ever* request as controlplane.
type AssignControlPlaneFunc func(meta RequestMeta, alreadyAssigned bool) bool

// New creates a Server with the provided Generator.
// If gen is nil, DefaultGenerator() is used.
// Optionally override the control-plane selection logic with WithAssignControlPlane.
func New(gen Generator, opts ...Option) *Server {
	if gen == nil {
		gen = DefaultGenerator()
	}
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	s := &Server{
		router:             r,
		gen:                gen,
		assignControlPlane: defaultAssignControlPlane,
	}

	for _, opt := range opts {
		opt(s)
	}

	r.GET("/config", s.handleConfig)

	return s
}

// Option configures the Server.
type Option func(*Server)

// WithAssignControlPlane overrides control-plane selection logic.
func WithAssignControlPlane(fn AssignControlPlaneFunc) Option {
	return func(s *Server) { s.assignControlPlane = fn }
}

// defaultAssignControlPlane marks the very first request as control-plane.
func defaultAssignControlPlane(_ RequestMeta, alreadyAssigned bool) bool {
	return !alreadyAssigned
}

// Start begins serving on addr (e.g., ":8080"). Blocks until the server stops.
func (s *Server) Start(addr string) error {
	s.srv = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

// handleConfig implements:
//
//	GET /config?h=${hostname}&m=${mac}&s=${serial}&u=${uuid}
func (s *Server) handleConfig(c *gin.Context) {
	meta := RequestMeta{
		Hostname:  strings.TrimSpace(c.Query("h")),
		MAC:       strings.TrimSpace(c.Query("m")),
		Serial:    strings.TrimSpace(c.Query("s")),
		UUID:      strings.TrimSpace(c.Query("u")),
		IP:        clientIP(c.Request),
		UserAgent: c.Request.UserAgent(),
		Time:      time.Now().UTC(),
	}

	// Require at least one identifier to avoid handing out configs indiscriminately.
	if meta.Hostname == "" && meta.MAC == "" && meta.Serial == "" && meta.UUID == "" {
		c.String(http.StatusBadRequest, "at least one of h (hostname), m (mac), s (serial), u (uuid) is required")
		return
	}

	// Decide role for this request.
	role := RoleWorker
	s.mu.Lock()
	isCP := s.assignControlPlane(meta, s.firstAssigned)
	if isCP && !s.firstAssigned {
		s.firstAssigned = true
		role = RoleControlPlane
	} else {
		role = RoleWorker
	}
	s.mu.Unlock()

	// Generate YAML.
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var (
		out []byte
		err error
	)
	switch role {
	case RoleControlPlane:
		out, err = s.gen.GenerateControlPlane(ctx, meta)
	default:
		out, err = s.gen.GenerateWorker(ctx, meta)
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "generation error: %v", err)
		return
	}

	c.Header("Content-Type", "application/yaml; charset=utf-8")
	c.String(http.StatusOK, string(out))
}

// clientIP tries common headers first, then falls back to RemoteAddr.
func clientIP(r *http.Request) string {
	// X-Forwarded-For may contain multiple IPs. Take the first.
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if xr := strings.TrimSpace(r.Header.Get("X-Real-Ip")); xr != "" {
		return xr
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func CreateServer() {
	gen := DefaultGenerator()
	srv := New(gen)
	go func() {
		if err := srv.Start(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
}
