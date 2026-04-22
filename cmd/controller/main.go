package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"

	"github.com/abevz/hybrid-cloud-optimizer/internal/cost"
)

func main() {
	region := flag.String("region", "eu-central-1", "...")
	instanceType := flag.String("type", "t3.micro", "...")
	flag.Parse()

	ctx := context.Background()
	defaultRegion := "us-east-1"
	client, err := cost.NewAWSPricingClient(ctx, defaultRegion, slog.Default())
	if err != nil {
		log.Fatal(err)
	}
	price, err := client.GetEC2HourlyPrice(ctx, *instanceType, *region)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s in %s: $%.4f/hour\n", *instanceType, *region, price)
}
