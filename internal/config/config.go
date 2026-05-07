// Package config provides environment variable loading and validation for the controller.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
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
}

// LoadConfig reads configuration from environment variables
func LoadConfig() (*Config, error) {
	var errs []error

	proxmoxScaleOutThreshold, err := lookupFloat("PROXMOX_SCALE_OUT_THRESHOLD", 0.85)
	if err != nil {
		errs = append(errs, err)
	}

	proxmoxScaleBackThreshold, err := lookupFloat("PROXMOX_SCALE_BACK_THRESHOLD", 0.70)
	if err != nil {
		errs = append(errs, err)
	}

	karpenterEnabled, err := lookupBool("KARPENTER_ENABLED", true)
	if err != nil {
		errs = append(errs, err)
	}

	cfg := &Config{
		AWSRegion:           lookupString("AWS_REGION", "us-east-1"),
		AWSPricingAPIRegion: lookupString("AWS_PRICING_API_REGION", "us-east-1"), // Pricing API global endpoint

		ProxmoxScaleOutThreshold:  proxmoxScaleOutThreshold,
		ProxmoxScaleBackThreshold: proxmoxScaleBackThreshold,
		VPNEndpoint:               lookupString("VPN_ENDPOINT", "10.0.1.1:51820"),
		KarpenterEnabled:          karpenterEnabled,
		KarpenterNamespace:        lookupString("KARPENTER_NAMESPACE", "karpenter"),
		LogLevel:                  strings.ToLower(lookupString("LOG_LEVEL", "info")),
	}

	if err := cfg.Validate(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid config: %w", errors.Join(errs...))
	}

	return cfg, nil
}

// Validate checks configuration sanity
func (c *Config) Validate() error {
	var errs []error

	if strings.TrimSpace(c.AWSRegion) == "" {
		errs = append(errs, errors.New("AWS_REGION must be non-empty"))
	}

	if strings.TrimSpace(c.AWSPricingAPIRegion) == "" {
		errs = append(errs, errors.New("AWS_PRICING_API_REGION must be non-empty"))
	}

	if c.ProxmoxScaleOutThreshold <= 0 || c.ProxmoxScaleOutThreshold > 1 {
		errs = append(errs, errors.New("PROXMOX_SCALE_OUT_THRESHOLD must be > 0 and <= 1"))
	}

	if c.ProxmoxScaleBackThreshold <= 0 || c.ProxmoxScaleBackThreshold > 1 {
		errs = append(errs, errors.New("PROXMOX_SCALE_BACK_THRESHOLD must be > 0 and <= 1"))
	}

	if c.ProxmoxScaleOutThreshold <= c.ProxmoxScaleBackThreshold {
		errs = append(errs, errors.New("PROXMOX_SCALE_OUT_THRESHOLD must be greater than PROXMOX_SCALE_BACK_THRESHOLD"))
	}

	if c.KarpenterEnabled && strings.TrimSpace(c.VPNEndpoint) == "" {
		errs = append(errs, errors.New("VPN_ENDPOINT must be non-empty when KARPENTER_ENABLED=true"))
	}

	if problems := k8svalidation.IsDNS1123Label(c.KarpenterNamespace); len(problems) > 0 {
		errs = append(errs, fmt.Errorf("KARPENTER_NAMESPACE must be an RFC 1123 label: %s", strings.Join(problems,
			"; ")))
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, errors.New("LOG_LEVEL must be one of debug, info, warn, error"))
	}

	return errors.Join(errs...)
}

// Helper functions
func lookupString(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}
	return value
}

func lookupFloat(key string, defaultValue float64) (float64, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a float64: %w", key, err)
	}

	return parsed, nil
}

func lookupBool(key string, defaultValue bool) (bool, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a bool: %w", key, err)
	}

	return parsed, nil
}
