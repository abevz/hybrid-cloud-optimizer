package config

import (
	"github.com/samber/do/v2"
)

// ProvideConfig registers Config in DI container
func ProvideConfig(i do.Injector) (*Config, error) {
	return LoadConfig()
}
