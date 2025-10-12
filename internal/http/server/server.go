package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"

	"github.com/devdudeio/verus-gateway/internal/chain"
	"github.com/devdudeio/verus-gateway/internal/config"
	"github.com/devdudeio/verus-gateway/internal/domain"
	"github.com/devdudeio/verus-gateway/internal/http/handler"
	"github.com/devdudeio/verus-gateway/internal/http/middleware"
	"github.com/devdudeio/verus-gateway/internal/observability/metrics"
	"github.com/devdudeio/verus-gateway/internal/service"
)

// Server represents the HTTP server
type Server struct {
	router       *chi.Mux
	httpServer   *http.Server
	chainManager *chain.Manager
	cache        domain.Cache
	config       *config.Config
	version      string
	logger       *zerolog.Logger
	metrics      *metrics.Metrics
}

// Config holds server configuration
type Config struct {
	ChainManager *chain.Manager
	Cache        domain.Cache
	Config       *config.Config
	Version      string
	Logger       *zerolog.Logger
	Metrics      *metrics.Metrics
}

// New creates a new HTTP server
func New(cfg Config) *Server {
	s := &Server{
		router:       chi.NewRouter(),
		chainManager: cfg.ChainManager,
		cache:        cfg.Cache,
		config:       cfg.Config,
		version:      cfg.Version,
		logger:       cfg.Logger,
		metrics:      cfg.Metrics,
	}

	// Setup middleware
	s.setupMiddleware()

	// Setup routes
	s.setupRoutes()

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Config.Server.Host, cfg.Config.Server.Port),
		Handler:      s.router,
		ReadTimeout:  time.Duration(cfg.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s
}

// setupMiddleware configures middleware stack
func (s *Server) setupMiddleware() {
	// Recoverer - must be first to catch panics in other middleware
	s.router.Use(middleware.Recoverer(s.logger))

	// Request ID - add unique ID to each request
	s.router.Use(middleware.RequestID)

	// Real IP - extract real client IP
	s.router.Use(middleware.RealIP)

	// Logger - log all requests with structured logging
	s.router.Use(middleware.Logger(s.logger))

	// Metrics - record Prometheus metrics
	if s.metrics != nil {
		s.router.Use(middleware.Metrics(s.metrics))
	}

	// Timeout - add request timeout
	s.router.Use(middleware.Timeout(time.Duration(s.config.Server.ReadTimeout) * time.Second))

	// Security headers
	s.router.Use(middleware.SecurityHeaders)

	// CORS
	if s.config.Security.CORS.Enabled {
		s.router.Use(cors.Handler(cors.Options{
			AllowedOrigins:   s.config.Security.CORS.AllowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
			ExposedHeaders:   []string{"X-Request-ID", "Content-Disposition"},
			AllowCredentials: false,
			MaxAge:           300,
		}))
	}

	// Compress responses
	s.router.Use(chimiddleware.Compress(5))
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Create services
	fileService := service.NewFileService(s.chainManager, s.cache)

	// Create handlers
	fileHandler := handler.NewFileHandler(fileService)
	adminHandler := handler.NewAdminHandler(fileService, s.chainManager, s.metrics, s.version)

	// Health endpoints (no prefix)
	s.router.Get("/health", adminHandler.Health)
	s.router.Get("/ready", adminHandler.Ready)
	s.router.Get("/metrics", adminHandler.PrometheusMetrics)
	s.router.Get("/chains", adminHandler.ListChains)

	// Chain-specific API endpoints - ALL API calls must include chain
	s.router.Route("/c/{chain}", func(r chi.Router) {
		r.Get("/file/{txid}", fileHandler.GetFile)
		r.Head("/file/{txid}", fileHandler.HeadFile)
		r.Get("/meta/{txid}", fileHandler.GetMeta)
	})

	// Admin endpoints
	s.router.Route("/admin", func(r chi.Router) {
		// TODO: Add authentication middleware in Phase 11
		r.Get("/cache/stats", adminHandler.GetCacheStats)
		r.Delete("/cache", adminHandler.ClearCache)
		r.Delete("/cache/{key}", adminHandler.DeleteCacheEntry)
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	fmt.Printf("Starting HTTP server on %s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	fmt.Println("Shutting down HTTP server...")
	return s.httpServer.Shutdown(ctx)
}

// Router returns the Chi router (useful for testing)
func (s *Server) Router() *chi.Mux {
	return s.router
}
