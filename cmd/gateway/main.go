package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	"github.com/devdudeio/verus-gateway/internal/cache"
	"github.com/devdudeio/verus-gateway/internal/chain"
	"github.com/devdudeio/verus-gateway/internal/config"
	"github.com/devdudeio/verus-gateway/internal/domain"
	"github.com/devdudeio/verus-gateway/internal/http/server"
	"github.com/devdudeio/verus-gateway/internal/observability/logger"
	"github.com/devdudeio/verus-gateway/internal/observability/metrics"
)

var (
	// Version information (set by build flags)
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Parse command line flags
	var (
		configPath  = flag.String("config", "", "path to configuration file")
		showVersion = flag.Bool("version", false, "show version information and exit")
	)
	flag.Parse()

	// Show version and exit if requested
	if *showVersion {
		fmt.Printf("Verus Gateway\n")
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	// Load configuration
	log.Println("Loading configuration...")
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("✓ Configuration loaded successfully")
	log.Printf("  Server: %s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("  Default chain: %s", cfg.Chains.Default)
	log.Printf("  Cache type: %s", cfg.Cache.Type)
	log.Printf("  Logging: level=%s, format=%s", cfg.Observability.Logging.Level, cfg.Observability.Logging.Format)

	// Print enabled chains
	enabledChains := 0
	for id, chain := range cfg.Chains.Chains {
		if chain.Enabled {
			enabledChains++
			log.Printf("  Chain %s: %s (RPC: %s)", id, chain.Name, chain.RPCURL)
		}
	}
	log.Printf("  Total enabled chains: %d", enabledChains)

	// Initialize logger
	log.Println("Initializing logger...")
	appLogger, err := logger.New(logger.Config{
		Level:    cfg.Observability.Logging.Level,
		Format:   cfg.Observability.Logging.Format,
		Output:   cfg.Observability.Logging.Output,
		FilePath: cfg.Observability.Logging.FilePath,
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	appLogger.Info().Msg("Logger initialized successfully")

	// Initialize metrics
	log.Println("Initializing metrics...")
	appMetrics := metrics.New("verus_gateway")
	appLogger.Info().Msg("Metrics initialized successfully")

	// Initialize cache
	appLogger.Info().Msg("Initializing cache...")
	cache, err := initializeCache(cfg)
	if err != nil {
		appLogger.Fatal().Err(err).Msg("Failed to initialize cache")
	}
	defer func() {
		if cache != nil {
			_ = cache.Close()
		}
	}()
	appLogger.Info().Str("type", cfg.Cache.Type).Msg("Cache initialized successfully")

	// Initialize chain manager
	appLogger.Info().Msg("Initializing chain manager...")
	chainManager, err := initializeChainManager(cfg)
	if err != nil {
		appLogger.Fatal().Err(err).Msg("Failed to initialize chain manager")
	}
	defer func() { _ = chainManager.Close() }()
	appLogger.Info().Msg("Chain manager initialized successfully")

	// Initialize HTTP server
	appLogger.Info().Msg("Initializing HTTP server...")
	httpServer := initializeHTTPServer(cfg, chainManager, cache, &appLogger, appMetrics)
	appLogger.Info().Msg("HTTP server initialized successfully")

	appLogger.Info().Msg("Verus Gateway initialized successfully")
	appLogger.Info().
		Str("host", cfg.Server.Host).
		Int("port", cfg.Server.Port).
		Msg("Starting server")

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Start HTTP server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := httpServer.Start(); err != nil {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal or server error
	select {
	case <-ctx.Done():
		appLogger.Info().Msg("Shutdown signal received, initiating graceful shutdown")
	case err := <-serverErr:
		appLogger.Error().Err(err).Msg("Server error occurred")
	}

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		appLogger.Error().Err(err).Msg("Error during server shutdown")
	}

	// Wait for shutdown to complete or timeout
	<-shutdownCtx.Done()

	if shutdownCtx.Err() == context.DeadlineExceeded {
		appLogger.Warn().Msg("Shutdown timeout exceeded, forcing exit")
	} else {
		appLogger.Info().Msg("Server stopped gracefully")
	}

	appLogger.Info().Msg("Shutdown complete")
}

func init() {
	// Set log format
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// initializeCache initializes the cache based on configuration
func initializeCache(cfg *config.Config) (domain.Cache, error) {
	switch cfg.Cache.Type {
	case "filesystem":
		return cache.NewFilesystemCache(cache.FilesystemCacheConfig{
			BaseDir:         cfg.Cache.Dir,
			MaxSize:         cfg.Cache.MaxSize,
			TTL:             cfg.Cache.TTL,
			CleanupInterval: cfg.Cache.CleanupInterval,
		})
	case "redis":
		return cache.NewRedisCache(cache.RedisCacheConfig{
			Addresses:  cfg.Cache.Redis.Addresses,
			Password:   cfg.Cache.Redis.Password,
			DB:         cfg.Cache.Redis.DB,
			MaxRetries: cfg.Cache.Redis.MaxRetries,
			PoolSize:   cfg.Cache.Redis.PoolSize,
			Timeout:    cfg.Cache.Redis.Timeout,
			TTL:        cfg.Cache.TTL, // Use top-level TTL
		})
	case "none", "":
		// No caching
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported cache type: %s", cfg.Cache.Type)
	}
}

// initializeChainManager initializes the chain manager
func initializeChainManager(cfg *config.Config) (*chain.Manager, error) {
	return chain.NewManager(cfg)
}

// initializeHTTPServer initializes the HTTP server
func initializeHTTPServer(cfg *config.Config, chainManager *chain.Manager, cache domain.Cache, logger *zerolog.Logger, m *metrics.Metrics) *server.Server {
	return server.New(server.Config{
		ChainManager: chainManager,
		Cache:        cache,
		Config:       cfg,
		Version:      Version,
		Logger:       logger,
		Metrics:      m,
	})
}
