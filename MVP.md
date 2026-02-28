# Hybrid Cloud Cost Optimizer - MVP Specification

> **Production-ready Kubernetes Controller** для динамического управления рабочими нагрузками между on-premise (Proxmox) и AWS на основе стоимости и утилизации ресурсов.

---

## 📋 MVP Scope: Что строим в первой версии

### ✅ В MVP

1. **Единый Kubernetes кластер**
   - **Control Plane**: на Proxmox (on-premise)
   - **Worker Nodes**: Proxmox (базовые) + AWS EC2 (burst capacity)
   - **Hybrid scheduling**: поды могут быть на любых нодах

2. **HybridWorkload CRD**
   - Аннотации для приоритета (`priority: low|medium|high`)
   - Budget constraints (`maxMonthlyCostUSD`)
   - Требования к ресурсам (CPU/Memory)
   - **Автоматическое создание NodePool в AWS** через Karpenter при необходимости

3. **Cost-Aware Scheduler Logic**
   - Мониторинг утилизации Proxmox нод
   - Расчет стоимости AWS EC2 инстансов (Pricing API)
   - Решение: "Proxmox (бесплатно) vs AWS (платно)"
   - **Автоматическое создание Karpenter NodePool** для AWS burst

4. **Production-Ready Foundation**
   - Structured logging (`slog`)
   - Graceful shutdown
   - Health checks (`/healthz`, `/readyz`)
   - Prometheus metrics (базовые)

5. **Testing**
   - Unit tests (table-driven, mocks)
   - Integration tests (`envtest`)
   - E2E тест (kind cluster)

### 🚫 Откладываем на Phase 2

- ~~Migration между платформами~~ → Manual re-deploy OK
- ~~OpenTelemetry tracing~~ → slog достаточно для MVP
- ~~Helm Chart~~ → Plain manifests OK
- ~~Vault integration~~ → Environment variables OK
- ~~Validating/Mutating Webhooks~~ → ✅ **Moved to MVP** (validating webhook for CRD spec)
- ~~Multi-tenancy~~ → Cluster-scoped OK

---

## 🏗️ Architecture Overview

### Infrastructure Stack

**Kubernetes**: v1.33.x  
**CNI**: Cilium (eBPF-based networking)  
**Node OS**: Ubuntu 22.04 LTS  
**VPN**: Tailscale (dev) → WireGuard (prod)  

**Why Cilium CNI?**
- ✅ **Hybrid cloud support**: Native support for cross-datacenter pod networking
- ✅ **WireGuard encryption**: Built-in transparent encryption for pod-to-pod traffic (Cilium 1.13+)
- ✅ **eBPF performance**: Low latency, high throughput (critical for VPN overlay)
- ✅ **Network policies**: Advanced L3/L4/L7 policies for multi-location security
- ✅ **Observability**: Hubble integration for network flow visibility

**Cilium Configuration for Hybrid Mode**:
```yaml
# helm values for Cilium (already deployed on Proxmox)
tunnel: "vxlan"  # Overlay mode for cross-location routing
encryption:
  enabled: true
  type: "wireguard"  # Cilium native WireGuard (alternative to separate VPN)
ipam:
  mode: "kubernetes"  # Standard K8s IPAM for simplicity
hubble:
  enabled: true  # Network observability
  relay:
    enabled: true
```

### Hybrid Kubernetes Cluster Topology

```
┌─────────────────────────────────────────────────────────────────┐
│  On-Premise (Proxmox VE)                                        │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Kubernetes Control Plane (Master Nodes)                 │   │
│  │  • kube-apiserver                                        │   │
│  │  • kube-scheduler (native + HCRO webhook filter)         │   │
│  │  • kube-controller-manager                               │   │
│  │  • etcd                                                   │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Worker Nodes (Proxmox VMs)                              │   │
│  │  • node-1 (8 CPU, 16GB RAM) ─┐                           │   │
│  │  • node-2 (8 CPU, 16GB RAM)  ├─ "Free" capacity          │   │
│  │  • node-3 (8 CPU, 16GB RAM) ─┘                           │   │
│  │                                                           │   │
│  │  Labels:                                                 │   │
│  │    platform: proxmox                                     │   │
│  │    cost: zero                                            │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  HCRO Controller Pod                                     │   │
│  │  • Watches HybridWorkload CRD                            │   │
│  │  • Monitors Proxmox node utilization                     │   │
│  │  • Creates Karpenter NodePools dynamically               │   │
│  │  • Fetches AWS EC2 pricing                               │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                             │
                             │ VPN / AWS Direct Connect
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  AWS Cloud                                                      │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Worker Nodes (EC2 Instances)                            │   │
│  │  • Managed by Karpenter (auto-scaling)                   │   │
│  │  • Created dynamically when Proxmox capacity > 85%       │   │
│  │  • t3.medium, t3.large, c5.xlarge, etc.                  │   │
│  │                                                           │   │
│  │  Labels:                                                 │   │
│  │    platform: aws                                         │   │
│  │    cost: paid                                            │   │
│  │    karpenter.sh/nodepool: hcro-burst-pool                │   │
│  │    capacity-type: spot|on-demand                         │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Decision Flow

```
User applies HybridWorkload CR
         │
         ▼
┌────────────────────────────────────────────────────────────────┐
│ HCRO Controller Reconcile Loop                                │
│                                                                │
│ 1. Fetch HybridWorkload spec:                                 │
│    • priority: low|medium|high                                │
│    • maxMonthlyCostUSD: 50                                    │
│    • resources: {cpu: 2, memory: 4Gi}                         │
│                                                                │
│ 2. Check Proxmox node utilization:                            │
│    • Query Kubernetes Metrics API                             │
│    • Calculate aggregate CPU/Memory usage                     │
│                                                                │
│ 3. Decision Logic:                                            │
│    ┌────────────────────────────────────────────────────┐     │
│    │ IF priority == "high" THEN                         │     │
│    │   → ALWAYS use AWS (reliability > cost)            │     │
│    │   → Create Karpenter NodePool (on-demand)          │     │
│    │                                                     │     │
│    │ ELSE IF proxmox_utilization < 85% THEN             │     │
│    │   → Prefer Proxmox (free capacity, scale-out)      │     │
│    │   → Add nodeSelector: {platform: proxmox}          │     │
│    │                                                     │     │
│    │ ELSE IF proxmox_utilization < 70% AND on AWS THEN  │     │
│    │   → Scale back to Proxmox (hysteresis)             │     │
│    │   → Add nodeSelector: {platform: proxmox}          │     │
│    │                                                     │     │
│    │ ELSE IF budget allows THEN                         │     │
│    │   → Check VPN health before AWS burst              │     │
│    │   → Create Karpenter NodePool (Spot preferred)     │     │
│    │   → Add nodeSelector: {platform: aws}              │     │
│    │                                                     │     │
│    │ ELSE                                                │     │
│    │   → WAIT (queue workload)                          │     │
│    │   → Update .status.phase = "Pending"               │     │
│    └────────────────────────────────────────────────────┘     │
│                                                                │
│ 4. Create/Update Resources:                                   │
│    • Karpenter NodePool (if AWS selected)                     │
│    • Deployment/StatefulSet (with nodeSelector)               │
│    • Update HybridWorkload.status                             │
└────────────────────────────────────────────────────────────────┘
```

---

## 📁 Project Structure

```
hybrid-cloud-optimizer/
├── cmd/
│   ├── controller/
│   │   └── main.go                    # Entry point, DI composition root
│   └── kubectl-hcro/
│       └── main.go                    # kubectl plugin: cost savings report
│
├── api/
│   └── v1alpha1/
│       ├── hybridworkload_types.go    # HybridWorkload CRD definition
│       ├── hybridworkload_webhook.go  # Validating webhook
│       └── zz_generated.deepcopy.go   # Generated by controller-gen
│
├── internal/
│   ├── config/
│   │   ├── config.go                  # Configuration struct
│   │   └── provider.go                # samber/do provider
│   │
│   ├── controller/
│   │   ├── hybridworkload_controller.go  # Reconciler
│   │   └── provider.go
│   │
│   ├── cost/
│   │   ├── aws_pricing_client.go      # AWS Pricing API client
│   │   └── provider.go
│   │
│   ├── metrics/
│   │   ├── proxmox_metrics.go         # K8s Metrics API client
│   │   └── provider.go
│   │
│   ├── scheduler/
│   │   ├── decision_engine.go         # Placement decision logic
│   │   └── provider.go
│   │
│   ├── karpenter/
│   │   ├── nodepool_manager.go        # Karpenter NodePool CRUD
│   │   └── provider.go
│   │
│   ├── errors/
│   │   └── errors.go                  # Typed errors with retry semantics
│   │
│   └── healthcheck/
│       ├── health.go                  # Health check aggregator
│       ├── vpn_health.go             # VPN tunnel connectivity checker
│       └── provider.go
│
├── config/
│   └── grafana/
│       └── hcro-dashboard.json        # Grafana dashboard (placement, cost, latency)
│
├── deploy/
│   ├── crds/
│   │   └── cost.hybrid.io_hybridworkloads.yaml
│   ├── rbac/
│   │   ├── role.yaml
│   │   ├── role_binding.yaml
│   │   └── service_account.yaml
│   └── deployment.yaml                # Controller deployment
│
├── docs/
│   └── conventions/
│       ├── go-patterns.md              # Go coding conventions (error handling, SafeGo, logging)
│       └── samber-do.md                # samber/do v2 DI conventions (providers, isolation)
│
├── examples/
│   ├── hybridworkload-low-priority.yaml
│   ├── hybridworkload-high-priority.yaml
│   └── karpenter-nodepool-example.yaml
│
├── test/
│   ├── unit/
│   │   ├── decision_engine_test.go
│   │   └── aws_pricing_test.go
│   ├── integration/
│   │   └── controller_test.go         # envtest
│   └── e2e/
│       └── e2e_test.go                # kind cluster
│
├── .pre-commit-config.yaml            # Go linters, formatting
├── Dockerfile                         # Multi-stage build
├── Makefile                           # make run, test, deploy
├── go.mod
├── go.sum
└── MVP.md                             # This document
```

---

## 🧩 Core Components

### 1. HybridWorkload CRD

**File:** `api/v1alpha1/hybridworkload_types.go`

```go
package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    corev1 "k8s.io/api/core/v1"
)

// HybridWorkloadSpec defines the desired state of HybridWorkload
type HybridWorkloadSpec struct {
    // Priority determines scheduling preference (high = always AWS)
    // +kubebuilder:validation:Enum=low;medium;high
    // +kubebuilder:default=medium
    Priority string `json:"priority"`

    // MaxMonthlyCostUSD is the budget constraint for this workload
    // +kubebuilder:validation:Minimum=0
    MaxMonthlyCostUSD float64 `json:"maxMonthlyCostUSD,omitempty"`

    // Resources specifies CPU and memory requirements
    Resources corev1.ResourceRequirements `json:"resources"`

    // WorkloadTemplate is the Pod spec for Deployment/StatefulSet
    WorkloadTemplate corev1.PodTemplateSpec `json:"workloadTemplate"`

    // CapacityType for AWS nodes: "spot" or "on-demand"
    // +kubebuilder:validation:Enum=spot;on-demand
    // +kubebuilder:default=spot
    CapacityType string `json:"capacityType,omitempty"`
}

// HybridWorkloadStatus defines the observed state
type HybridWorkloadStatus struct {
    // Phase: Pending, Running, Failed
    // +kubebuilder:validation:Enum=Pending;Running;Failed
    Phase string `json:"phase,omitempty"`

    // RecommendedPlatform: "proxmox" or "aws"
    RecommendedPlatform string `json:"recommendedPlatform,omitempty"`

    // EstimatedMonthlyCostUSD for current placement
    EstimatedMonthlyCostUSD float64 `json:"estimatedMonthlyCostUSD,omitempty"`

    // KarpenterNodePoolName if AWS burst was created
    KarpenterNodePoolName string `json:"karpenterNodePoolName,omitempty"`

    // DryRun indicates decision was computed but no resources were created
    DryRun bool `json:"dryRun,omitempty"`

    // Conditions for status tracking (SchedulingDecision, NodePoolReady, VPNHealthy)
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // LastReconcileTime
    LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Priority",type=string,JSONPath=`.spec.priority`
// +kubebuilder:printcolumn:name="Platform",type=string,JSONPath=`.status.recommendedPlatform`
// +kubebuilder:printcolumn:name="Cost",type=string,JSONPath=`.status.estimatedMonthlyCostUSD`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// HybridWorkload is the Schema for the hybridworkloads API
type HybridWorkload struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   HybridWorkloadSpec   `json:"spec,omitempty"`
    Status HybridWorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HybridWorkloadList contains a list of HybridWorkload
type HybridWorkloadList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []HybridWorkload `json:"items"`
}

func init() {
    SchemeBuilder.Register(&HybridWorkload{}, &HybridWorkloadList{})
}
```

---

### 2. Configuration

**File:** `internal/config/config.go`

```go
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
    KarpenterEnabled bool
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
        AWSRegion:                   getEnv("AWS_REGION", "us-east-1"),
        AWSPricingAPIRegion:         "us-east-1", // Pricing API global endpoint
        ProxmoxScaleOutThreshold:  getEnvFloat("PROXMOX_SCALE_OUT_THRESHOLD", 0.85),
        ProxmoxScaleBackThreshold: getEnvFloat("PROXMOX_SCALE_BACK_THRESHOLD", 0.70),
        VPNEndpoint:               getEnv("VPN_ENDPOINT", "10.0.1.1:51820"),
        KarpenterEnabled:            getEnvBool("KARPENTER_ENABLED", true),
        KarpenterNamespace:          getEnv("KARPENTER_NAMESPACE", "karpenter"),
        LogLevel:                    getEnv("LOG_LEVEL", "info"),
        MetricsAddr:                 getEnv("METRICS_ADDR", ":8080"),
        ProbeAddr:                   getEnv("PROBE_ADDR", ":8081"),
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
```

**File:** `internal/config/provider.go`

```go
package config

import (
    "github.com/samber/do/v2"
)

// ProvideConfig registers Config in DI container
func ProvideConfig(i do.Injector) (*Config, error) {
    return LoadConfig()
}
```

---

### 3. AWS Pricing Client

**File:** `internal/cost/aws_pricing_client.go`

```go
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
```

**File:** `internal/cost/provider.go`

```go
package cost

import (
    "context"
    "log/slog"

    "github.com/samber/do/v2"
    "hybrid-cloud-optimizer/internal/config"
)

// ProvideAWSPricingClient registers AWSPricingClient in DI container
func ProvideAWSPricingClient(i do.Injector) (*AWSPricingClient, error) {
    cfg := do.MustInvoke[*config.Config](i)
    logger := do.MustInvoke[*slog.Logger](i)

    return NewAWSPricingClient(context.Background(), cfg.AWSPricingAPIRegion, logger)
}
```

---

### 4. Proxmox Metrics Client

**File:** `internal/metrics/proxmox_metrics.go`

```go
package metrics

import (
    "context"
    "fmt"
    "log/slog"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
    metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// ProxmoxMetricsClient monitors Proxmox node utilization via K8s Metrics API
type ProxmoxMetricsClient struct {
    clientset        *kubernetes.Clientset
    metricsClientset *metricsv.Clientset
    logger           *slog.Logger
}

// NewProxmoxMetricsClient creates a new metrics client
func NewProxmoxMetricsClient(clientset *kubernetes.Clientset, metricsClientset *metricsv.Clientset, logger *slog.Logger) *ProxmoxMetricsClient {
    return &ProxmoxMetricsClient{
        clientset:        clientset,
        metricsClientset: metricsClientset,
        logger:           logger,
    }
}

// GetProxmoxUtilization calculates aggregate CPU and memory utilization for Proxmox nodes
func (m *ProxmoxMetricsClient) GetProxmoxUtilization(ctx context.Context) (cpuUtilization, memoryUtilization float64, err error) {
    // List nodes with label "platform=proxmox"
    nodes, err := m.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
        LabelSelector: "platform=proxmox",
    })
    if err != nil {
        return 0, 0, fmt.Errorf("list Proxmox nodes: %w", err)
    }

    if len(nodes.Items) == 0 {
        return 0, 0, fmt.Errorf("no Proxmox nodes found")
    }

    // Fetch node metrics
    nodeMetrics, err := m.metricsClientset.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
    if err != nil {
        return 0, 0, fmt.Errorf("fetch node metrics: %w", err)
    }

    // Calculate totals
    var totalCPUCapacity, totalCPUUsage int64
    var totalMemoryCapacity, totalMemoryUsage int64

    for _, node := range nodes.Items {
        // CPU capacity (in millicores)
        cpuCapacity := node.Status.Capacity.Cpu().MilliValue()
        totalCPUCapacity += cpuCapacity

        // Memory capacity (in bytes)
        memCapacity := node.Status.Capacity.Memory().Value()
        totalMemoryCapacity += memCapacity

        // Find corresponding metrics
        for _, nm := range nodeMetrics.Items {
            if nm.Name == node.Name {
                totalCPUUsage += nm.Usage.Cpu().MilliValue()
                totalMemoryUsage += nm.Usage.Memory().Value()
                break
            }
        }
    }

    if totalCPUCapacity == 0 || totalMemoryCapacity == 0 {
        return 0, 0, fmt.Errorf("invalid node capacity")
    }

    cpuUtilization = float64(totalCPUUsage) / float64(totalCPUCapacity)
    memoryUtilization = float64(totalMemoryUsage) / float64(totalMemoryCapacity)

    m.logger.InfoContext(ctx, "Proxmox cluster utilization",
        "cpu_utilization", fmt.Sprintf("%.2f%%", cpuUtilization*100),
        "memory_utilization", fmt.Sprintf("%.2f%%", memoryUtilization*100),
    )

    return cpuUtilization, memoryUtilization, nil
}
```

**File:** `internal/metrics/provider.go`

```go
package metrics

import (
    "log/slog"

    "github.com/samber/do/v2"
    "k8s.io/client-go/kubernetes"
    metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// ProvideProxmoxMetricsClient registers ProxmoxMetricsClient in DI container
func ProvideProxmoxMetricsClient(i do.Injector) (*ProxmoxMetricsClient, error) {
    clientset := do.MustInvoke[*kubernetes.Clientset](i)
    metricsClientset := do.MustInvoke[*metricsv.Clientset](i)
    logger := do.MustInvoke[*slog.Logger](i)

    return NewProxmoxMetricsClient(clientset, metricsClientset, logger), nil
}
```

---

### 5. Decision Engine

**File:** `internal/scheduler/decision_engine.go`

```go
package scheduler

import (
    "context"
    "fmt"
    "log/slog"

    "hybrid-cloud-optimizer/api/v1alpha1"
    "hybrid-cloud-optimizer/internal/config"
    "hybrid-cloud-optimizer/internal/cost"
    hcroerrors "hybrid-cloud-optimizer/internal/errors"
    "hybrid-cloud-optimizer/internal/healthcheck"
    "hybrid-cloud-optimizer/internal/metrics"
)

// DecisionEngine determines optimal placement for workloads
type DecisionEngine struct {
    config         *config.Config
    pricingClient  *cost.AWSPricingClient
    metricsClient  *metrics.ProxmoxMetricsClient
    vpnChecker     *healthcheck.VPNHealthChecker
    logger         *slog.Logger
}

// NewDecisionEngine creates a new decision engine
func NewDecisionEngine(
    cfg *config.Config,
    pricingClient *cost.AWSPricingClient,
    metricsClient *metrics.ProxmoxMetricsClient,
    vpnChecker *healthcheck.VPNHealthChecker,
    logger *slog.Logger,
) *DecisionEngine {
    return &DecisionEngine{
        config:        cfg,
        pricingClient: pricingClient,
        metricsClient: metricsClient,
        vpnChecker:    vpnChecker,
        logger:        logger,
    }
}

// PlacementDecision represents the scheduling decision
type PlacementDecision struct {
    Platform              string  // "proxmox" or "aws"
    Reason                string  // Human-readable explanation
    EstimatedMonthlyCost  float64 // USD
    RequiresKarpenterPool bool    // True if AWS NodePool needed
    InstanceType          string  // e.g., "t3.medium" (for AWS)
    CapacityType          string  // "spot" or "on-demand"
}

// Decide makes placement decision based on priority, utilization, and budget.
// Uses hysteresis thresholds (ScaleOut=85%, ScaleBack=70%) to prevent flapping
// between platforms when utilization hovers near the boundary.
func (d *DecisionEngine) Decide(ctx context.Context, workload *v1alpha1.HybridWorkload) (*PlacementDecision, error) {
    priority := workload.Spec.Priority
    budget := workload.Spec.MaxMonthlyCostUSD
    currentPlatform := workload.Status.RecommendedPlatform

    // Rule 1: High priority workloads ALWAYS go to AWS
    if priority == "high" {
        if err := d.requireVPNHealthy(ctx); err != nil {
            return nil, err
        }
        return d.decideAWS(ctx, workload, "high priority requires AWS reliability")
    }

    // Rule 2: Check Proxmox utilization
    cpuUtil, memUtil, err := d.metricsClient.GetProxmoxUtilization(ctx)
    if err != nil {
        return nil, hcroerrors.NewProxmoxUnavailableError(fmt.Sprintf("failed to get Proxmox metrics: %v", err))
    }

    maxUtil := max(cpuUtil, memUtil)

    // Rule 3: Hysteresis — use different thresholds based on current placement
    // If currently on Proxmox (or new): burst to AWS only when > ScaleOutThreshold (85%)
    // If currently on AWS: return to Proxmox only when < ScaleBackThreshold (70%)
    if currentPlatform == "aws" {
        // Already on AWS — only scale back if utilization drops below ScaleBackThreshold
        if maxUtil < d.config.ProxmoxScaleBackThreshold {
            return &PlacementDecision{
                Platform:              "proxmox",
                Reason:                fmt.Sprintf("Proxmox utilization %.2f%% < scale-back threshold %.2f%%, returning from AWS", maxUtil*100, d.config.ProxmoxScaleBackThreshold*100),
                EstimatedMonthlyCost:  0,
                RequiresKarpenterPool: false,
            }, nil
        }
        // Stay on AWS (between 70-85%, hysteresis zone)
        return d.decideAWS(ctx, workload, fmt.Sprintf("staying on AWS (utilization %.2f%% in hysteresis zone)", maxUtil*100))
    }

    // New workload or currently on Proxmox — use ScaleOutThreshold
    if maxUtil < d.config.ProxmoxScaleOutThreshold {
        return &PlacementDecision{
            Platform:              "proxmox",
            Reason:                fmt.Sprintf("Proxmox utilization %.2f%% < scale-out threshold %.2f%%", maxUtil*100, d.config.ProxmoxScaleOutThreshold*100),
            EstimatedMonthlyCost:  0,
            RequiresKarpenterPool: false,
        }, nil
    }

    // Rule 4: Proxmox overloaded, check VPN health before AWS burst
    if err := d.requireVPNHealthy(ctx); err != nil {
        return nil, err
    }

    // Rule 5: Check budget for AWS burst
    if budget > 0 {
        decision, err := d.decideAWS(ctx, workload, fmt.Sprintf("Proxmox overloaded (%.2f%%), bursting to AWS", maxUtil*100))
        if err != nil {
            return nil, err
        }

        if decision.EstimatedMonthlyCost > budget {
            return nil, hcroerrors.NewBudgetExceededError(
                fmt.Sprintf("AWS cost $%.2f exceeds budget $%.2f", decision.EstimatedMonthlyCost, budget),
            )
        }

        return decision, nil
    }

    // No budget, must wait
    return &PlacementDecision{
        Platform: "pending",
        Reason:   "Proxmox overloaded and no budget for AWS",
    }, nil
}

// requireVPNHealthy checks VPN connectivity before making AWS decisions.
// Returns ErrVPNUnhealthy if the VPN tunnel is down, preventing AWS workload creation.
func (d *DecisionEngine) requireVPNHealthy(ctx context.Context) error {
    if !d.vpnChecker.IsHealthy(ctx) {
        return hcroerrors.NewVPNUnhealthyError("VPN tunnel to AWS is down, cannot create AWS workloads")
    }
    return nil
}

// decideAWS calculates AWS placement details
func (d *DecisionEngine) decideAWS(ctx context.Context, workload *v1alpha1.HybridWorkload, reason string) (*PlacementDecision, error) {
    // Select instance type based on resource requirements (simplified logic)
    instanceType := d.selectInstanceType(workload)

    // Fetch pricing
    hourlyPrice, err := d.pricingClient.GetEC2HourlyPrice(ctx, instanceType, d.config.AWSRegion)
    if err != nil {
        return nil, fmt.Errorf("fetch EC2 price: %w", err)
    }

    monthlyPrice := d.pricingClient.EstimateMonthlyPrice(hourlyPrice)

    capacityType := workload.Spec.CapacityType
    if capacityType == "" {
        capacityType = "spot" // Default to Spot for cost savings
    }

    // Spot instances are ~70% cheaper
    if capacityType == "spot" {
        monthlyPrice *= 0.3
    }

    return &PlacementDecision{
        Platform:              "aws",
        Reason:                reason,
        EstimatedMonthlyCost:  monthlyPrice,
        RequiresKarpenterPool: true,
        InstanceType:          instanceType,
        CapacityType:          capacityType,
    }, nil
}

// selectInstanceType picks an instance type based on resource requirements
func (d *DecisionEngine) selectInstanceType(workload *v1alpha1.HybridWorkload) string {
    cpuRequest := workload.Spec.Resources.Requests.Cpu().MilliValue()
    memRequest := workload.Spec.Resources.Requests.Memory().Value()

    // Simplified mapping (in production, use AWS EC2 instance metadata)
    switch {
    case cpuRequest <= 1000 && memRequest <= 2*1024*1024*1024: // 1 CPU, 2 GiB
        return "t3.small"
    case cpuRequest <= 2000 && memRequest <= 4*1024*1024*1024: // 2 CPU, 4 GiB
        return "t3.medium"
    case cpuRequest <= 4000 && memRequest <= 8*1024*1024*1024: // 4 CPU, 8 GiB
        return "t3.large"
    default:
        return "t3.xlarge"
    }
}

func max(a, b float64) float64 {
    if a > b {
        return a
    }
    return b
}
```

**File:** `internal/scheduler/provider.go`

```go
package scheduler

import (
    "log/slog"

    "github.com/samber/do/v2"
    "hybrid-cloud-optimizer/internal/config"
    "hybrid-cloud-optimizer/internal/cost"
    "hybrid-cloud-optimizer/internal/healthcheck"
    "hybrid-cloud-optimizer/internal/metrics"
)

// ProvideDecisionEngine registers DecisionEngine in DI container
func ProvideDecisionEngine(i do.Injector) (*DecisionEngine, error) {
    cfg := do.MustInvoke[*config.Config](i)
    pricingClient := do.MustInvoke[*cost.AWSPricingClient](i)
    metricsClient := do.MustInvoke[*metrics.ProxmoxMetricsClient](i)
    vpnChecker := do.MustInvoke[*healthcheck.VPNHealthChecker](i)
    logger := do.MustInvoke[*slog.Logger](i)

    return NewDecisionEngine(cfg, pricingClient, metricsClient, vpnChecker, logger), nil
}
```

---

### 6. Karpenter NodePool Manager

**File:** `internal/karpenter/nodepool_manager.go`

```go
package karpenter

import (
    "context"
    "fmt"
    "log/slog"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "k8s.io/client-go/dynamic"
)

// NodePoolManager manages Karpenter NodePools dynamically
type NodePoolManager struct {
    dynamicClient dynamic.Interface
    namespace     string
    logger        *slog.Logger
}

// NewNodePoolManager creates a new Karpenter NodePool manager
func NewNodePoolManager(dynamicClient dynamic.Interface, namespace string, logger *slog.Logger) *NodePoolManager {
    return &NodePoolManager{
        dynamicClient: dynamicClient,
        namespace:     namespace,
        logger:        logger,
    }
}

var nodepoolGVR = schema.GroupVersionResource{
    Group:    "karpenter.sh",
    Version:  "v1beta1",
    Resource: "nodepools",
}

// CreateOrUpdateNodePool creates or updates a Karpenter NodePool for AWS burst
func (m *NodePoolManager) CreateOrUpdateNodePool(ctx context.Context, name, instanceType, capacityType string, maxNodes int) error {
    nodepool := &unstructured.Unstructured{
        Object: map[string]interface{}{
            "apiVersion": "karpenter.sh/v1beta1",
            "kind":       "NodePool",
            "metadata": map[string]interface{}{
                "name": name,
            },
            "spec": map[string]interface{}{
                "template": map[string]interface{}{
                    "spec": map[string]interface{}{
                        "requirements": []interface{}{
                            map[string]interface{}{
                                "key":      "karpenter.sh/capacity-type",
                                "operator": "In",
                                "values":   []interface{}{capacityType},
                            },
                            map[string]interface{}{
                                "key":      "node.kubernetes.io/instance-type",
                                "operator": "In",
                                "values":   []interface{}{instanceType},
                            },
                            map[string]interface{}{
                                "key":      "kubernetes.io/arch",
                                "operator": "In",
                                "values":   []interface{}{"amd64"},
                            },
                        },
                        "nodeClassRef": map[string]interface{}{
                            "apiVersion": "karpenter.k8s.aws/v1beta1",
                            "kind":       "EC2NodeClass",
                            "name":       "default", // Assume default EC2NodeClass exists
                        },
                        "taints": []interface{}{
                            map[string]interface{}{
                                "key":    "platform",
                                "value":  "aws",
                                "effect": corev1.TaintEffectNoSchedule,
                            },
                        },
                    },
                },
                "limits": map[string]interface{}{
                    "cpu": fmt.Sprintf("%d", maxNodes*4), // Example: 4 CPUs per node
                },
                "disruption": map[string]interface{}{
                    "consolidationPolicy": "WhenUnderutilized",
                    "expireAfter":         "720h", // 30 days
                },
            },
        },
    }

    // Try to create NodePool
    _, err := m.dynamicClient.Resource(nodepoolGVR).Namespace(m.namespace).Create(ctx, nodepool, metav1.CreateOptions{})
    if err != nil {
        // If already exists, update it
        _, updateErr := m.dynamicClient.Resource(nodepoolGVR).Namespace(m.namespace).Update(ctx, nodepool, metav1.UpdateOptions{})
        if updateErr != nil {
            return fmt.Errorf("create/update NodePool: %w", updateErr)
        }
        m.logger.InfoContext(ctx, "updated Karpenter NodePool", "name", name)
    } else {
        m.logger.InfoContext(ctx, "created Karpenter NodePool", "name", name)
    }

    return nil
}

// DeleteNodePool deletes a Karpenter NodePool
func (m *NodePoolManager) DeleteNodePool(ctx context.Context, name string) error {
    err := m.dynamicClient.Resource(nodepoolGVR).Namespace(m.namespace).Delete(ctx, name, metav1.DeleteOptions{})
    if err != nil {
        return fmt.Errorf("delete NodePool: %w", err)
    }

    m.logger.InfoContext(ctx, "deleted Karpenter NodePool", "name", name)
    return nil
}
```

**File:** `internal/karpenter/provider.go`

```go
package karpenter

import (
    "log/slog"

    "github.com/samber/do/v2"
    "hybrid-cloud-optimizer/internal/config"
    "k8s.io/client-go/dynamic"
)

// ProvideNodePoolManager registers NodePoolManager in DI container
func ProvideNodePoolManager(i do.Injector) (*NodePoolManager, error) {
    dynamicClient := do.MustInvoke[dynamic.Interface](i)
    cfg := do.MustInvoke[*config.Config](i)
    logger := do.MustInvoke[*slog.Logger](i)

    return NewNodePoolManager(dynamicClient, cfg.KarpenterNamespace, logger), nil
}
```

---

### 7. Controller Reconciler

**File:** `internal/controller/hybridworkload_controller.go`

```go
package controller

import (
    "context"
    "errors"
    "fmt"
    "time"

    corev1 "k8s.io/api/core/v1"
    apierrors "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/log"

    costv1alpha1 "hybrid-cloud-optimizer/api/v1alpha1"
    hcroerrors "hybrid-cloud-optimizer/internal/errors"
    "hybrid-cloud-optimizer/internal/karpenter"
    "hybrid-cloud-optimizer/internal/scheduler"
)

// HybridWorkloadReconciler reconciles a HybridWorkload object
type HybridWorkloadReconciler struct {
    client.Client
    Scheme         *runtime.Scheme
    DecisionEngine *scheduler.DecisionEngine
    NodePoolMgr    *karpenter.NodePoolManager
}

// +kubebuilder:rbac:groups=cost.hybrid.io,resources=hybridworkloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cost.hybrid.io,resources=hybridworkloads/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cost.hybrid.io,resources=hybridworkloads/finalizers,verbs=update
// +kubebuilder:rbac:groups=karpenter.sh,resources=nodepools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=metrics.k8s.io,resources=nodes,verbs=get;list

// Reconcile is the main reconciliation loop
func (r *HybridWorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := log.FromContext(ctx)

    // Fetch HybridWorkload
    var workload costv1alpha1.HybridWorkload
    if err := r.Get(ctx, req.NamespacedName, &workload); err != nil {
        if apierrors.IsNotFound(err) {
            return ctrl.Result{}, nil
        }
        return ctrl.Result{}, err
    }

    // Check dry-run mode via annotation
    dryRun := workload.Annotations["hcro.io/dry-run"] == "true"

    // Make placement decision
    decision, err := r.DecisionEngine.Decide(ctx, &workload)
    if err != nil {
        // Structured error handling: different retry intervals per error type
        var requeueAfter time.Duration
        switch {
        case errors.As(err, &hcroerrors.ProxmoxUnavailableError{}):
            requeueAfter = 10 * time.Second
            r.setCondition(&workload, "SchedulingDecision", metav1.ConditionFalse, "ProxmoxUnavailable", err.Error())
        case errors.As(err, &hcroerrors.VPNUnhealthyError{}):
            requeueAfter = 1 * time.Minute
            r.setCondition(&workload, "VPNHealthy", metav1.ConditionFalse, "Unhealthy", err.Error())
        case errors.As(err, &hcroerrors.BudgetExceededError{}):
            requeueAfter = 5 * time.Minute
            workload.Status.Phase = "Pending"
            r.setCondition(&workload, "SchedulingDecision", metav1.ConditionFalse, "BudgetExceeded", err.Error())
        case errors.As(err, &hcroerrors.PricingAPIUnavailableError{}):
            requeueAfter = 30 * time.Second
            logger.Warn("AWS Pricing API unavailable, using cached price", "error", err)
        default:
            requeueAfter = 30 * time.Second
        }
        logger.Error(err, "failed to make placement decision", "requeue_after", requeueAfter)
        _ = r.Status().Update(ctx, &workload)
        return ctrl.Result{RequeueAfter: requeueAfter}, err
    }

    logger.Info("placement decision made",
        "platform", decision.Platform,
        "reason", decision.Reason,
        "cost", decision.EstimatedMonthlyCost,
        "dry_run", dryRun,
    )

    // Update status
    workload.Status.RecommendedPlatform = decision.Platform
    workload.Status.EstimatedMonthlyCostUSD = decision.EstimatedMonthlyCost
    workload.Status.DryRun = dryRun
    now := metav1.Now()
    workload.Status.LastReconcileTime = &now
    r.setCondition(&workload, "VPNHealthy", metav1.ConditionTrue, "Healthy", "VPN tunnel is up")

    // Create Karpenter NodePool if needed (skip in dry-run mode)
    if decision.RequiresKarpenterPool && !dryRun {
        poolName := fmt.Sprintf("hcro-%s", workload.Name)
        if err := r.NodePoolMgr.CreateOrUpdateNodePool(ctx, poolName, decision.InstanceType, decision.CapacityType, 5); err != nil {
            logger.Error(err, "failed to create Karpenter NodePool")
            workload.Status.Phase = "Failed"
            r.setCondition(&workload, "NodePoolReady", metav1.ConditionFalse, "CreationFailed", err.Error())
        } else {
            workload.Status.KarpenterNodePoolName = poolName
            workload.Status.Phase = "Running"
            r.setCondition(&workload, "NodePoolReady", metav1.ConditionTrue, "Created", "Karpenter NodePool created")
        }
    } else if dryRun {
        workload.Status.Phase = "Running"
        r.setCondition(&workload, "SchedulingDecision", metav1.ConditionTrue, "DryRun", fmt.Sprintf("dry-run: would place on %s", decision.Platform))
    } else {
        workload.Status.Phase = "Running"
        r.setCondition(&workload, "SchedulingDecision", metav1.ConditionTrue, "Decided", decision.Reason)
    }

    // Update status subresource
    if err := r.Status().Update(ctx, &workload); err != nil {
        return ctrl.Result{}, err
    }

    // Requeue after 5 minutes to re-evaluate
    return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// setCondition updates or adds a condition
func (r *HybridWorkloadReconciler) setCondition(workload *costv1alpha1.HybridWorkload, condType string, status metav1.ConditionStatus, reason, message string) {
    cond := metav1.Condition{
        Type:               condType,
        Status:             status,
        Reason:             reason,
        Message:            message,
        LastTransitionTime: metav1.Now(),
    }

    // Update existing condition or append new
    found := false
    for i, c := range workload.Status.Conditions {
        if c.Type == condType {
            workload.Status.Conditions[i] = cond
            found = true
            break
        }
    }
    if !found {
        workload.Status.Conditions = append(workload.Status.Conditions, cond)
    }
}

// SetupWithManager sets up the controller with the Manager.
func (r *HybridWorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&costv1alpha1.HybridWorkload{}).
        Complete(r)
}
```

---

### 8. Main Entry Point (DI Composition Root)

**File:** `cmd/controller/main.go`

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/samber/do/v2"
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/kubernetes"
    metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/healthz"
    "sigs.k8s.io/controller-runtime/pkg/log/zap"

    costv1alpha1 "hybrid-cloud-optimizer/api/v1alpha1"
    "hybrid-cloud-optimizer/internal/config"
    "hybrid-cloud-optimizer/internal/controller"
    "hybrid-cloud-optimizer/internal/cost"
    "hybrid-cloud-optimizer/internal/healthcheck"
    "hybrid-cloud-optimizer/internal/karpenter"
    "hybrid-cloud-optimizer/internal/metrics"
    "hybrid-cloud-optimizer/internal/scheduler"
)

func main() {
    var metricsAddr string
    var probeAddr string
    var enableLeaderElection bool

    flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
    flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
    flag.BoolVar(&enableLeaderElection, "leader-elect", true, "Enable leader election for controller manager. Required for HA deployments.")
    flag.Parse()

    // Setup structured logging
    opts := zap.Options{Development: true}
    ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    // Create DI container
    injector := do.New()

    // Register providers in layered order
    do.Provide(injector, config.ProvideConfig)
    do.Provide(injector, func(i do.Injector) (*slog.Logger, error) { return logger, nil })

    // K8s clients
    do.Provide(injector, func(i do.Injector) (*kubernetes.Clientset, error) {
        cfg := ctrl.GetConfigOrDie()
        return kubernetes.NewForConfig(cfg)
    })
    do.Provide(injector, func(i do.Injector) (metricsv.Interface, error) {
        cfg := ctrl.GetConfigOrDie()
        return metricsv.NewForConfig(cfg)
    })
    do.Provide(injector, func(i do.Injector) (dynamic.Interface, error) {
        cfg := ctrl.GetConfigOrDie()
        return dynamic.NewForConfig(cfg)
    })

    // Core services
    do.Provide(injector, cost.ProvideAWSPricingClient)
    do.Provide(injector, metrics.ProvideProxmoxMetricsClient)
    do.Provide(injector, healthcheck.ProvideVPNHealthChecker)
    do.Provide(injector, scheduler.ProvideDecisionEngine)
    do.Provide(injector, karpenter.ProvideNodePoolManager)

    // Resolve dependencies
    cfg := do.MustInvoke[*config.Config](injector)
    decisionEngine := do.MustInvoke[*scheduler.DecisionEngine](injector)
    nodePoolMgr := do.MustInvoke[*karpenter.NodePoolManager](injector)

    // Setup controller-runtime manager
    mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
        Scheme:                 scheme,
        MetricsBindAddress:     cfg.MetricsAddr,
        HealthProbeBindAddress: cfg.ProbeAddr,
        LeaderElection:         enableLeaderElection,
        LeaderElectionID:       "hcro.hybrid.io",
    })
    if err != nil {
        logger.Error("unable to start manager", "error", err)
        os.Exit(1)
    }

    // Register CRD scheme
    if err = costv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
        logger.Error("unable to add scheme", "error", err)
        os.Exit(1)
    }

    // Setup controller
    if err = (&controller.HybridWorkloadReconciler{
        Client:         mgr.GetClient(),
        Scheme:         mgr.GetScheme(),
        DecisionEngine: decisionEngine,
        NodePoolMgr:    nodePoolMgr,
    }).SetupWithManager(mgr); err != nil {
        logger.Error("unable to create controller", "error", err)
        os.Exit(1)
    }

    // Add health checks
    if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
        logger.Error("unable to set up health check", "error", err)
        os.Exit(1)
    }
    if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
        logger.Error("unable to set up ready check", "error", err)
        os.Exit(1)
    }

    // Graceful shutdown
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    logger.Info("starting manager")
    if err := mgr.Start(ctx); err != nil {
        logger.Error("problem running manager", "error", err)
        os.Exit(1)
    }

    logger.Info("controller stopped gracefully")
}

var scheme = runtime.NewScheme()

func init() {
    _ = costv1alpha1.AddToScheme(scheme)
}
```

---

### 9. Structured Error Types

**File:** `internal/errors/errors.go`

```go
package errors

import "fmt"

// ProxmoxUnavailableError — Proxmox metrics fetch failed.
// Retry: fast (10s), Proxmox may be temporarily unreachable.
type ProxmoxUnavailableError struct{ msg string }

func (e ProxmoxUnavailableError) Error() string { return e.msg }
func NewProxmoxUnavailableError(msg string) ProxmoxUnavailableError {
    return ProxmoxUnavailableError{msg: msg}
}

// VPNUnhealthyError — VPN tunnel is down, cannot manage AWS workloads.
// Retry: moderate (1min), do not attempt AWS placement.
type VPNUnhealthyError struct{ msg string }

func (e VPNUnhealthyError) Error() string { return e.msg }
func NewVPNUnhealthyError(msg string) VPNUnhealthyError {
    return VPNUnhealthyError{msg: msg}
}

// BudgetExceededError — monthly cost limit hit.
// Retry: slow (5min), wait for Proxmox capacity to free up.
type BudgetExceededError struct{ msg string }

func (e BudgetExceededError) Error() string { return e.msg }
func NewBudgetExceededError(msg string) BudgetExceededError {
    return BudgetExceededError{msg: msg}
}

// KarpenterTimeoutError — NodePool creation timed out.
// Retry: moderate (30s).
type KarpenterTimeoutError struct{ msg string }

func (e KarpenterTimeoutError) Error() string { return e.msg }
func NewKarpenterTimeoutError(msg string) KarpenterTimeoutError {
    return KarpenterTimeoutError{msg: msg}
}

// PricingAPIUnavailableError — AWS Pricing API down.
// Action: use cached price, log warning, retry (30s).
type PricingAPIUnavailableError struct{ msg string }

func (e PricingAPIUnavailableError) Error() string { return e.msg }
func NewPricingAPIUnavailableError(msg string) PricingAPIUnavailableError {
    return PricingAPIUnavailableError{msg: msg}
}
```

**Retry Semantics:**

| Error Type | RequeueAfter | Behavior |
|------------|-------------|----------|
| `ProxmoxUnavailableError` | 10s | Fast retry, Proxmox may be temporarily unreachable |
| `VPNUnhealthyError` | 1min | Don't attempt AWS placement, set `VPNHealthy` condition |
| `BudgetExceededError` | 5min | Set phase to Pending, wait for Proxmox capacity |
| `KarpenterTimeoutError` | 30s | Retry NodePool creation |
| `PricingAPIUnavailableError` | 30s | Use cached price, log warning |

---

### 10. VPN Health Checker

**File:** `internal/healthcheck/vpn_health.go`

```go
package healthcheck

import (
    "context"
    "log/slog"
    "net"
    "time"

    "github.com/samber/do/v2"
    "hybrid-cloud-optimizer/internal/config"
)

// VPNHealthChecker verifies VPN tunnel connectivity to AWS.
// Used by DecisionEngine to prevent AWS placement when tunnel is down.
type VPNHealthChecker struct {
    endpoint string        // VPN endpoint address (e.g., "10.0.1.1:51820")
    timeout  time.Duration // Health check timeout
    logger   *slog.Logger
}

func NewVPNHealthChecker(endpoint string, timeout time.Duration, logger *slog.Logger) *VPNHealthChecker {
    return &VPNHealthChecker{
        endpoint: endpoint,
        timeout:  timeout,
        logger:   logger,
    }
}

// IsHealthy checks if the VPN tunnel is reachable via TCP dial.
func (v *VPNHealthChecker) IsHealthy(ctx context.Context) bool {
    conn, err := net.DialTimeout("tcp", v.endpoint, v.timeout)
    if err != nil {
        v.logger.WarnContext(ctx, "VPN health check failed", "endpoint", v.endpoint, "error", err)
        return false
    }
    conn.Close()
    return true
}

// ProvideVPNHealthChecker registers VPNHealthChecker in DI container
func ProvideVPNHealthChecker(i do.Injector) (*VPNHealthChecker, error) {
    cfg := do.MustInvoke[*config.Config](i)
    logger := do.MustInvoke[*slog.Logger](i)
    return NewVPNHealthChecker(cfg.VPNEndpoint, 5*time.Second, logger), nil
}
```

> **Config addition**: Add `VPNEndpoint string` to Config struct and `VPN_ENDPOINT` env var (e.g., `10.0.1.1:51820`).

---

### 11. Admission Webhooks

**File:** `api/v1alpha1/hybridworkload_webhook.go`

```go
package v1alpha1

import (
    "fmt"

    "k8s.io/apimachinery/pkg/runtime"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/webhook"
    "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/validate-cost-hybrid-io-v1alpha1-hybridworkload,mutating=false,failurePolicy=fail,sideEffects=None,groups=cost.hybrid.io,resources=hybridworkloads,verbs=create;update,versions=v1alpha1,name=vhybridworkload.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &HybridWorkload{}

func (r *HybridWorkload) SetupWebhookWithManager(mgr ctrl.Manager) error {
    return ctrl.NewWebhookManagedBy(mgr).For(r).Complete()
}

// ValidateCreate implements webhook.Validator
func (r *HybridWorkload) ValidateCreate() (admission.Warnings, error) {
    return r.validate()
}

// ValidateUpdate implements webhook.Validator
func (r *HybridWorkload) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
    return r.validate()
}

// ValidateDelete implements webhook.Validator
func (r *HybridWorkload) ValidateDelete() (admission.Warnings, error) {
    return nil, nil
}

func (r *HybridWorkload) validate() (admission.Warnings, error) {
    // Priority validation (backup for CRD schema validation)
    validPriorities := map[string]bool{"low": true, "medium": true, "high": true}
    if !validPriorities[r.Spec.Priority] {
        return nil, fmt.Errorf("spec.priority must be one of: low, medium, high (got %q)", r.Spec.Priority)
    }

    // Budget validation
    if r.Spec.MaxMonthlyCostUSD < 0 {
        return nil, fmt.Errorf("spec.maxMonthlyCostUSD must be >= 0 (got %.2f)", r.Spec.MaxMonthlyCostUSD)
    }

    // Resource requirements validation
    cpu := r.Spec.Resources.Requests.Cpu()
    mem := r.Spec.Resources.Requests.Memory()
    if cpu.IsZero() {
        return nil, fmt.Errorf("spec.resources.requests.cpu must be > 0")
    }
    if mem.IsZero() {
        return nil, fmt.Errorf("spec.resources.requests.memory must be > 0")
    }

    // Capacity type validation (backup for CRD schema validation)
    if r.Spec.CapacityType != "" {
        validTypes := map[string]bool{"spot": true, "on-demand": true}
        if !validTypes[r.Spec.CapacityType] {
            return nil, fmt.Errorf("spec.capacityType must be one of: spot, on-demand (got %q)", r.Spec.CapacityType)
        }
    }

    return nil, nil
}
```

**Main.go addition** — register webhook after controller setup:

```go
    // Setup webhook
    if err = (&costv1alpha1.HybridWorkload{}).SetupWebhookWithManager(mgr); err != nil {
        logger.Error("unable to create webhook", "error", err)
        os.Exit(1)
    }
```

---

### 12. Cost Savings CLI (kubectl plugin)

**File:** `cmd/kubectl-hcro/main.go`

```go
package main

import (
    "context"
    "fmt"
    "os"
    "text/tabwriter"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/dynamic"
    "k8s.io/apimachinery/pkg/runtime/schema"
    ctrl "sigs.k8s.io/controller-runtime"
)

// Usage: kubectl hcro savings [--namespace <ns>]
// Aggregates cost savings across all HybridWorkload resources.
func main() {
    cfg := ctrl.GetConfigOrDie()
    client, err := dynamic.NewForConfig(cfg)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
        os.Exit(1)
    }

    gvr := schema.GroupVersionResource{
        Group:    "cost.hybrid.io",
        Version:  "v1alpha1",
        Resource: "hybridworkloads",
    }

    list, err := client.Resource(gvr).Namespace("").List(context.Background(), metav1.ListOptions{})
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error listing HybridWorkloads: %v\n", err)
        os.Exit(1)
    }

    w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
    fmt.Fprintln(w, "NAME\tNAMESPACE\tPLATFORM\tEST. COST\tSAVINGS vs AWS")

    var totalSavings, totalAWSCost float64
    proxmoxCount, awsCount := 0, 0

    for _, item := range list.Items {
        status := item.Object["status"].(map[string]interface{})
        spec := item.Object["spec"].(map[string]interface{})
        platform := fmt.Sprintf("%v", status["recommendedPlatform"])
        cost := status["estimatedMonthlyCostUSD"].(float64)
        name := item.GetName()
        ns := item.GetNamespace()

        // Estimate what it would cost on AWS for comparison
        awsEquivalent := cost
        if platform == "proxmox" {
            awsEquivalent = 15.0 // Approximate t3.small monthly cost
            totalSavings += awsEquivalent
            proxmoxCount++
        } else {
            totalAWSCost += cost
            awsCount++
        }

        fmt.Fprintf(w, "%s\t%s\t%s\t$%.2f\t$%.2f\n", name, ns, platform, cost, awsEquivalent-cost)
    }

    w.Flush()
    fmt.Printf("\n--- Summary ---\n")
    fmt.Printf("Workloads on Proxmox: %d (savings: $%.2f/mo)\n", proxmoxCount, totalSavings)
    fmt.Printf("Workloads on AWS: %d (cost: $%.2f/mo)\n", awsCount, totalAWSCost)
    fmt.Printf("Net monthly savings: $%.2f\n", totalSavings-totalAWSCost)
}
```

**Installation:**
```bash
go build -o kubectl-hcro ./cmd/kubectl-hcro
sudo mv kubectl-hcro /usr/local/bin/
kubectl hcro savings
```

---

## 🛤️ Implementation Roadmap

### Phase 1: Foundation (Week 1-2)

**Goal:** Basic K8s controller with CRD

- [ ] Initialize Kubebuilder project
- [ ] Define `HybridWorkload` CRD
- [ ] Implement basic reconciler (no logic yet)
- [ ] Setup DI with `samber/do`
- [ ] Add unit tests for config loading
- [ ] Create `.pre-commit-config.yaml`

**Deliverable:** Controller can watch `HybridWorkload` CR and print events.

---

### Phase 2: Core Decision Logic (Week 3-4)

**Goal:** Working decision engine with hysteresis and resilience

- [ ] Implement `ProxmoxMetricsClient` (K8s Metrics API)
- [ ] Implement `AWSPricingClient` (AWS SDK)
- [ ] Implement `DecisionEngine` with hysteresis thresholds (85% scale-out / 70% scale-back)
- [ ] Implement `internal/errors/errors.go` — structured error types with retry semantics
- [ ] Implement `VPNHealthChecker` (TCP dial to VPN endpoint)
- [ ] Add unit tests (table-driven) for decision logic, including hysteresis scenarios
- [ ] Integration test with `envtest`

**Deliverable:** Controller makes correct placement decisions with flap prevention (logs only).

---

### Phase 3: Karpenter Integration (Week 5)

**Goal:** Automatic AWS burst with dry-run support

- [ ] Implement `NodePoolManager` (create/update/delete)
- [ ] Update reconciler to create Karpenter NodePools
- [ ] Implement dry-run mode (`hcro.io/dry-run` annotation)
- [ ] Enable leader election in manager options (default: true)
- [ ] Test NodePool creation in kind cluster
- [ ] Add status updates to `HybridWorkload` CR

**Deliverable:** Controller creates AWS EC2 nodes via Karpenter when needed. Supports dry-run for safe evaluation.

---

### Phase 4: Production Readiness (Week 6)

**Goal:** Deploy to real cluster with admission validation

- [ ] Add Prometheus metrics (workload_placement_total, aws_cost_estimate)
- [ ] Implement graceful shutdown
- [ ] Add health checks (`/healthz`, `/readyz`)
- [ ] Implement validating webhook for `HybridWorkload` CRD
- [ ] Create RBAC manifests
- [ ] Write E2E test (kind cluster + mock workload)

**Deliverable:** Production-ready controller with observability and admission validation.

---

### Phase 5: Documentation & Polish (Week 7)

**Goal:** Portfolio-ready project

- [ ] Write README with architecture diagrams
- [ ] Add example `HybridWorkload` manifests
- [ ] Create Grafana dashboard JSON (`config/grafana/hcro-dashboard.json`) — placement trends, cost estimates, decision latency
- [ ] Build `kubectl-hcro` plugin (cost savings report)
- [ ] Record demo video (YouTube)
- [ ] Publish to GitHub

**Deliverable:** Complete portfolio project for DevSecOps interviews.

---

## 🛠️ Technology Stack

| Component | Technology | Justification |
|-----------|-----------|---------------|
| **Language** | Go 1.23+ | Industry standard for K8s controllers |
| **K8s Framework** | Kubebuilder + controller-runtime | Official K8s tooling |
| **DI** | samber/do v2 | Production DI pattern (go-expert skill) |
| **Logging** | slog (stdlib) | Native structured logging |
| **Metrics** | Prometheus (controller-runtime) | K8s ecosystem standard |
| **AWS SDK** | aws-sdk-go-v2 | Official AWS SDK |
| **Testing** | envtest, testify | K8s integration testing |
| **Karpenter** | v0.33+ | AWS-native K8s autoscaler |

---

## 🧪 Development Environment

### Prerequisites

```bash
# Install Go 1.23+
go version

# Install Kubebuilder
curl -L -o kubebuilder "https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)"
chmod +x kubebuilder && mv kubebuilder /usr/local/bin/

# Install kind (for local testing)
go install sigs.k8s.io/kind@latest

# Install kubectl
# https://kubernetes.io/docs/tasks/tools/

# Install pre-commit
pip install pre-commit
```

### Setup

```bash
# 1. Initialize project
cd /home/abevz/github/hybrid-cloud-optimizer
kubebuilder init --domain hybrid.io --repo hybrid-cloud-optimizer

# 2. Create CRD
kubebuilder create api --group cost --version v1alpha1 --kind HybridWorkload

# 3. Install dependencies
go mod tidy

# 4. Setup pre-commit
pre-commit install

# 5. Create kind cluster with Proxmox-like labels
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
  labels:
    platform: proxmox
    cost: zero
- role: worker
  labels:
    platform: proxmox
    cost: zero
EOF

# 6. Install metrics-server (for Proxmox metrics simulation)
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# 7. Install Karpenter (optional, for testing NodePool creation)
# Follow: https://karpenter.sh/docs/getting-started/
```

---

## 🔐 VPN Setup: Connecting Proxmox and AWS

### Overview

Hybrid K8s cluster требует **secure network connectivity** между Proxmox (on-premise) и AWS (cloud). Для MVP мы сравниваем три решения по критериям: стоимость, сложность, production-readiness.

### 💰 Cost Comparison

| Solution | Monthly Cost | Setup Time | Production-Ready | Recommendation |
|----------|-------------|------------|------------------|----------------|
| **AWS Site-to-Site VPN** | $63+ | 2-4 hours | ✅ Yes | ❌ Expensive for MVP |
| **WireGuard (Free Tier)** | **$0** (year 1) | 30 min | ✅ Yes | ✅ **Best for new AWS accounts** |
| **WireGuard (t4g.nano)** | $3-5 | 30 min | ✅ Yes | ✅ **Best after Free Tier** |
| **Tailscale** | $0 (Free tier) | 5 min | ⚠️ Yes (3rd-party) | ✅ **Fastest POC** |
| **OpenVPN на EC2** | $3-5 | 1-2 hours | ✅ Yes | ⚠️ Slower than WireGuard |

**💡 Data Transfer Costs (all options except Tailscale):**
- **First 100 GB/месяц:** $0 (AWS Free Tier permanent)
- **101-1024 GB/месяц:** $0.09/GB
- **Real hybrid K8s traffic (optimized):** ~4-5 GB/месяц
- **Cost:** $0/мес ✅ (далеко в пределах Free Tier лимита)

### 📊 Decision Matrix

**For MVP Development:** Используй **Tailscale** (быстрый старт, бесплатно)
**For Production Demo/Portfolio:** Используй **WireGuard** (контроль, cost story для интервью)

---

### Option 1: Tailscale (Recommended for MVP Start)

**Architecture:**
```
Proxmox Nodes (Tailscale clients)
      ↓
Tailscale Mesh Network (координация через Tailscale servers)
      ↓
AWS EC2 Worker Nodes (Tailscale clients)
```

**Pros:**
- ✅ **$0/месяц** (Free tier: до 100 devices)
- ✅ **5 минут setup** (apt install + tailscale up)
- ✅ **Zero config NAT traversal** (работает за любым NAT/firewall)
- ✅ **Mesh VPN**: каждая нода видит друг друга напрямую
- ✅ **ACL support**: можно ограничить доступ между группами нод

**Cons:**
- ⚠️ Third-party dependency (Tailscale координирует peer discovery)
- ⚠️ Latency может быть выше если нет direct p2p connection (fallback на DERP relay)

**Setup Instructions:**

```bash
# ===== На Proxmox Master Node =====
# 1. Установить Tailscale
curl -fsSL https://tailscale.com/install.sh | sh

# 2. Авторизоваться (откроется браузер)
tailscale up

# 3. Получить Tailscale IP
tailscale ip -4
# Output: 100.64.0.1

# 4. Настроить kube-apiserver слушать на Tailscale IP
# /etc/kubernetes/manifests/kube-apiserver.yaml
spec:
  containers:
  - command:
    - kube-apiserver
    - --advertise-address=100.64.0.1  # Tailscale IP
    - --bind-address=0.0.0.0

# 5. Restart kubelet
systemctl restart kubelet


# ===== На Proxmox Worker Nodes =====
curl -fsSL https://tailscale.com/install.sh | sh
tailscale up


# ===== На AWS EC2 Instances (добавить в User Data) =====
#!/bin/bash
# Install Tailscale
curl -fsSL https://tailscale.com/install.sh | sh

# Connect (используй pre-auth key из Tailscale admin console)
tailscale up --authkey=tskey-auth-xxxxxxxxxxxxx \
             --advertise-routes=172.16.0.0/16  # AWS VPC CIDR

# Configure kubelet to use Tailscale IP
TAILSCALE_IP=$(tailscale ip -4)
echo "KUBELET_KUBEADM_ARGS=--node-ip=$TAILSCALE_IP" > /var/lib/kubelet/kubeadm-flags.env
systemctl restart kubelet


# ===== Join AWS node to Proxmox cluster =====
# На Proxmox master получить join command:
kubeadm token create --print-join-command

# На AWS EC2:
kubeadm join 100.64.0.1:6443 \  # Tailscale IP Proxmox master
  --token <token> \
  --discovery-token-ca-cert-hash sha256:<hash>


# ===== Verify connectivity =====
# На Proxmox:
kubectl get nodes -o wide
# Должны увидеть AWS ноды с Tailscale IPs (100.x.x.x)

# Проверить pod-to-pod connectivity:
kubectl run test-proxmox --image=nginx -n default \
  --overrides='{"spec":{"nodeSelector":{"platform":"proxmox"}}}'

kubectl run test-aws --image=nginx -n default \
  --overrides='{"spec":{"nodeSelector":{"platform":"aws"}}}'

kubectl exec test-proxmox -- curl http://$(kubectl get pod test-aws -o jsonpath='{.status.podIP}')
```

**Cost Breakdown:**
- Tailscale Free tier: **$0/месяц**
- Data transfer: P2P (не через Tailscale relay, если есть direct connectivity)
- **Total: $0/месяц**

**When to Use:**
- 🎯 Week 1-3: Rapid development (фокус на controller logic, не на network debugging)
- 🎯 POC для быстрой демонстрации концепции

---

### Option 2: WireGuard на EC2 (Recommended for Production Demo)

**Architecture:**
```
Proxmox Nodes (WireGuard clients)
      ↓ UDP 51820
EC2 t2.micro/t3.micro (WireGuard server, router)
      ↓ Private IPs
AWS VPC (EC2 worker nodes)
```

**💰 Cost Options:**

| Instance Type | vCPU | RAM | Network | Free Tier | After 12 months |
|---------------|------|-----|---------|-----------|-----------------|
| **t2.micro** | 1 | 1 GB | Low-Moderate | ✅ **$0/мес** | $7.49/мес |
| **t3.micro** | 2 | 1 GB | Up to 5 Gbps | ✅ **$0/мес** | $8.76/мес |
| **t4g.nano** | 2 | 512 MB | Up to 5 Gbps | ❌ $3.04/мес | $3.04/мес |

**💡 Recommended Strategy:**
```
Year 1 (new AWS account):  t2.micro or t3.micro (Free Tier)
  └─ Total: $0/месяц (только data transfer > 100 GB)

Year 2+:                   t4g.nano (ARM64, дешевле долгосрочно)
  └─ Total: $3.04/месяц
```

**Pros:**
- ✅ **$0-3/месяц** (t2.micro Free Tier первый год, затем t4g.nano)
- ✅ **Полный контроль** конфигурации
- ✅ **Высокая производительность** (WireGuard kernel module, multi-threaded)
- ✅ **Production-ready** (используется в production у многих компаний)
- ✅ **Показывает технические навыки** (network engineering, VPN protocols)
- ✅ **Cost optimization story** для интервью ($0 vs $63 AWS VPN)

**Cons:**
- ⚠️ Ручная настройка (но можно автоматизировать через Terraform)
- ⚠️ Single point of failure (можно решить через Auto Scaling Group + Route53 health checks)
- ⚠️ Free Tier ограничен 750 часами/месяц (только 1 инстанс)
- ⚠️ Data transfer > 100 GB/месяц платный ($0.09/GB)

**🚨 Free Tier Limits (12 месяцев для новых аккаунтов):**
- EC2: **750 часов/месяц** t2.micro или t3.micro
- Data Transfer OUT: **100 GB/месяц** (aggregated)
- EBS: **30 GB** General Purpose SSD
- Elastic IP: **Бесплатно** если attached к running instance

**Setup Instructions:**

#### Step 1: Deploy WireGuard Server в AWS

**Terraform код (Free Tier variant):**
```hcl
# wireguard.tf

# Variables для гибкости
variable "use_free_tier" {
  description = "Use Free Tier eligible instance (t2.micro). Set to false for t4g.nano after 12 months."
  type        = bool
  default     = true  # Начни с Free Tier
}

variable "your_public_ip" {
  description = "Your public IP for SSH access (security)"
  type        = string
  default     = "0.0.0.0/0"  # Замени на свой IP!
}

# Instance selection
locals {
  instance_config = var.use_free_tier ? {
    type = "t2.micro"
    ami  = "ami-0c55b159cbfafe1f0"  # Ubuntu 22.04 x86_64 (обнови для региона)
  } : {
    type = "t4g.nano"
    ami  = "ami-0a1b2c3d4e5f67890"  # Ubuntu 22.04 ARM64 (обнови для региона)
  }
}

resource "aws_instance" "wireguard_server" {
  ami           = local.instance_config.ami
  instance_type = local.instance_config.type
  
  vpc_security_group_ids = [aws_security_group.wireguard.id]
  subnet_id              = aws_subnet.public.id
  
  user_data = file("${path.module}/wireguard_server_init.sh")
  
  tags = {
    Name = "hcro-wireguard-server"
    Project = "hybrid-cloud-optimizer"
  }
}

resource "aws_security_group" "wireguard" {
  name        = "hcro-wireguard-sg"
  description = "Allow WireGuard UDP traffic"
  vpc_id      = aws_vpc.main.id
  
  # WireGuard UDP port
  ingress {
    from_port   = 51820
    to_port     = 51820
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  # SSH (для управления, можно ограничить твоим IP)
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["YOUR_IP/32"]
  }
  
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_eip" "wireguard" {
  instance = aws_instance.wireguard_server.id
  domain   = "vpc"
}

output "wireguard_public_ip" {
  value = aws_eip.wireguard.public_ip
}
```

**User Data Script:** `wireguard_server_init.sh`
```bash
#!/bin/bash
set -e

# Install WireGuard
apt-get update
apt-get install -y wireguard iptables

# Generate server keys
wg genkey | tee /etc/wireguard/server_private.key | wg pubkey > /etc/wireguard/server_public.key
chmod 600 /etc/wireguard/server_private.key

# Configure WireGuard
cat > /etc/wireguard/wg0.conf <<EOF
[Interface]
Address = 10.200.0.1/24
ListenPort = 51820
PrivateKey = $(cat /etc/wireguard/server_private.key)

# NAT для трафика из VPN в AWS VPC
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

# Proxmox Master Node
[Peer]
PublicKey = PROXMOX_MASTER_PUBLIC_KEY_PLACEHOLDER
AllowedIPs = 10.200.0.2/32

# Proxmox Worker 1
[Peer]
PublicKey = PROXMOX_WORKER1_PUBLIC_KEY_PLACEHOLDER
AllowedIPs = 10.200.0.3/32

# Proxmox Worker 2
[Peer]
PublicKey = PROXMOX_WORKER2_PUBLIC_KEY_PLACEHOLDER
AllowedIPs = 10.200.0.4/32
EOF

# Enable IP forwarding
echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
sysctl -p

# Start WireGuard
systemctl enable wg-quick@wg0
systemctl start wg-quick@wg0

# Log server public key (найдешь в /var/log/cloud-init-output.log)
echo "WireGuard Server Public Key:"
cat /etc/wireguard/server_public.key
```

#### Step 2: Configure Proxmox Nodes

**На каждой Proxmox ноде (master + workers):**

```bash
# 1. Install WireGuard
apt-get install -y wireguard resolvconf

# 2. Generate keys
wg genkey | tee /etc/wireguard/client_private.key | wg pubkey > /etc/wireguard/client_public.key
chmod 600 /etc/wireguard/client_private.key

# 3. Сохрани public key (нужен для добавления в server config)
cat /etc/wireguard/client_public.key

# 4. Configure client
cat > /etc/wireguard/wg0.conf <<EOF
[Interface]
Address = 10.200.0.2/24  # Уникальный IP для каждой ноды (10.200.0.2, .3, .4)
PrivateKey = $(cat /etc/wireguard/client_private.key)

[Peer]
PublicKey = AWS_SERVER_PUBLIC_KEY_FROM_STEP1
Endpoint = AWS_EIP_FROM_TERRAFORM:51820
AllowedIPs = 10.200.0.0/24, 172.16.0.0/16  # VPN subnet + AWS VPC CIDR
PersistentKeepalive = 25
EOF

# 5. Start WireGuard
systemctl enable wg-quick@wg0
systemctl start wg-quick@wg0

# 6. Verify connectivity
ping 10.200.0.1  # WireGuard server
ping 172.16.0.10 # AWS VPC private IP (EC2 worker node)
```

#### Step 3: Update AWS Server Config

**После генерации ключей на Proxmox нодах:**

```bash
# На AWS WireGuard server
# Добавить Proxmox peer public keys в /etc/wireguard/wg0.conf
wg set wg0 peer PROXMOX_MASTER_PUBLIC_KEY allowed-ips 10.200.0.2/32
wg set wg0 peer PROXMOX_WORKER1_PUBLIC_KEY allowed-ips 10.200.0.3/32

# Restart WireGuard
systemctl restart wg-quick@wg0

# Verify connections
wg show
```

#### Step 4: K8s Configuration

**На Proxmox Master (kube-apiserver):**
```yaml
# /etc/kubernetes/manifests/kube-apiserver.yaml
spec:
  containers:
  - command:
    - kube-apiserver
    - --advertise-address=10.200.0.2  # WireGuard IP Proxmox master
```

**На AWS EC2 Worker Nodes (kubelet):**
```bash
# User Data для AWS EC2
#!/bin/bash
# Установить WireGuard client на EC2 (опционально, если нужен direct access)
# Либо использовать AWS VPC routing через WireGuard server

# Kubelet config
echo "KUBELET_EXTRA_ARGS=--node-ip=$(hostname -I | awk '{print $1}')" > /etc/default/kubelet
systemctl restart kubelet
```

**Join command:**
```bash
# На AWS EC2
kubeadm join 10.200.0.2:6443 \  # WireGuard IP Proxmox master
  --token <token> \
  --discovery-token-ca-cert-hash sha256:<hash>
```

**Cost Breakdown (Free Tier - Year 1):**
- EC2 t2.micro/t3.micro: **$0/месяц** (750 hours Free Tier)
- Elastic IP: **$0** (если attached к running instance)
- EBS 8 GB: **$0** (в пределах 30 GB лимита)
- Data Transfer Out (50 GB/мес hybrid K8s): **$0** (в пределах 100 GB лимита)
- **Total Year 1: $0/месяц** 🎉

**Cost Breakdown (After Free Tier or t4g.nano):**
- EC2 t4g.nano: **$3.04/месяц**
- Elastic IP: **$0** (если attached)
- Data Transfer Out (50 GB/мес): **$0** (AWS always includes 100 GB free)
- **Total Year 2+: $3.04/месяц**

**Comparison:**
```
Free Tier Strategy (t2.micro year 1 → t4g.nano year 2):
  Year 1: $0 × 12 = $0
  Year 2: $3.04 × 12 = $36.48
  Total 2 years: $36.48

t4g.nano from Day 1:
  Year 1: $3.04 × 12 = $36.48
  Year 2: $3.04 × 12 = $36.48
  Total 2 years: $72.96

Savings with Free Tier: $36.48 (50% off!)
```

**When to Use:**
- 🎯 **Year 1:** t2.micro или t3.micro (Free Tier, новый AWS аккаунт)
- 🎯 **Year 2+:** Переключиться на t4g.nano (ARM64, лучше performance/cost ratio)
- 🎯 **Интервью:** Cost optimization story + Terraform automation

---

### Option 3: AWS Site-to-Site VPN (NOT Recommended for MVP)

**Cost:** $36/месяц (VPN connection) + $27/месяц (data transfer) = **$63/месяц**

**Why Expensive:**
- AWS VPN Gateway стоит $0.05/час **независимо от использования**
- 720 часов/месяц × $0.05 = $36 fixed cost

**When to Use:**
- ❌ Не для MVP (overkill)
- ✅ Enterprise production с compliance требованиями (IPsec, BGP routing, high availability)

---

### Roadmap: VPN Evolution

**Рекомендуемый путь для проекта:**

```
Week 1-3: Tailscale ($0)
  ↓ Быстрый старт, фокус на controller
Week 4-5: WireGuard ($3/мес)
  ↓ Production demo, Terraform IaC
Week 6+: Cost optimization story для интервью
  ↓ "$3 vs $63" (показываешь engineering decision making)
```

**Demo для интервью:**
1. **Show Terraform код** для WireGuard setup
2. **Explain trade-offs**: WireGuard (контроль, $3) vs Tailscale (простота, $0) vs AWS VPN (enterprise, $63)
3. **Cost optimization story**: "Сэкономили $60/месяц, выбрав WireGuard для hybrid K8s"

---

### Network Diagram

```
┌─────────────────────────────────┐
│  Proxmox (On-Premise)           │
│  ┌───────────────────────────┐  │
│  │ K8s Control Plane         │  │
│  │ IP: 10.200.0.2 (WG)       │  │
│  └───────────────────────────┘  │
│  ┌───────────────────────────┐  │
│  │ Worker Nodes              │  │
│  │ IP: 10.200.0.3-0.5 (WG)   │  │
│  └───────────────────────────┘  │
└────────────┬────────────────────┘
             │ WireGuard Tunnel
             │ (UDP 51820)
             ▼
┌─────────────────────────────────┐
│  AWS Cloud                      │
│  ┌───────────────────────────┐  │
│  │ WireGuard Server          │  │
│  │ EC2 t4g.nano              │  │
│  │ Public IP: 3.x.x.x        │  │
│  │ VPN IP: 10.200.0.1        │  │
│  └─────────┬─────────────────┘  │
│            │ Route to VPC      │
│            ▼                   │
│  ┌───────────────────────────┐  │
│  │ EC2 Worker Nodes          │  │
│  │ VPC IP: 172.16.0.x        │  │
│  └───────────────────────────┘  │
└─────────────────────────────────┘
```

---

### 💡 Data Transfer Optimization для Free Tier

**Проблема:** Free Tier дает только **100 GB/месяц** data transfer OUT.

**Решение:** Минимизируй трафик через VPN.

#### Стратегии оптимизации:

**1. Используй VPN только для control plane traffic**
```yaml
# kube-apiserver на Proxmox слушает на WireGuard IP
# Только control plane traffic через VPN:
#   - kubelet → API server heartbeats
#   - kubectl commands
#   - Controller reconciliation

# Pod-to-Pod traffic НЕ через VPN:
#   - Поды на Proxmox не общаются с подами на AWS напрямую
#   - Используй Service Mesh (Istio/Linkerd) только если нужно
```

**2. Metrics & Logs — локально**
```yaml
# Prometheus на Proxmox собирает метрики только с Proxmox нод
# Prometheus на AWS собирает метрики только с AWS нод
# Grafana объединяет через federation (минимальный traffic)

# Loki/ELK — раздельные инстансы
```

**3. Image Registry — используй ECR в AWS**
```bash
# Для AWS нод — пулить из ECR (free in-region transfer)
# Для Proxmox нод — локальный registry или Harbor

# Избегай pulling multi-GB images через VPN!
```

**Реальный расчет VPN трафика (детально):**
```
1. Kubelet heartbeats (5 AWS nodes):
   - Frequency: каждые 10 секунд
   - Payload: ~3 KB per update
   - Traffic: 5 nodes × 3 KB × 8640 updates/день = ~130 MB/день
   - Per month: ~4 GB/месяц

2. Pod status updates:
   - Frequency: ~70 events/день (create, delete, health checks)
   - Payload: ~5 KB per event
   - Per month: ~10 MB/месяц

3. HCRO Controller → AWS API:
   - AWS Pricing API: 1 call/час × 50 KB = ~36 MB/месяц
   - Karpenter NodePool CRUD: 5 ops/день × 2 KB = ~0.3 MB/месяц
   - Per month: ~40 MB/месяц

4. kubectl commands (dev usage):
   - Commands: 10/день × 10 KB = ~3 MB/месяц
   - Logs fetching: 5/день × 100 KB = ~15 MB/месяц
   - Per month: ~18 MB/месяц

5. Prometheus metrics (ТОЛЬКО если scrape через VPN):
   - ⚠️ 5 nodes × 100 KB × 5760 scrapes/день = ~86 GB/месяц
   - ✅ Solution: Local Prometheus на AWS → 0 GB через VPN

6. Image pulls (ТОЛЬКО если registry на Proxmox):
   - ⚠️ 200 MB/image × 5 pulls/неделя × 4 = ~4 GB/месяц
   - ✅ Solution: Use ECR (AWS) → 0 GB через VPN

────────────────────────────────────────────────────
ИТОГО (с правильной архитектурой):
  Kubelet + Pod updates:    ~4 GB/месяц
  Controller + kubectl:     ~60 MB/месяц
  Prometheus (local):       0 GB
  Images (ECR):             0 GB
────────────────────────────────────────────────────
TOTAL:                      ~4-5 GB/месяц ✅
                           (в пределах 100 GB Free Tier!)
```

**Бонус:** Первые **100 GB data transfer OUT всегда бесплатны** в AWS (даже после Free Tier 12 месяцев!)

---

### Security Considerations

**Для Production:**

1. **Limit SSH access** к WireGuard server (только твой IP):
   ```hcl
   ingress {
     from_port   = 22
     to_port     = 22
     protocol    = "tcp"
     cidr_blocks = ["YOUR_PUBLIC_IP/32"]
   }
   ```

2. **Rotate WireGuard keys** периодически:
   ```bash
   # Generate new server keys
   wg genkey | tee /etc/wireguard/server_private.key.new | wg pubkey
   
   # Update config, restart
   systemctl restart wg-quick@wg0
   ```

3. **Enable AWS VPC Flow Logs** для audit:
   ```hcl
   resource "aws_flow_log" "vpc" {
     vpc_id          = aws_vpc.main.id
     traffic_type    = "ALL"
     log_destination = aws_cloudwatch_log_group.vpc_logs.arn
   }
   ```

4. **Implement monitoring** (CloudWatch Alarms):
   ```hcl
   resource "aws_cloudwatch_metric_alarm" "wireguard_cpu" {
     alarm_name          = "hcro-wireguard-high-cpu"
     comparison_operator = "GreaterThanThreshold"
     evaluation_periods  = 2
     metric_name         = "CPUUtilization"
     namespace           = "AWS/EC2"
     period              = 300
     statistic           = "Average"
     threshold           = 80
     
     dimensions = {
       InstanceId = aws_instance.wireguard_server.id
     }
   }
   ```

---

## 🧪 Testing Strategy

### 1. Unit Tests (Table-Driven)

**File:** `internal/scheduler/decision_engine_test.go`

```go
package scheduler_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "hybrid-cloud-optimizer/api/v1alpha1"
    "hybrid-cloud-optimizer/internal/scheduler"
)

func TestDecisionEngine_Decide(t *testing.T) {
    tests := []struct {
        name                string
        priority            string
        proxmoxUtilization  float64
        budget              float64
        expectedPlatform    string
        expectedKarpenter   bool
    }{
        {
            name:               "high priority always AWS",
            priority:           "high",
            proxmoxUtilization: 0.5,
            budget:             100,
            expectedPlatform:   "aws",
            expectedKarpenter:  true,
        },
        {
            name:               "low priority with free Proxmox capacity",
            priority:           "low",
            proxmoxUtilization: 0.6,
            budget:             50,
            expectedPlatform:   "proxmox",
            expectedKarpenter:  false,
        },
        {
            name:               "Proxmox overloaded, burst to AWS",
            priority:           "medium",
            proxmoxUtilization: 0.85,
            budget:             100,
            expectedPlatform:   "aws",
            expectedKarpenter:  true,
        },
        {
            name:               "no budget, must wait",
            priority:           "low",
            proxmoxUtilization: 0.9,
            budget:             0,
            expectedPlatform:   "pending",
            expectedKarpenter:  false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Mock dependencies
            engine := &scheduler.DecisionEngine{
                // ... setup mocks
            }

            workload := &v1alpha1.HybridWorkload{
                Spec: v1alpha1.HybridWorkloadSpec{
                    Priority:          tt.priority,
                    MaxMonthlyCostUSD: tt.budget,
                },
            }

            decision, err := engine.Decide(context.Background(), workload)
            assert.NoError(t, err)
            assert.Equal(t, tt.expectedPlatform, decision.Platform)
            assert.Equal(t, tt.expectedKarpenter, decision.RequiresKarpenterPool)
        })
    }
}
```

### 2. Integration Tests (envtest)

```go
package controller_test

import (
    "context"
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestControllerIntegration(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Controller Integration Suite")
}

var _ = Describe("HybridWorkload Controller", func() {
    It("should create Karpenter NodePool for high priority", func() {
        // Create HybridWorkload CR
        // Wait for reconciliation
        // Assert NodePool created
    })
})
```

### 3. E2E Test (kind cluster)

```bash
# Run E2E test
make e2e-test
```

---

## ❓ FAQ: Proxmox + AWS Architecture

### Q1: Как реализована связка Proxmox + AWS?

**A:** Это **hybrid Kubernetes cluster**:

1. **Control Plane (мастер)**: на Proxmox (on-premise)
2. **Worker Nodes**:
   - **Статичные**: 3-5 VM на Proxmox (platform=proxmox)
   - **Динамические**: EC2 инстансы в AWS (platform=aws, создаются Karpenter)

**Network connectivity:**
- VPN или AWS Direct Connect между Proxmox и AWS VPC
- Kubelet на AWS нодах подключается к API server на Proxmox

**Example:**
```bash
kubectl get nodes

NAME                  STATUS   ROLES           LABELS
proxmox-node-1        Ready    <none>          platform=proxmox,cost=zero
proxmox-node-2        Ready    <none>          platform=proxmox,cost=zero
ip-10-0-1-23.ec2      Ready    <none>          platform=aws,cost=paid,karpenter.sh/nodepool=hcro-burst
```

---

### Q2: Нужен ли Kubernetes в Proxmox?

**A:** Да, но **только worker nodes**. Control plane может быть на Proxmox или в AWS.

**MVP подход:**
- **Control Plane**: на Proxmox (экономия, подходит для dev/staging)
- **Worker Nodes**: Proxmox (базовые) + AWS (burst)

**Production вариант:**
- **Control Plane**: AWS EKS (managed, HA)
- **Worker Nodes**: Proxmox (dev) + AWS (prod)

---

### Q3: Как контроллер принимает решения?

**Decision tree:**

```
1. IF priority == "high"
   → ALWAYS AWS (reliability > cost)

2. ELSE IF proxmox_utilization < 85% (scale-out threshold)
   → Proxmox (free capacity)

2b. ELSE IF currently on AWS AND proxmox_utilization < 70% (scale-back threshold)
   → Return to Proxmox (hysteresis)

3. ELSE IF vpn_healthy AND budget > 0 AND aws_cost < budget
   → AWS (burst via Karpenter)

4. ELSE
   → WAIT (queue workload, status=Pending)
```

**Metrics sources:**
- **Proxmox utilization**: Kubernetes Metrics API (`kubectl top nodes`)
- **AWS cost**: AWS Pricing API (hourly rate → monthly estimate)

---

### Q4: Что делает Karpenter?

**A:** Karpenter автоматически создает EC2 инстансы в AWS когда:
1. Controller решил: "нужен AWS"
2. Controller создал `NodePool` CR
3. Karpenter подхватывает `NodePool` и:
   - Запускает EC2 инстанс нужного типа (t3.medium, t3.large, ...)
   - Подключает его к K8s кластеру (kubelet → API server)
   - Применяет labels (`platform=aws`, `capacity-type=spot`)

**Lifecycle:**
```
HybridWorkload CR applied
     ↓
Controller reconciles
     ↓
Decision: "AWS burst needed"
     ↓
Controller creates Karpenter NodePool CR
     ↓
Karpenter launches EC2 instance
     ↓
New node joins cluster
     ↓
Pod scheduled on new node
```

---

### Q5: Можно ли использовать EKS вместо self-hosted K8s?

**A:** Да, но trade-offs:

| Approach | Control Plane | Worker Nodes | Cost | Complexity |
|----------|--------------|--------------|------|------------|
| **Self-hosted (MVP)** | Proxmox | Proxmox + AWS EC2 | Low | Medium |
| **EKS Hybrid** | AWS EKS | Proxmox + AWS EC2 | Medium | High (EKS Anywhere) |
| **Full EKS** | AWS EKS | AWS EC2 only | High | Low |

**MVP рекомендация:** Self-hosted K8s на Proxmox для dev/staging.

---

## 🚀 Next Steps

1. **Start implementation:**
   ```bash
   cd /home/abevz/github/hybrid-cloud-optimizer
   kubebuilder init --domain hybrid.io --repo hybrid-cloud-optimizer
   ```

2. **Setup Proxmox cluster:**
   - Label nodes: `kubectl label node <node-name> platform=proxmox cost=zero`
   - Install metrics-server

3. **Test decision logic locally:**
   ```bash
   make run  # Run controller outside cluster
   kubectl apply -f examples/hybridworkload-low-priority.yaml
   kubectl get hybridworkload -w
   ```

4. **Deploy to staging:**
   ```bash
   make docker-build
   make deploy
   ```

---

## 📚 References

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Karpenter Documentation](https://karpenter.sh/docs/)
- [AWS Pricing API Guide](https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/price-changes.html)
- [K8s Metrics API](https://kubernetes.io/docs/tasks/debug/debug-cluster/resource-metrics-pipeline/)
- [samber/do Best Practices](~/.config/opencode/skills/samber-do/)
- [go-expert Production Patterns](~/.config/opencode/skills/go-expert/)

---

**Last Updated:** 2026-02-19  
**MVP Status:** Ready for implementation  
**Estimated Effort:** 7 weeks (1 developer)
