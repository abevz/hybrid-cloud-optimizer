package config

import (
	"testing"

	"github.com/samber/do/v2"
)

func TestProvideConfig(t *testing.T) {
	clearConfigEnv(t)

	injector := do.New()
	got, err := ProvideConfig(injector)
	if err != nil {
		t.Fatalf("ProvideConfig() error = %v", err)
	}

	want := &Config{
		AWSRegion:                 "us-east-1",
		AWSPricingAPIRegion:       "us-east-1",
		ProxmoxScaleOutThreshold:  0.85,
		ProxmoxScaleBackThreshold: 0.70,
		VPNEndpoint:               "10.0.1.1:51820",
		KarpenterEnabled:          true,
		KarpenterNamespace:        "karpenter",
		LogLevel:                  "info",
	}

	if *got != *want {
		t.Fatalf("ProvideConfig() = %#v, want %#v", *got, *want)
	}
}
