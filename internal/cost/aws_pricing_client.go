package cost

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"
)

// AWSPricingClient fetches EC2 instance pricing
type AWSPricingClient struct {
	client *pricing.Client
	logger *slog.Logger
}

// NewAWSPricingClient creates a new pricing client
func NewAWSPricingClient(ctx context.Context, region string, logger *slog.Logger) (*AWSPricingClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	return &AWSPricingClient{
		client: pricing.NewFromConfig(cfg),
		logger: logger,
	}, nil
}

// GetEC2HourlyPrice fetches the on-demand hourly price for an EC2 instance type
func (c *AWSPricingClient) GetEC2HourlyPrice(ctx context.Context, instanceType, region string) (float64, error) {
	filters := []types.Filter{
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("ServiceCode"),
			Value: aws.String("AmazonEC2"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("instanceType"),
			Value: aws.String(instanceType),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("regionCode"),
			Value: aws.String(region),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("tenancy"),
			Value: aws.String("Shared"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("operatingSystem"),
			Value: aws.String("Linux"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("preInstalledSw"),
			Value: aws.String("NA"),
		},
		{
			Type:  types.FilterTypeTermMatch,
			Field: aws.String("capacitystatus"),
			Value: aws.String("Used"),
		},
	}

	input := &pricing.GetProductsInput{
		ServiceCode: aws.String("AmazonEC2"),
		Filters:     filters,
		MaxResults:  aws.Int32(1),
	}

	output, err := c.client.GetProducts(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("AWS Pricing API error: %w", err)
	}

	if len(output.PriceList) == 0 {
		return 0, fmt.Errorf("no pricing data for instance type %s in region %s", instanceType, region)
	}

	//fmt.Println(output.PriceList[0])
	//
	// Parse JSON response
	var priceData map[string]interface{}
	if err := json.Unmarshal([]byte(output.PriceList[0]), &priceData); err != nil {
		return 0, fmt.Errorf("parse pricing JSON: %w", err)
	}

	// Extract on-demand price (simplified parsing)
	terms, ok := priceData["terms"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("missing 'terms' in pricing data")
	}

	onDemand, ok := terms["OnDemand"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("missing 'OnDemand' term")
	}

	// Iterate through pricing dimensions
	for _, v := range onDemand {
		priceDimensions, ok := v.(map[string]interface{})["priceDimensions"].(map[string]interface{})
		if !ok {
			continue
		}

		for _, pd := range priceDimensions {
			pricePerUnit, ok := pd.(map[string]interface{})["pricePerUnit"].(map[string]interface{})
			if !ok {
				continue
			}

			usdStr, ok := pricePerUnit["USD"].(string)
			if !ok {
				continue
			}

			var price float64
			if _, err := fmt.Sscanf(usdStr, "%f", &price); err == nil {
				c.logger.InfoContext(ctx, "fetched EC2 price",
					"instance_type", instanceType,
					"region", region,
					"hourly_usd", price,
				)
				return price, nil
			}
		}
	}

	return 0, fmt.Errorf("could not extract price from response")
}

// EstimateMonthlyPrice calculates monthly cost (730 hours/month)
func (c *AWSPricingClient) EstimateMonthlyPrice(hourlyPrice float64) float64 {
	return hourlyPrice * 730 // Average hours per month
}
