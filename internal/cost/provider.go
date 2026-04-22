package cost

import (
	"context"
	"log/slog"

	"github.com/abevz/hybrid-cloud-optimizer/internal/config"

	"github.com/samber/do/v2"
)

// ProvideAWSPricingClient registers AWSPricingClient in DI container
func ProvideAWSPricingClient(i do.Injector) (*AWSPricingClient, error) {
	cfg := do.MustInvoke[*config.Config](i)
	logger := do.MustInvoke[*slog.Logger](i)

	return NewAWSPricingClient(context.Background(), cfg.AWSPricingAPIRegion, logger)
}
