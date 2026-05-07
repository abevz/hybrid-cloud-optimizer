package config

import (
	"os"
	"strings"
	"testing"
)

var configEnvKeys = []string{
	"AWS_REGION",
	"AWS_PRICING_API_REGION",
	"PROXMOX_SCALE_OUT_THRESHOLD",
	"PROXMOX_SCALE_BACK_THRESHOLD",
	"VPN_ENDPOINT",
	"KARPENTER_ENABLED",
	"KARPENTER_NAMESPACE",
	"LOG_LEVEL",
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	for _, key := range configEnvKeys {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("Unsetenv(%q) failed: %v", key, err)
		}
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name         string
		cfg          Config
		wantErrParts []string
	}{
		{
			name: "empty AWS region",
			cfg: Config{
				AWSRegion:                 "",
				AWSPricingAPIRegion:       "us-east-1",
				ProxmoxScaleOutThreshold:  0.85,
				ProxmoxScaleBackThreshold: 0.70,
				VPNEndpoint:               "10.0.1.1:51820",
				KarpenterEnabled:          true,
				KarpenterNamespace:        "karpenter",
				LogLevel:                  "info",
			},
			wantErrParts: []string{"AWS_REGION must be non-empty"},
		},
		{
			name: "empty AWS pricing API region",
			cfg: Config{
				AWSRegion:                 "us-east-1",
				AWSPricingAPIRegion:       "",
				ProxmoxScaleOutThreshold:  0.85,
				ProxmoxScaleBackThreshold: 0.70,
				VPNEndpoint:               "10.0.1.1:51820",
				KarpenterEnabled:          true,
				KarpenterNamespace:        "karpenter",
				LogLevel:                  "info",
			},
			wantErrParts: []string{"AWS_PRICING_API_REGION must be non-empty"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
			for _, part := range tt.wantErrParts {
				if !strings.Contains(err.Error(), part) {
					t.Fatalf("Validate() error = %q, want substring %q", err.Error(), part)
				}
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name         string
		env          map[string]string
		want         *Config
		wantErrParts []string
	}{
		{
			name: "defaults",
			env:  map[string]string{},
			want: &Config{
				AWSRegion:                 "us-east-1",
				AWSPricingAPIRegion:       "us-east-1",
				ProxmoxScaleOutThreshold:  0.85,
				ProxmoxScaleBackThreshold: 0.70,
				VPNEndpoint:               "10.0.1.1:51820",
				KarpenterEnabled:          true,
				KarpenterNamespace:        "karpenter",
				LogLevel:                  "info",
			},
		},
		{
			name: "valid overrides",
			env: map[string]string{
				"AWS_REGION":                   "eu-central-1",
				"AWS_PRICING_API_REGION":       "us-east-1",
				"PROXMOX_SCALE_OUT_THRESHOLD":  "0.90",
				"PROXMOX_SCALE_BACK_THRESHOLD": "0.60",
				"VPN_ENDPOINT":                 "10.1.2.3:51820",
				"KARPENTER_ENABLED":            "false",
				"KARPENTER_NAMESPACE":          "autoscaler",
				"LOG_LEVEL":                    "DEBUG",
			},
			want: &Config{
				AWSRegion:                 "eu-central-1",
				AWSPricingAPIRegion:       "us-east-1",
				ProxmoxScaleOutThreshold:  0.90,
				ProxmoxScaleBackThreshold: 0.60,
				VPNEndpoint:               "10.1.2.3:51820",
				KarpenterEnabled:          false,
				KarpenterNamespace:        "autoscaler",
				LogLevel:                  "debug",
			},
		},
		{
			name: "invalid float parse",
			env: map[string]string{
				"PROXMOX_SCALE_OUT_THRESHOLD": "abc",
			},
			wantErrParts: []string{
				"invalid config",
				"PROXMOX_SCALE_OUT_THRESHOLD",
				"float64",
			},
		},
		{
			name: "invalid bool parse",
			env: map[string]string{
				"KARPENTER_ENABLED": "maybe",
			},
			wantErrParts: []string{
				"invalid config",
				"KARPENTER_ENABLED",
				"bool",
			},
		},
		{
			name: "invalid threshold pair",
			env: map[string]string{
				"PROXMOX_SCALE_OUT_THRESHOLD":  "0.70",
				"PROXMOX_SCALE_BACK_THRESHOLD": "0.70",
			},
			wantErrParts: []string{
				"PROXMOX_SCALE_OUT_THRESHOLD must be greater than PROXMOX_SCALE_BACK_THRESHOLD",
			},
		},
		{
			name: "missing vpn endpoint when karpenter enabled",
			env: map[string]string{
				"KARPENTER_ENABLED": "true",
				"VPN_ENDPOINT":      "",
			},
			wantErrParts: []string{
				"VPN_ENDPOINT must be non-empty when KARPENTER_ENABLED=true",
			},
		},
		{
			name: "invalid log level",
			env: map[string]string{
				"LOG_LEVEL": "trace",
			},
			wantErrParts: []string{
				"LOG_LEVEL must be one of debug, info, warn, error",
			},
		},
		{
			name: "invalid karpenter namespace",
			env: map[string]string{
				"KARPENTER_NAMESPACE": "bad_namespace",
			},
			wantErrParts: []string{
				"KARPENTER_NAMESPACE must be an RFC 1123 label",
			},
		},
		{
			name: "aggregates multiple errors",
			env: map[string]string{
				"PROXMOX_SCALE_OUT_THRESHOLD":  "abc",
				"PROXMOX_SCALE_BACK_THRESHOLD": "2.0",
				"LOG_LEVEL":                    "trace",
			},
			wantErrParts: []string{
				"PROXMOX_SCALE_OUT_THRESHOLD",
				"PROXMOX_SCALE_BACK_THRESHOLD must be > 0 and <= 1",
				"LOG_LEVEL must be one of debug, info, warn, error",
			},
		},
		{
			name: "aggregates bool parse and validate errors",
			env: map[string]string{
				"KARPENTER_ENABLED":            "maybe",
				"PROXMOX_SCALE_OUT_THRESHOLD":  "0.60",
				"PROXMOX_SCALE_BACK_THRESHOLD": "0.70",
				"LOG_LEVEL":                    "trace",
			},
			wantErrParts: []string{
				"KARPENTER_ENABLED",
				"PROXMOX_SCALE_OUT_THRESHOLD must be greater than PROXMOX_SCALE_BACK_THRESHOLD",
				"LOG_LEVEL must be one of debug, info, warn, error",
			},
		},
		{
			name: "invalid scale back threshold parse",
			env: map[string]string{
				"PROXMOX_SCALE_BACK_THRESHOLD": "abc",
			},
			wantErrParts: []string{
				"invalid config",
				"PROXMOX_SCALE_BACK_THRESHOLD",
				"float64",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)

			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			got, err := LoadConfig()

			if len(tt.wantErrParts) > 0 {
				if err == nil {
					t.Fatal("LoadConfig() error = nil, want error")
				}
				for _, part := range tt.wantErrParts {
					if !strings.Contains(err.Error(), part) {
						t.Fatalf("LoadConfig() error = %q, want substring %q", err.Error(), part)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			if *got != *tt.want {
				t.Fatalf("LoadConfig() = %#v, want %#v", *got, *tt.want)
			}
		})
	}
}
