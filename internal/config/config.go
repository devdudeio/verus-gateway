package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Chains        ChainsConfig        `mapstructure:"chains"`
	Cache         CacheConfig         `mapstructure:"cache"`
	Security      SecurityConfig      `mapstructure:"security"`
	RateLimit     RateLimitConfig     `mapstructure:"rate_limit"`
	Observability ObservabilityConfig `mapstructure:"observability"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	Host            string        `mapstructure:"host"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	MaxRequestSize  int64         `mapstructure:"max_request_size"`
}

// ChainsConfig holds blockchain configuration
type ChainsConfig struct {
	Default string                 `mapstructure:"default"`
	Chains  map[string]ChainConfig `mapstructure:"chains"`
}

// ChainConfig holds configuration for a single blockchain
type ChainConfig struct {
	Name        string        `mapstructure:"name"`
	Enabled     bool          `mapstructure:"enabled"`
	RPCURL      string        `mapstructure:"rpc_url"`
	RPCUser     string        `mapstructure:"rpc_user"`
	RPCPassword string        `mapstructure:"rpc_password"`
	RPCTimeout  time.Duration `mapstructure:"rpc_timeout"`
	TLSInsecure bool          `mapstructure:"tls_insecure"`
	MaxRetries  int           `mapstructure:"max_retries"`
	RetryDelay  time.Duration `mapstructure:"retry_delay"`
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Type            string               `mapstructure:"type"` // filesystem, redis, memcached, multi
	Dir             string               `mapstructure:"dir"`
	MaxSize         int64                `mapstructure:"max_size"`
	TTL             time.Duration        `mapstructure:"ttl"`
	CleanupInterval time.Duration        `mapstructure:"cleanup_interval"`
	Redis           RedisCacheConfig     `mapstructure:"redis"`
	Memcached       MemcachedCacheConfig `mapstructure:"memcached"`
}

// RedisCacheConfig holds Redis cache configuration
type RedisCacheConfig struct {
	Addresses  []string      `mapstructure:"addresses"`
	Password   string        `mapstructure:"password"`
	DB         int           `mapstructure:"db"`
	MaxRetries int           `mapstructure:"max_retries"`
	PoolSize   int           `mapstructure:"pool_size"`
	Timeout    time.Duration `mapstructure:"timeout"`
}

// MemcachedCacheConfig holds Memcached cache configuration
type MemcachedCacheConfig struct {
	Servers []string      `mapstructure:"servers"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	CORS           CORSConfig `mapstructure:"cors"`
	MaxFilenameLen int        `mapstructure:"max_filename_length"`
	AllowedMethods []string   `mapstructure:"allowed_methods"`
	TrustedProxies []string   `mapstructure:"trusted_proxies"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
	ExposeHeaders  []string `mapstructure:"expose_headers"`
	MaxAge         int      `mapstructure:"max_age"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled     bool          `mapstructure:"enabled"`
	WindowSize  time.Duration `mapstructure:"window_size"`
	MaxRequests int           `mapstructure:"max_requests"`
	Burst       int           `mapstructure:"burst"`
}

// ObservabilityConfig holds observability configuration
type ObservabilityConfig struct {
	Logging LoggingConfig `mapstructure:"logging"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	Tracing TracingConfig `mapstructure:"tracing"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level    string `mapstructure:"level"`  // debug, info, warn, error
	Format   string `mapstructure:"format"` // json, text
	Output   string `mapstructure:"output"` // stdout, stderr, file
	FilePath string `mapstructure:"file_path"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
	Port    int    `mapstructure:"port"`
}

// TracingConfig holds tracing configuration
type TracingConfig struct {
	Enabled    bool    `mapstructure:"enabled"`
	Provider   string  `mapstructure:"provider"` // jaeger, otlp
	Endpoint   string  `mapstructure:"endpoint"`
	SampleRate float64 `mapstructure:"sample_rate"`
}

// Load loads configuration from multiple sources with priority:
// 1. Command line flags (highest)
// 2. Environment variables
// 3. Config file
// 4. Defaults (lowest)
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config file path if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Look for config file in standard locations
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.verus-gateway")
		v.AddConfigPath("/etc/verus-gateway")
	}

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; ignore error if desired
	}

	// Environment variables
	v.SetEnvPrefix("VERUS_GATEWAY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.read_timeout", 10*time.Second)
	v.SetDefault("server.write_timeout", 60*time.Second)
	v.SetDefault("server.idle_timeout", 120*time.Second)
	v.SetDefault("server.shutdown_timeout", 30*time.Second)
	v.SetDefault("server.max_request_size", 32*1024*1024) // 32MB

	// Cache defaults
	v.SetDefault("cache.type", "filesystem")
	v.SetDefault("cache.dir", "./cache")
	v.SetDefault("cache.max_size", 1024*1024*1024) // 1GB
	v.SetDefault("cache.ttl", 24*time.Hour)
	v.SetDefault("cache.cleanup_interval", 1*time.Hour)

	// Redis defaults
	v.SetDefault("cache.redis.addresses", []string{"localhost:6379"})
	v.SetDefault("cache.redis.db", 0)
	v.SetDefault("cache.redis.max_retries", 3)
	v.SetDefault("cache.redis.pool_size", 10)
	v.SetDefault("cache.redis.timeout", 5*time.Second)

	// Memcached defaults
	v.SetDefault("cache.memcached.servers", []string{"localhost:11211"})
	v.SetDefault("cache.memcached.timeout", 5*time.Second)

	// Security defaults
	v.SetDefault("security.cors.enabled", true)
	v.SetDefault("security.cors.allowed_origins", []string{"*"})
	v.SetDefault("security.cors.allowed_methods", []string{"GET", "HEAD", "OPTIONS"})
	v.SetDefault("security.cors.allowed_headers", []string{"Content-Type", "Authorization"})
	v.SetDefault("security.cors.max_age", 3600)
	v.SetDefault("security.max_filename_length", 255)

	// Rate limit defaults
	v.SetDefault("rate_limit.enabled", true)
	v.SetDefault("rate_limit.window_size", 10*time.Second)
	v.SetDefault("rate_limit.max_requests", 30)
	v.SetDefault("rate_limit.burst", 10)

	// Logging defaults
	v.SetDefault("observability.logging.level", "info")
	v.SetDefault("observability.logging.format", "json")
	v.SetDefault("observability.logging.output", "stdout")

	// Metrics defaults
	v.SetDefault("observability.metrics.enabled", true)
	v.SetDefault("observability.metrics.path", "/metrics")
	v.SetDefault("observability.metrics.port", 9090)

	// Tracing defaults
	v.SetDefault("observability.tracing.enabled", false)
	v.SetDefault("observability.tracing.sample_rate", 0.1)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Validate at least one chain is configured
	if len(c.Chains.Chains) == 0 {
		return fmt.Errorf("no chains configured")
	}

	// Validate default chain exists
	if c.Chains.Default != "" {
		if _, ok := c.Chains.Chains[c.Chains.Default]; !ok {
			return fmt.Errorf("default chain '%s' not found in chains", c.Chains.Default)
		}
	}

	// Validate each chain
	for id, chain := range c.Chains.Chains {
		if err := chain.Validate(id); err != nil {
			return fmt.Errorf("invalid chain config for '%s': %w", id, err)
		}
	}

	// Validate cache config
	validCacheTypes := map[string]bool{
		"filesystem": true,
		"redis":      true,
		"memcached":  true,
		"multi":      true,
	}
	if !validCacheTypes[c.Cache.Type] {
		return fmt.Errorf("invalid cache type: %s", c.Cache.Type)
	}

	// Validate logging level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.Observability.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Observability.Logging.Level)
	}

	return nil
}

// Validate validates a chain configuration
func (cc *ChainConfig) Validate(id string) error {
	if !cc.Enabled {
		return nil // Skip validation for disabled chains
	}

	if cc.RPCURL == "" {
		return fmt.Errorf("rpc_url is required")
	}

	if cc.RPCUser == "" {
		return fmt.Errorf("rpc_user is required")
	}

	if cc.RPCPassword == "" {
		return fmt.Errorf("rpc_password is required")
	}

	if cc.RPCTimeout < time.Second {
		return fmt.Errorf("rpc_timeout must be at least 1 second")
	}

	if cc.MaxRetries < 0 || cc.MaxRetries > 10 {
		return fmt.Errorf("max_retries must be between 0 and 10")
	}

	return nil
}
