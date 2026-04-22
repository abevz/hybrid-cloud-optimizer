// Package config provides environment variable loading and validation for the controller.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	// AWS settings
	AWSRegion           string
	AWSPricingAPIRegion string // Pricing API only in us-east-1

	// Proxmox thresholds (hysteresis to prevent flapping)
	ProxmoxScaleOutThreshold  float64 // Default: 0.85 (burst to AWS when above)
	ProxmoxScaleBackThreshold float64 // Default: 0.70 (return to Proxmox when below)

	// VPN settings
	VPNEndpoint string // VPN endpoint for health checks (e.g., "10.0.1.1:51820")

	// Karpenter settings
	KarpenterEnabled   bool
	KarpenterNamespace string

	// Logging
	LogLevel string // debug, info, warn, error

	// Metrics
	MetricsAddr string // :8080
	ProbeAddr   string // :8081
}

// LoadConfig reads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		AWSRegion:                 getEnv("AWS_REGION", "us-east-1"),
		AWSPricingAPIRegion:       "us-east-1", // Pricing API global endpoint
		ProxmoxScaleOutThreshold:  getEnvFloat("PROXMOX_SCALE_OUT_THRESHOLD", 0.85),
		ProxmoxScaleBackThreshold: getEnvFloat("PROXMOX_SCALE_BACK_THRESHOLD", 0.70),
		VPNEndpoint:               getEnv("VPN_ENDPOINT", "10.0.1.1:51820"),
		KarpenterEnabled:          getEnvBool("KARPENTER_ENABLED", true),
		KarpenterNamespace:        getEnv("KARPENTER_NAMESPACE", "karpenter"),
		LogLevel:                  getEnv("LOG_LEVEL", "info"),
		MetricsAddr:               getEnv("METRICS_ADDR", ":8080"),
		ProbeAddr:                 getEnv("PROBE_ADDR", ":8081"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// Validate checks configuration sanity
func (c *Config) Validate() error {
	if c.ProxmoxScaleOutThreshold <= 0 || c.ProxmoxScaleOutThreshold > 1 {
		return fmt.Errorf("PROXMOX_SCALE_OUT_THRESHOLD must be between 0 and 1")
	}
	if c.ProxmoxScaleBackThreshold <= 0 || c.ProxmoxScaleBackThreshold > 1 {
		return fmt.Errorf("PROXMOX_SCALE_BACK_THRESHOLD must be between 0 and 1")
	}
	if c.ProxmoxScaleOutThreshold <= c.ProxmoxScaleBackThreshold {
		return fmt.Errorf("PROXMOX_SCALE_OUT_THRESHOLD (%.2f) must be greater than PROXMOX_SCALE_BACK_THRESHOLD (%.2f)", c.ProxmoxScaleOutThreshold, c.ProxmoxScaleBackThreshold)
	}
	return nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
