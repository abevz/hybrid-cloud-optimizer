# Hybrid Cloud Resource Optimizer (HCRO) - Implementation Roadmap

**Timeline**: March 10 - April 19, 2026 (40 days)
**Mode**: Balanced Showcase (50% working MVP + 50% docs/demo)
**Effort**: 6-8 hours/day after AWS SAA exam (March 10)

---

## Week 0: Post-SAA Recovery + Setup (March 10-15, 6 days)

**Goal**: Rest after certification, prepare environment, scaffold project structure.

**Effort**: 2-3 hours/day (recovery mode, no burnout!)

### Day 1 (Mon, Mar 10) - AWS SAA Exam Day
- ✅ Pass AWS SAA certification exam
- 🛌 Rest, celebrate
- 📝 Quick brain dump: useful patterns from exam prep (VPN, hybrid networking, cost optimization)

### Day 2 (Tue, Mar 11) - Environment Validation
**B-Day (Review only)**
- [ ] Validate Proxmox K8s cluster health:
  - `kubectl cluster-info` (1 CP + 2 workers)
  - Cilium status: `cilium status` (should be healthy)
  - K8s version: `kubectl version` (expecting ~1.33.x)
- [ ] Validate AWS Free Tier status:
  - Check EC2 dashboard "Free tier eligible" instances
  - Verify first 100 GB data transfer/month available
- [ ] Go environment check:
  - `go version` (need 1.23+)
  - Install kubebuilder: `curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)`
  - Install controller-gen, kustomize
- [ ] GitHub repo setup:
  - Initialize `/home/abevz/github/hybrid-cloud-optimizer` as Git repo
  - First commit: MVP.md, ROADMAP.md, .gitignore (Go template)
  - Push to GitHub (public repo for portfolio visibility)

**Deliverable**: Clean Go 1.23+ environment, kubebuilder ready, GitHub repo initialized.

### Day 3 (Wed, Mar 12) - Project Scaffold
**A-Day (Can code)**
- [ ] `kubebuilder init` - initialize project:
  ```bash
  cd /home/abevz/github/hybrid-cloud-optimizer
  kubebuilder init \
    --domain hybrid-cloud.dev \
    --repo github.com/abevz/hybrid-cloud-optimizer \
    --project-name hybrid-cloud-resource-optimizer
  ```
- [ ] Apply go-expert skill patterns:
  - Setup samber/do DI structure in `cmd/main.go`
  - Create `internal/di/` directory for providers
  - Add `.pre-commit-config.yaml` (Go hooks: gofmt, golangci-lint, gitleaks)
  - Add Makefile with: `make test`, `make lint`, `make build`, `make docker-build`
- [ ] First CRD scaffold:
  ```bash
  kubebuilder create api \
    --group scheduling \
    --version v1alpha1 \
    --kind HybridWorkload
  ```
- [ ] Commit: "chore: kubebuilder init + HybridWorkload CRD scaffold"

**Deliverable**: Kubebuilder project structure with samber/do DI, first CRD scaffolded.

### Day 4 (Thu, Mar 13) - Dependency Setup + Tailscale VPN (5 min!)
**B-Day (Review only)**
- [ ] Install dependencies:
  ```bash
  go get github.com/samber/do/v2
  go get sigs.k8s.io/controller-runtime
  go get github.com/aws/aws-sdk-go-v2/...
  go mod tidy
  ```
- [ ] Tailscale VPN setup (5-minute quickstart):
  - Install on Proxmox control plane: `curl -fsSL https://tailscale.com/install.sh | sh`
  - `tailscale up` (authenticate via browser)
  - Enable subnet routes: `tailscale up --advertise-routes=10.0.0.0/24` (Proxmox pod CIDR)
  - Install on local dev machine (for kubectl access from laptop)
  - Test: `ping <proxmox-tailscale-ip>` from laptop
  - Document Tailscale IPs in `docs/vpn-setup.md`
- [ ] First controller run test:
  - `make run` (should start controller, watch HybridWorkload CRD)
  - Create test CR: `kubectl apply -f config/samples/scheduling_v1alpha1_hybridworkload.yaml`
  - Verify reconciliation logs
- [ ] Commit: "feat: add Tailscale VPN + first controller test run"

**Deliverable**: Tailscale VPN connected (Proxmox ↔ laptop), controller runs locally, dependencies installed.

### Day 5 (Fri, Mar 14) - HybridWorkload CRD Design
**A-Day (Can code)**
- [ ] Implement HybridWorkload CRD spec (from MVP.md section 3.1):
  - `api/v1alpha1/hybridworkload_types.go`:
    - Spec: workloadType, minCost, minLatency, resourceRequirements, preferredLocation
    - Status: placementDecision, currentCost, reason, conditions
  - Run: `make manifests` (generate YAML)
  - Unit tests: `api/v1alpha1/hybridworkload_types_test.go`
- [ ] Apply go-expert security patterns:
  - No hardcoded AWS credentials in code
  - Use `os.Getenv()` with validation for AWS region
  - Add credential validation helper in `internal/aws/credentials.go`
- [ ] Commit: "feat: implement HybridWorkload CRD v1alpha1 spec + status"

**Deliverable**: HybridWorkload CRD complete with typed fields, unit tests passing.

### Day 6 (Sat, Mar 15) - Buffer Day + Pre-Sprint Prep
**Weekend (Flexible)**
- [ ] Review Week 1 plan (starting Monday)
- [ ] Optional: Read Viaene Go course Chapter 1-2 (context for next week)
- [ ] Optional: Setup Prometheus on Proxmox (if not already running)
- [ ] Create `QUICKSTART.md` for Week 1 Day 1 (March 16)
- [ ] Mental prep: English Sprint starts Monday (10h/week target)

**Deliverable**: QUICKSTART.md ready, mentally prepared for full-focus mode.

---

## Week 1: Core Controller + Tailscale Integration (March 16-22, 7 days)

**Goal**: Implement HybridWorkload controller reconciliation loop, integrate Proxmox metrics client, test with Tailscale VPN.

**Effort**: 6-8 hours/day + 2 hours/day English (total 8-10h/day)

**English Integration**: Daily commit messages in English, code comments in English, start drafting `docs/architecture.md` (broken English → self-review).

### Day 7 (Mon, Mar 16) - Controller Reconciliation Skeleton
**A-Day (Can code)**
- [ ] Morning: Read QUICKSTART.md, review go-expert skill controller patterns
- [ ] Implement controller reconciliation skeleton:
  - `internal/controller/hybridworkload_controller.go`:
    - Reconcile() function structure
    - Fetch HybridWorkload CR
    - Update status conditions (Pending → Analyzing → Scheduled)
    - Handle deletion with finalizers
  - Apply controller-runtime best practices:
    - Return `ctrl.Result{RequeueAfter: 5*time.Minute}` for periodic checks
    - Use `r.Patch()` for status updates (not Update())
    - Add logging with structured fields
- [ ] Unit tests: `internal/controller/hybridworkload_controller_test.go`
  - Test reconciliation with fake client
  - Test finalizer logic
- [ ] **English practice (2h)**: Write commit message in broken English, self-review for 5 min, then commit
- [ ] Commit: "feat: implement HybridWorkload controller reconciliation skeleton"

**Deliverable**: Controller reconciles HybridWorkload CRs, updates status, handles deletion.

### Day 8 (Tue, Mar 17) - Proxmox Metrics Client (Read-Only)
**B-Day (Review only, ask before implementing)**
- [ ] Design Proxmox metrics client interface:
  - `internal/proxmox/client.go`:
    - Interface: `GetNodeMetrics(ctx context.Context) (*NodeMetrics, error)`
    - Struct: `NodeMetrics` (CPU %, Memory %, Available Capacity)
  - Use Proxmox API or prometheus-pve-exporter (if already running)
  - Tagging strategy: Proxmox nodes tagged with `location=proxmox`
- [ ] Implement provider pattern (samber/do):
  - `internal/di/proxmox.go`:
    - `ProvideProxmoxClient(i *do.Injector) (*proxmox.Client, error)`
    - Load API endpoint from env var `PROXMOX_API_ENDPOINT`
    - Validate credentials (token auth)
  - Wire in `cmd/main.go`
- [ ] **English practice (2h)**: Document Proxmox API integration in `docs/proxmox-integration.md` (broken English)
- [ ] Ask user for review before commit

**Deliverable**: Proxmox metrics client implemented, returns node capacity data.

### Day 9 (Wed, Mar 18) - AWS Pricing Client (Stub)
**A-Day (Can code)**
- [ ] Implement AWS EC2 pricing client (stub version):
  - `internal/aws/pricing.go`:
    - `GetEC2HourlyCost(instanceType string, region string) (float64, error)`
    - Hardcoded pricing for demo: t3.micro = $0.0104/hour
    - Add comment: "// TODO: integrate AWS Pricing API in Week 3"
  - Provider: `internal/di/aws.go`
    - `ProvideAWSPricingClient(i *do.Injector) (*aws.PricingClient, error)`
    - Load region from `AWS_REGION` env var (default: us-east-1)
- [ ] Unit tests with table-driven tests (go-expert pattern):
  ```go
  func TestGetEC2HourlyCost(t *testing.T) {
      tests := []struct {
          instanceType string
          region       string
          wantCost     float64
      }{
          {"t3.micro", "us-east-1", 0.0104},
          {"t3.small", "us-east-1", 0.0208},
      }
      // ...
  }
  ```
- [ ] **English practice (2h)**: Add code comments explaining pricing logic in broken English
- [ ] Commit: "feat: add AWS EC2 pricing client (stub version)"

**Deliverable**: AWS pricing client returns hardcoded costs for t3.micro/small.

### Day 10 (Thu, Mar 19) - Decision Engine (Cost-Only Logic)
**B-Day (Review only)**
- [ ] Design Decision Engine interface:
  - `internal/engine/decision_engine.go`:
    - `Decide(ctx context.Context, workload *v1alpha1.HybridWorkload) (*PlacementDecision, error)`
    - Struct: `PlacementDecision` (Location: "proxmox"|"aws", Reason: string, EstimatedCost: float64)
  - **Simple logic for Week 1** (hysteresis deferred to Week 3):
    - If `workload.Spec.MinCost == true` → choose Proxmox (assume $0)
    - If `workload.Spec.MinCost == false` → choose AWS (faster provisioning)
    - Ignore resource capacity checks (add in Week 3)
- [ ] Implement structured error types (`internal/errors/errors.go`):
  - `ProxmoxUnavailableError` (retry: 10s)
  - `VPNUnhealthyError` (retry: 1min)
  - `BudgetExceededError` (retry: 5min)
  - `KarpenterTimeoutError` (retry: 30s)
  - `PricingAPIUnavailableError` (retry: 30s, use cached price)
- [ ] Provider: `internal/di/engine.go`
  - Inject ProxmoxClient + AWSPricingClient into DecisionEngine
- [ ] **English practice (2h)**: Write algorithm explanation in `docs/decision-logic.md` (broken English)
- [ ] Ask user for review before commit

**Deliverable**: Decision engine returns placement decision based on minCost preference.

### Day 11 (Fri, Mar 20) - Integrate Decision Engine into Controller
**A-Day (Can code)**
- [ ] Wire decision engine into controller reconciliation:
  - In `Reconcile()`:
    - Call `engine.Decide(ctx, hybridWorkload)`
    - Update `status.PlacementDecision` = decision.Location
    - Update `status.CurrentCost` = decision.EstimatedCost
    - Update `status.Reason` = decision.Reason
    - Set condition: `type: Scheduled, status: True`
  - Add structured error handling with differentiated retry intervals:
    - `ProxmoxUnavailableError` → requeue 10s
    - `VPNUnhealthyError` → requeue 1min, set `VPNHealthy` condition
    - `BudgetExceededError` → requeue 5min, phase = Pending
  - Add structured logging:
    ```go
    log.Info("placement decision made",
        "workload", hybridWorkload.Name,
        "location", decision.Location,
        "cost", decision.EstimatedCost)
    ```
- [ ] E2E test (manual):
  - Create HybridWorkload CR with `minCost: true`
  - Verify `status.placementDecision == "proxmox"`
  - Create HybridWorkload CR with `minCost: false`
  - Verify `status.placementDecision == "aws"`
- [ ] **English practice (2h)**: Write E2E test results in `docs/week1-test-results.md` (broken English → self-review)
- [ ] Commit: "feat: integrate decision engine into controller reconciliation"

**Deliverable**: Controller makes placement decisions, updates CR status, E2E tests pass.

### Day 12 (Sat, Mar 21) - Code Review + Refactoring
**Weekend (Flexible)**
- [ ] Self-review Week 1 code against go-expert checklist:
  - [ ] All errors wrapped with `fmt.Errorf("context: %w", err)`
  - [ ] No naked returns
  - [ ] All exported functions have godoc comments
  - [ ] All providers follow pure constructor pattern (no side effects)
  - [ ] No `panic()` in production code (only in init() validation)
- [ ] Run pre-commit hooks: `pre-commit run --all-files`
- [ ] Fix linter issues: `golangci-lint run ./...`
- [ ] **English practice (2h)**: Start `docs/architecture.md` (system overview, 500 words, broken English)

**Deliverable**: Clean, linted code ready for Week 2.

### Day 13 (Sun, Mar 22) - Buffer + English Writing
**Weekend (Light work)**
- [ ] Continue `docs/architecture.md`:
  - System components diagram (mermaid or ASCII art)
  - HybridWorkload lifecycle
  - Controller reconciliation flow
  - Proxmox + AWS integration points
- [ ] Self-review (5 min), refine with grammar check (optional: use opencode or Grammarly)
- [ ] Optional: Read Viaene Go course Chapter 3-4 (prepare for Week 2 patterns)

**Deliverable**: `docs/architecture.md` draft (500-800 words), ready for Week 2 expansion.

---

## Week 2: Karpenter Integration + Live Testing (March 23-29, 7 days)

**Goal**: Implement Karpenter NodePool manager, test AWS node provisioning via Tailscale VPN, validate hybrid cluster mode.

**Effort**: 6-8 hours/day + 2 hours/day English

### Day 14 (Mon, Mar 23) - Karpenter NodePool CRD Research
**A-Day (Can code)**
- [ ] Research Karpenter v1beta1 NodePool API:
  - Read Karpenter docs: https://karpenter.sh/docs/concepts/nodepools/
  - Identify fields: `spec.template.spec.requirements`, `spec.limits`, `spec.disruption`
  - Note: Karpenter NodeClass for AWS (EC2NodeClass)
- [ ] Design NodePool manager interface:
  - `internal/karpenter/manager.go`:
    - `CreateOrUpdateNodePool(ctx context.Context, workload *v1alpha1.HybridWorkload) error`
    - `DeleteNodePool(ctx context.Context, name string) error`
  - Use controller-runtime dynamic client for CRD manipulation
- [ ] **English practice (2h)**: Document Karpenter integration plan in `docs/karpenter-integration.md` (broken English)
- [ ] Commit: "docs: add Karpenter integration research + design"

**Deliverable**: Karpenter NodePool API understood, manager interface designed.

### Day 15 (Tue, Mar 24) - Karpenter Manager Implementation
**B-Day (Review only)**
- [ ] Implement Karpenter manager:
  - `internal/karpenter/manager.go`:
    - `CreateOrUpdateNodePool()`:
      - Generate NodePool name: `hcro-<workload-name>`
      - Set instance types: t3.micro (Free Tier eligible)
      - Set capacity limits: min=0, max=2 (demo scale)
      - Set subnet tags: `karpenter.sh/discovery=<cluster-name>`
      - Apply NodePool using dynamic client
    - `DeleteNodePool()`:
      - Delete NodePool CR when HybridWorkload is deleted
  - Provider: `internal/di/karpenter.go`
- [ ] Enable leader election in manager options (`LeaderElection: true`, `LeaderElectionID: "hcro.hybrid.io"`)
- [ ] Unit tests: mock dynamic client, verify NodePool spec generation
- [ ] **English practice (2h)**: Add detailed code comments explaining NodePool fields
- [ ] Ask user for review before commit

**Deliverable**: Karpenter manager creates NodePool CRs with correct spec. Leader election enabled for HA.

### Day 16 (Wed, Mar 25) - AWS Credentials + IAM Roles for Karpenter
**A-Day (Can code)**
- [ ] Setup AWS IAM roles for Karpenter (if not already done):
  - KarpenterControllerRole (EC2 permissions: RunInstances, TerminateInstances, DescribeInstances)
  - KarpenterNodeRole (ECR pull, EKS join, CloudWatch logs)
  - Reference: https://karpenter.sh/docs/getting-started/getting-started-with-karpenter/
- [ ] Configure IRSA (IAM Roles for Service Accounts) for hybrid cluster:
  - Create OIDC provider for Proxmox K8s cluster (if not using EKS control plane)
  - Bind KarpenterControllerRole to `karpenter` ServiceAccount
  - Document in `docs/aws-iam-setup.md`
- [ ] Test AWS credentials from Proxmox:
  - `kubectl exec -it <karpenter-pod> -- aws sts get-caller-identity`
  - Should return KarpenterControllerRole ARN
- [ ] **English practice (2h)**: Write IAM setup guide in `docs/aws-iam-setup.md` (broken English → self-review)
- [ ] Commit: "docs: add AWS IAM roles setup for Karpenter"

**Deliverable**: Karpenter has AWS permissions, can provision EC2 instances.

### Day 17 (Thu, Mar 26) - First AWS Node Provisioning Test
**B-Day (Review only)**
- [ ] Integrate Karpenter manager into controller:
  - In `Reconcile()`:
    - If `decision.Location == "aws"` → call `karpenterManager.CreateOrUpdateNodePool()`
    - If `decision.Location == "proxmox"` → call `karpenterManager.DeleteNodePool()` (scale down AWS)
  - Add finalizer handling for NodePool cleanup
- [ ] Manual E2E test:
  - Create HybridWorkload with `minCost: false` (should trigger AWS placement)
  - Watch NodePool creation: `kubectl get nodepool -w`
  - Watch EC2 instance provisioning: `aws ec2 describe-instances --region us-east-1`
  - Verify node joins cluster: `kubectl get nodes` (should see new AWS node via Tailscale)
  - Check Tailscale connectivity: `kubectl exec -it <aws-pod> -- ping <proxmox-service-ip>`
- [ ] **English practice (2h)**: Document test results in `docs/week2-e2e-test.md` (screenshots, broken English)
- [ ] Ask user for review before commit

**Deliverable**: AWS EC2 node provisioned via Karpenter, joins hybrid cluster over Tailscale.

### Day 18 (Fri, Mar 27) - Workload Migration Logic (Pod Scheduling)
**A-Day (Can code)**
- [ ] Implement workload migration controller:
  - Option A: Mutating webhook (sets nodeSelector/nodeAffinity on Pods)
  - Option B: Controller watches Deployment/StatefulSet, patches with nodeSelector
  - **Recommended: Option B** (simpler for demo, no webhook cert management)
  - `internal/controller/workload_migrator.go`:
    - Watch Deployment/StatefulSet with label `hybrid-cloud.dev/managed=true`
    - Read HybridWorkload CR referenced by label `hybrid-cloud.dev/workload=<name>`
    - Patch Deployment with `spec.template.spec.nodeSelector`:
      - If decision.Location == "proxmox" → `location: proxmox`
      - If decision.Location == "aws" → `karpenter.sh/nodepool: hcro-<workload-name>`
- [ ] E2E test:
  - Deploy test Deployment with label `hybrid-cloud.dev/managed=true`
  - Verify pods scheduled on correct location based on HybridWorkload decision
- [ ] **English practice (2h)**: Write migration controller design in `docs/workload-migration.md`
- [ ] Commit: "feat: implement workload migration controller with nodeSelector patching"

**Deliverable**: Pods scheduled to Proxmox or AWS based on HybridWorkload placement decision.

### Day 19 (Sat, Mar 28) - Week 2 Integration Testing
**Weekend (Testing focus)**
- [ ] End-to-end integration test scenario:
  1. Create HybridWorkload `demo-app` with `minCost: true`
  2. Verify decision: `status.placementDecision == "proxmox"`
  3. Deploy Deployment `demo-app` with `hybrid-cloud.dev/workload=demo-app`
  4. Verify pods running on Proxmox nodes
  5. Update HybridWorkload: `minCost: false`
  6. Wait for reconciliation
  7. Verify decision: `status.placementDecision == "aws"`
  8. Verify NodePool created, EC2 provisioned
  9. Verify pods migrated to AWS node
  10. Delete HybridWorkload
  11. Verify NodePool deleted, EC2 terminated
- [ ] Record test run (script output, kubectl commands, timing)
- [ ] **English practice (2h)**: Write test report in `docs/week2-integration-test-report.md` (broken English → self-review → refine)

**Deliverable**: Full integration test passing, documented in English.

### Day 20 (Sun, Mar 29) - Buffer + Video Script Draft
**Weekend (Light work)**
- [ ] Draft demo video script outline:
  - Intro: Problem statement (hybrid cloud cost optimization)
  - Architecture overview (Proxmox + AWS, hybrid K8s cluster)
  - Demo: Create HybridWorkload, show placement decision, workload migration
  - Technical deep dive: Controller code walkthrough, Karpenter integration
  - Outro: Skills demonstrated (K8s operators, Go, cost optimization)
- [ ] Self-review script (English practice, 5-10 min)
- [ ] Optional: Record rough draft video (no editing, just practice speaking English)

**Deliverable**: Demo video script v1 draft, optional rough video recording.

---

## Week 3: Decision Engine Enhancement + AWS API Integration (March 30 - April 5, 7 days)

**Goal**: Enhance decision engine with real resource capacity checks, integrate AWS Pricing API, add cost estimation accuracy.

**Effort**: 6-8 hours/day + 2 hours/day English

### Day 21 (Mon, Mar 30) - Proxmox Capacity Tracking + VPN Health + Hysteresis
**A-Day (Can code)**
- [ ] Enhance Proxmox metrics client:
  - Add `GetNodeCapacity(ctx context.Context, nodeSelector map[string]string) (*Capacity, error)`:
    - Query Proxmox API or Prometheus for node allocatable resources
    - Return: `Capacity{CPU: "8000m", Memory: "16Gi", AvailablePods: 110}`
  - Filter nodes by label selector (e.g., `location=proxmox`)
- [ ] Implement VPN health checker (`internal/healthcheck/vpn_health.go`):
  - TCP dial to VPN endpoint with configurable timeout (5s)
  - Used by DecisionEngine before any AWS placement decision
  - Config: `VPN_ENDPOINT` env var (e.g., `10.0.1.1:51820`)
- [ ] Update decision engine with hysteresis thresholds:
  - Replace single 80% threshold with two:
    - `PROXMOX_SCALE_OUT_THRESHOLD=0.85` (burst to AWS)
    - `PROXMOX_SCALE_BACK_THRESHOLD=0.70` (return to Proxmox)
  - Check `workload.Status.RecommendedPlatform` to determine which threshold applies
  - Check VPN health before AWS placement → `VPNUnhealthyError` if down
  - Check if Proxmox has capacity for `workload.Spec.ResourceRequirements`
  - If insufficient → force AWS placement even if `minCost: true`
- [ ] Unit tests: mock Proxmox client, test capacity overflow, hysteresis flapping, VPN failure
- [ ] **English practice (2h)**: Document capacity checking logic in `docs/capacity-management.md`
- [ ] Commit: "feat: add capacity tracking, hysteresis thresholds, VPN health check"

**Deliverable**: Decision engine respects capacity constraints, prevents flapping, checks VPN health.

### Day 22 (Tue, Mar 31) - AWS Pricing API Integration
**B-Day (Review only)**
- [ ] Replace hardcoded pricing with AWS Pricing API:
  - `internal/aws/pricing.go`:
    - Use AWS SDK v2 Pricing API: `pricing.GetProducts()`
    - Filter by: `ServiceCode=AmazonEC2`, `Region=us-east-1`, `InstanceType=t3.micro`
    - Parse response JSON, extract `pricePerUnit`
    - Add caching: cache prices for 24h (TTL) to reduce API calls
  - Error handling: fallback to hardcoded pricing if API fails
- [ ] Unit tests: mock Pricing API client, test response parsing
- [ ] **English practice (2h)**: Add comments explaining Pricing API response structure
- [ ] Ask user for review before commit

**Deliverable**: Real-time AWS EC2 pricing fetched from Pricing API with caching.

### Day 23 (Wed, Apr 1) - Cost Estimation Accuracy
**A-Day (Can code)**
- [ ] Add cost projection logic:
  - `internal/engine/cost_estimator.go`:
    - `EstimateMonthlyCost(decision *PlacementDecision, resourceReqs v1.ResourceRequirements) (float64, error)`
    - Formula: `hourlyCost * 730 hours/month * (requested CPU / instance vCPU count)`
    - Example: t3.micro (2 vCPUs) costs $0.0104/hour
      - If workload requests 500m CPU → uses 25% of instance
      - But Karpenter provisions full instance → cost = $0.0104 * 730 = $7.59/month
    - Add to `status.EstimatedMonthlyCost` field (new field in CRD)
- [ ] Update HybridWorkload CRD:
  - Add `status.estimatedMonthlyCost` field (float64)
  - Run `make manifests`
- [ ] **English practice (2h)**: Write cost estimation algorithm explanation in `docs/cost-estimation.md`
- [ ] Commit: "feat: add monthly cost estimation to placement decisions"

**Deliverable**: HybridWorkload status shows estimated monthly cost for placement.

### Day 24 (Thu, Apr 2) - Latency Preference Logic
**B-Day (Review only)**
- [ ] Add latency-aware placement:
  - `internal/engine/latency.go`:
    - Hardcoded latency values for demo:
      - Proxmox → Client: 5ms (local network)
      - AWS us-east-1 → Client: 50ms (internet)
    - Decision logic:
      - If `workload.Spec.MinLatency == true` → prefer Proxmox
      - If `workload.Spec.MinLatency == false` → allow AWS
      - If `minCost: true` AND `minLatency: true` → conflict, fail with clear error
  - Update decision engine to consider latency preference
- [ ] Unit tests: test all preference combinations
- [ ] **English practice (2h)**: Document decision matrix in `docs/decision-matrix.md`:
  ```
  | minCost | minLatency | Available Capacity | Decision     |
  |---------|------------|-------------------|--------------|
  | true    | false      | Proxmox: Yes      | Proxmox      |
  | true    | false      | Proxmox: No       | AWS (forced) |
  | false   | true       | N/A               | Proxmox      |
  | true    | true       | N/A               | Error        |
  ```
- [ ] Ask user for review before commit

**Deliverable**: Decision engine supports latency preference, handles conflicts gracefully.

### Day 25 (Fri, Apr 3) - Metrics + Observability
**A-Day (Can code)**
- [ ] Add Prometheus metrics to controller:
  - Use `controller-runtime/pkg/metrics`:
    - `hybridworkload_placement_total{location="proxmox"|"aws"}` (Counter)
    - `hybridworkload_decision_duration_seconds` (Histogram)
    - `hybridworkload_estimated_cost_dollars{location="proxmox"|"aws"}` (Gauge)
  - Instrument decision engine and reconciliation loop
- [ ] Add structured logging:
  - Replace `log.Info()` with structured fields:
    ```go
    log.Info("placement decision",
        "workload", workload.Name,
        "location", decision.Location,
        "cost", decision.EstimatedCost,
        "reason", decision.Reason,
        "duration_ms", duration.Milliseconds())
    ```
- [ ] Create Grafana dashboard JSON (`config/grafana/hcro-dashboard.json`):
  - Panel 1: Placement decisions over time (bar chart)
  - Panel 2: Estimated cost by location (gauge)
  - Panel 3: Decision latency (histogram)
- [ ] **English practice (2h)**: Document metrics in `docs/observability.md`
- [ ] Commit: "feat: add Prometheus metrics and structured logging"

**Deliverable**: Controller exposes Prometheus metrics, rich structured logs.

### Day 26 (Sat, Apr 4) - Week 3 Testing + Refinement
**Weekend (Testing focus)**
- [ ] Integration test with real workloads:
  - Deploy 3 HybridWorkloads with different preferences:
    1. `minCost: true` → expect Proxmox
    2. `minCost: false` → expect AWS
    3. `minLatency: true` → expect Proxmox
  - Test dry-run mode: apply workload with annotation `hcro.io/dry-run: "true"`, verify decision in status but no NodePool created
  - Test hysteresis: verify workload doesn't flap when utilization oscillates near threshold
  - Verify placement decisions, cost estimates, Prometheus metrics
- [ ] Load test (optional):
  - Create 10 HybridWorkloads simultaneously
  - Measure reconciliation time, decision latency
  - Verify no race conditions (Go race detector: `go test -race ./...`)
- [ ] **English practice (2h)**: Write Week 3 test report in `docs/week3-test-results.md`

**Deliverable**: All Week 3 features tested, documented.

### Day 27 (Sun, Apr 5) - Webhook Validation + Buffer + Prep for Week 4
**Weekend (Light work + webhook)**
- [ ] Implement validating webhook for HybridWorkload CRD (`api/v1alpha1/hybridworkload_webhook.go`):
  - Validate `priority` ∈ {"low", "medium", "high"}
  - Validate `maxMonthlyCostUSD` ≥ 0
  - Validate `resources.requests.cpu` and `memory` > 0
  - Validate `capacityType` ∈ {"spot", "on-demand"}
  - Register webhook in `main.go`
- [ ] Unit tests for webhook validation (invalid specs rejected)
- [ ] Commit: "feat: add validating webhook for HybridWorkload CRD"
- [ ] Review Week 4 plan (WireGuard VPN + OpenSpec docs)
- [ ] Read WireGuard documentation: https://www.wireguard.com/quickstart/
- [ ] Research OpenSpec format (if not familiar): https://github.com/OAI/OpenAPI-Specification
- [ ] Draft outline for Week 4 OpenSpec document (1000-1500 words)

**Deliverable**: Webhook validation implemented, mental prep for Week 4.

---

## Week 4: WireGuard VPN + OpenSpec Documentation (April 6-12, 7 days)

**Goal**: Deploy WireGuard VPN on AWS EC2 t2.micro (Free Tier), document architecture in OpenSpec format (English practice), refine demo.

**Effort**: 4 hours/day coding + 2 hours/day English writing (Week 4 strategy from English-Integration-Plan)

**English Integration**: Write component spec in broken English → 5-minute self-review → refine → publish.

### Day 28 (Mon, Apr 6) - WireGuard Terraform Module Design
**A-Day (Can code)**
- [ ] Create Terraform module for WireGuard VPN:
  - Directory: `terraform/wireguard/`
  - Files:
    - `main.tf`: EC2 instance (t2.micro, Free Tier eligible)
    - `variables.tf`: VPC ID, subnet ID, allowed CIDR (Proxmox network)
    - `outputs.tf`: WireGuard public IP, config file path
    - `user_data.sh`: WireGuard installation script
  - EC2 config:
    - AMI: Ubuntu 22.04 LTS (Free Tier eligible)
    - Instance type: `t2.micro` (1 vCPU, 1 GB RAM)
    - Security group: Allow UDP 51820 (WireGuard), SSH 22 (admin)
    - Tags: `Name=hcro-wireguard-vpn`, `Project=hybrid-cloud-optimizer`
- [ ] WireGuard user_data script:
  ```bash
  #!/bin/bash
  apt update && apt install -y wireguard
  wg genkey | tee /etc/wireguard/privatekey | wg pubkey > /etc/wireguard/publickey
  # Generate wg0.conf with Proxmox peer config
  systemctl enable wg-quick@wg0
  systemctl start wg-quick@wg0
  ```
- [ ] **English practice (2h)**: Document Terraform module in `terraform/wireguard/README.md` (broken English)
- [ ] Commit: "feat: add WireGuard VPN Terraform module (EC2 t2.micro)"

**Deliverable**: Terraform module ready to deploy WireGuard VPN on AWS.

### Day 29 (Tue, Apr 7) - WireGuard Deployment
**B-Day (Review only)**
- [ ] Deploy WireGuard VPN:
  ```bash
  cd terraform/wireguard
  terraform init
  terraform plan -out=plan.tfplan
  terraform apply plan.tfplan
  ```
- [ ] Configure Proxmox side:
  - Install WireGuard on Proxmox control plane: `apt install wireguard`
  - Generate keys: `wg genkey | tee privatekey | wg pubkey > publickey`
  - Create `/etc/wireguard/wg0.conf`:
    ```ini
    [Interface]
    PrivateKey = <proxmox-private-key>
    Address = 10.100.0.1/24
    ListenPort = 51820

    [Peer]
    PublicKey = <aws-ec2-public-key>
    Endpoint = <ec2-public-ip>:51820
    AllowedIPs = 10.100.0.0/24
    PersistentKeepalive = 25
    ```
  - Start: `wg-quick up wg0`
- [ ] Test connectivity:
  - Ping from Proxmox to AWS EC2: `ping 10.100.0.2`
  - Ping from AWS EC2 to Proxmox: `ping 10.100.0.1`
- [ ] Update Cilium CNI to route pod traffic through WireGuard (if needed):
  - Option A: Use Cilium native WireGuard encryption (if supported in 1.33)
  - Option B: Add static routes on nodes to use WireGuard tunnel for cross-location traffic
- [ ] **English practice (2h)**: Document deployment steps in `docs/wireguard-setup.md` (broken English → self-review)
- [ ] Ask user for review before commit

**Deliverable**: WireGuard VPN live, Proxmox ↔ AWS EC2 connectivity verified.

### Day 30 (Wed, Apr 8) - WireGuard Cost Monitoring
**A-Day (Can code)**
- [ ] Add cost tracking for WireGuard VPN:
  - Create CloudWatch dashboard:
    - NetworkIn/NetworkOut metrics for EC2 instance
    - Estimated data transfer cost (first 100 GB free, then $0.09/GB)
  - Add cost alert (optional):
    - SNS topic for cost > $5/month (Free Tier exceeded warning)
- [ ] Document Free Tier limits:
  - Update `docs/aws-free-tier.md`:
    - EC2 t2.micro: 750 hours/month (covers 24/7 for 1 instance)
    - Data transfer: 100 GB/month outbound to internet (FREE)
    - Estimated HCRO usage: 4-5 GB/month (well under limit)
    - Year 2+ cost: $7.49/month (t2.micro) + $0/month (data transfer under 100 GB)
- [ ] **English practice (2h)**: Write cost analysis in `docs/vpn-cost-comparison.md`:
  - Compare Tailscale ($0), WireGuard t2.micro ($0 year 1, $7.49 year 2), AWS Site-to-Site VPN ($63/month)
  - Add table with pros/cons
  - Conclusion: WireGuard recommended for portfolio (shows IaC skills, still cost-effective)

**Deliverable**: WireGuard cost monitoring setup, cost comparison documented.

### Day 31 (Thu, Apr 9) - OpenSpec Component Documentation (Part 1)
**English Focus Day (2h coding, 4h writing)**
- [ ] Write OpenSpec-style component spec (1000-1500 words):
  - Document: `docs/openspec-decision-engine.md`
  - Sections (use OpenAPI/OpenSpec format loosely):
    1. **Overview**: Purpose, responsibilities
    2. **API/Interface**:
       ```go
       type DecisionEngine interface {
           Decide(ctx context.Context, workload *HybridWorkload) (*PlacementDecision, error)
       }
       ```
    3. **Input Schema**: HybridWorkload CRD spec (YAML example)
    4. **Output Schema**: PlacementDecision struct (JSON example)
    5. **Decision Algorithm**: Flowchart (mermaid) + step-by-step logic
    6. **Error Handling**: Error cases (capacity exceeded, conflicting preferences)
    7. **Dependencies**: ProxmoxClient, AWSPricingClient
    8. **Metrics**: Prometheus metrics exposed
  - **Write in broken English first** (no AI assistance)
  - **Self-review for 5 minutes** (grammar, clarity)
  - **Refine with opencode** (optional, use sparingly)
- [ ] Commit: "docs: add OpenSpec component spec for Decision Engine (English practice)"

**Deliverable**: Decision Engine component spec (1000-1500 words, English practice completed).

### Day 32 (Fri, Apr 10) - OpenSpec Component Documentation (Part 2)
**English Focus Day (2h coding, 4h writing)**
- [ ] Write second component spec:
  - Document: `docs/openspec-karpenter-manager.md`
  - Same structure as Day 31 (Overview, API, Input/Output, Algorithm, Errors, Dependencies, Metrics)
  - Focus on Karpenter NodePool creation/deletion logic
  - Include example NodePool YAML
  - Add sequence diagram (mermaid):
    ```
    Controller -> KarpenterManager: CreateOrUpdateNodePool(workload)
    KarpenterManager -> K8s API: Apply NodePool CR
    K8s API -> Karpenter Controller: Watch NodePool
    Karpenter Controller -> AWS EC2: RunInstances(t3.micro)
    AWS EC2 -> Karpenter Controller: Instance Running
    Karpenter Controller -> K8s API: Register Node
    ```
  - **Write in broken English → self-review → refine**
- [ ] Commit: "docs: add OpenSpec component spec for Karpenter Manager (English practice)"

**Deliverable**: Karpenter Manager component spec (1000-1500 words, English practice).

### Day 33 (Sat, Apr 11) - Week 4 Integration Testing
**Weekend (Testing focus)**
- [ ] End-to-end test with WireGuard VPN:
  - Switch from Tailscale to WireGuard (update node configs)
  - Create HybridWorkload, verify AWS node provisioning
  - Verify pod-to-pod connectivity across Proxmox ↔ AWS via WireGuard
  - Test workload migration: Proxmox → AWS → Proxmox
  - Measure VPN data transfer: check CloudWatch NetworkOut metric
- [ ] Document test results in `docs/week4-wireguard-test.md`
- [ ] **English practice (2h)**: Write test report in broken English → self-review

**Deliverable**: WireGuard VPN tested in hybrid cluster, documented.

### Day 34 (Sun, Apr 12) - kubectl-hcro Plugin + Demo Video Script Refinement
**Weekend (Light work + kubectl plugin)**
- [ ] Build `kubectl-hcro` cost savings plugin (`cmd/kubectl-hcro/main.go`):
  - `kubectl hcro savings` — aggregate cost across all HybridWorkloads
  - Show: workloads on Proxmox (savings vs AWS equivalent), AWS spend, net savings
  - Install: `go build -o kubectl-hcro ./cmd/kubectl-hcro && sudo mv kubectl-hcro /usr/local/bin/`
- [ ] Commit: "feat: add kubectl-hcro savings plugin for cost reporting"
- [ ] Refine demo video script (from Day 20):
  - Add WireGuard VPN section (show Terraform apply, connectivity test)
  - Add cost comparison segment (Tailscale vs WireGuard)
  - Add `kubectl hcro savings` output to demo
  - Add English documentation showcase (OpenSpec component specs)
- [ ] Practice demo run (no recording, just rehearsal)
- [ ] Self-review script in English (5-10 min)

**Deliverable**: kubectl-hcro savings plugin working. Demo video script v2 ready for Week 5 recording.

---

## Week 5: E2E Testing + Demo Video + Interview Prep (April 13-19, 7 days)

**Goal**: Final testing, record demo video, create INTERVIEW.md with DevSecOps talking points, polish for job applications.

**Effort**: 4-6 hours/day (wind down, avoid burnout before job search)

### Day 35 (Mon, Apr 13) - E2E Test Suite
**A-Day (Can code)**
- [ ] Create E2E test suite:
  - Script: `test/e2e/hybrid_placement_test.sh`
  - Test cases:
    1. Deploy HybridWorkload with `minCost: true` → verify Proxmox placement
    2. Deploy HybridWorkload with `minCost: false` → verify AWS placement + EC2 provisioned
    3. Exceed Proxmox capacity → verify forced AWS placement even with `minCost: true`
    4. Update HybridWorkload preference → verify workload migration
    5. Delete HybridWorkload → verify NodePool cleanup, EC2 termination
  - Assertions: check `kubectl get` outputs, verify status fields
  - Screenshot each step for documentation
- [ ] Run E2E suite, document results in `docs/e2e-test-report.md`
- [ ] **English practice (2h)**: Write E2E test report in broken English → self-review
- [ ] Commit: "test: add E2E test suite for hybrid placement scenarios"

**Deliverable**: Automated E2E test suite passing, documented.

### Day 36 (Tue, Apr 14) - Demo Video Recording (Part 1: Architecture)
**B-Day (Review only, then record)**
- [ ] Record demo video Part 1 (15-20 minutes):
  - Segment 1: Problem statement (3 min)
    - Show slide: hybrid cloud cost optimization challenge
    - Explain Proxmox (cheap, on-premise) vs AWS (expensive, scalable)
  - Segment 2: Architecture overview (5 min)
    - Show architecture diagram (from `docs/architecture.md`)
    - Explain hybrid K8s cluster (Proxmox control plane + AWS workers)
    - Explain VPN options (Tailscale for dev, WireGuard for prod)
  - Segment 3: Components walkthrough (7 min)
    - Show CRD definition (HybridWorkload YAML)
    - Show controller code (reconciliation loop, decision engine)
    - Show Karpenter manager code (NodePool creation)
  - Segment 4: English documentation showcase (3 min)
    - Show OpenSpec component specs (Decision Engine, Karpenter Manager)
    - Explain English learning integration (broken English → self-review → refine)
- [ ] Review recording, retake if needed (don't perfectionist, 1-2 takes max)
- [ ] **English practice (2h)**: Write video transcript in `docs/demo-video-transcript.md`

**Deliverable**: Demo video Part 1 recorded (architecture + code walkthrough).

### Day 37 (Wed, Apr 15) - Demo Video Recording (Part 2: Live Demo)
**A-Day (Can code/record)**
- [ ] Record demo video Part 2 (15-20 minutes):
  - Segment 1: Initial state (2 min)
    - Show Proxmox K8s cluster: `kubectl get nodes` (1 CP + 2 workers)
    - Show AWS console: no EC2 instances running
  - Segment 2: Create HybridWorkload (minCost: true) (5 min)
    - Apply YAML: `kubectl apply -f examples/demo-workload-cost.yaml`
    - Show controller logs: decision made (Proxmox)
    - Show status: `kubectl get hybridworkload demo-workload-cost -o yaml`
    - Deploy app: `kubectl apply -f examples/demo-app-deployment.yaml`
    - Show pods running on Proxmox: `kubectl get pods -o wide`
  - Segment 3: Update HybridWorkload (minCost: false) (5 min)
    - Patch YAML: `kubectl patch hybridworkload demo-workload-cost -p '{"spec":{"minCost":false}}'`
    - Show controller logs: decision changed (AWS)
    - Show NodePool created: `kubectl get nodepool`
    - Show EC2 provisioned: AWS console, `kubectl get nodes` (new AWS node)
    - Show pods migrating: `kubectl get pods -o wide -w`
  - Segment 4: Cleanup (2 min)
    - Delete HybridWorkload: `kubectl delete hybridworkload demo-workload-cost`
    - Show NodePool deleted, EC2 terminated
  - Segment 5: Cost summary (3 min)
    - Show Prometheus metrics (placement count, estimated cost)
    - Show CloudWatch data transfer (WireGuard VPN usage: ~5 GB/month)
    - Show cost comparison table (Proxmox $0, AWS $7.59/month for t3.micro)
- [ ] Review recording, retake if needed
- [ ] **English practice (2h)**: Update video transcript with Part 2 content

**Deliverable**: Demo video Part 2 recorded (live demo + cost analysis).

### Day 38 (Thu, Apr 16) - INTERVIEW.md Creation
**B-Day (Review only)**
- [ ] Create `INTERVIEW.md` with DevSecOps talking points:
  - **Section 1: Project Overview**
    - Elevator pitch (30 seconds): "Hybrid cloud resource optimizer that reduces costs by intelligently placing workloads between on-premise Proxmox and AWS based on cost and latency constraints"
    - Key metrics: Reduced AWS costs by X% in demo scenario
  - **Section 2: Technical Skills Demonstrated**
    - Kubernetes: Custom controllers, CRDs, Karpenter, hybrid cluster
    - Go: Production patterns (DI with samber/do, error handling, concurrency safety)
    - IaC: Terraform (WireGuard VPN), Ansible (Proxmox K8s management)
    - Cost Optimization: AWS Free Tier usage, pricing API integration, VPN traffic minimization
    - Security: IAM roles, IRSA, no hardcoded credentials
    - Observability: Prometheus metrics, structured logging
  - **Section 3: Architecture Decisions**
    - Q: Why hybrid cluster instead of two separate clusters?
      - A: Simpler management (single control plane), unified scheduling, lower overhead
    - Q: Why Tailscale AND WireGuard?
      - A: Tailscale for rapid dev (5-min setup), WireGuard for prod (shows Terraform/IaC skills, cost-effective)
    - Q: Why Karpenter instead of Cluster Autoscaler?
      - A: Faster provisioning, better bin-packing, NodePool flexibility
    - Q: How to avoid Free Tier overage?
      - A: CloudWatch monitoring, cost alerts, VPN traffic optimization (4-5 GB/month well under 100 GB limit)
  - **Section 4: Trade-offs & Production Considerations**
    - Proxmox single point of failure (control plane) → mitigation: HA control plane (3 nodes)
    - VPN bandwidth bottleneck → mitigation: monitor latency, upgrade to t3.small if needed
    - Karpenter cold start time (EC2 provisioning ~2 min) → mitigation: keep warm pool (min=1)
  - **Section 5: Demo Talking Points**
    - Show live infrastructure OR video recording
    - Walk through code: controller reconciliation, decision engine, Karpenter manager
    - Explain English documentation (OpenSpec specs written in broken English, refined through self-review)
  - **Section 6: Future Enhancements** (if asked "what would you add?")
    - Multi-region AWS support (us-west-2, eu-central-1)
    - Spot instance integration for further cost savings
    - Real-time cost tracking (CloudWatch billing API)
    - GitOps with Flux/ArgoCD for declarative management
- [ ] **English practice (2h)**: Self-review INTERVIEW.md, refine wording
- [ ] Ask user for review before commit

**Deliverable**: INTERVIEW.md ready with all DevSecOps talking points.

### Day 39 (Fri, Apr 17) - Video Editing + Publishing
**A-Day (Can edit)**
- [ ] Edit demo video:
  - Combine Part 1 + Part 2 (30-40 minutes total)
  - Add intro/outro slides
  - Add captions/subtitles (optional, helps with English clarity)
  - Add timestamps in description:
    - 0:00 - Problem Statement
    - 3:00 - Architecture Overview
    - 8:00 - Code Walkthrough
    - 15:00 - Live Demo (Proxmox → AWS Migration)
    - 30:00 - Cost Analysis
    - 35:00 - Conclusion
- [ ] Upload to YouTube (unlisted link for portfolio):
  - Title: "Hybrid Cloud Resource Optimizer - K8s Operator Demo (Proxmox + AWS)"
  - Description: Link to GitHub repo, INTERVIEW.md, tech stack
  - Tags: kubernetes, golang, devops, cost-optimization, hybrid-cloud
- [ ] Update README.md:
  - Add "Demo Video" section with YouTube embed
  - Add "Documentation" section (link to docs/*.md)
  - Add "Interview Guide" section (link to INTERVIEW.md)
- [ ] **English practice (2h)**: Write README.md in broken English → self-review → refine
- [ ] Commit: "docs: add demo video, update README with portfolio links"

**Deliverable**: Demo video published, README polished for GitHub portfolio.

### Day 40 (Sat, Apr 18) - Final Polish + GitHub Portfolio
**Weekend (Final touches)**
- [ ] GitHub repository polish:
  - Add screenshots to README (architecture diagram, HybridWorkload CRD, demo screenshots)
  - Add badges: Go version, license (MIT), build status (if CI setup)
  - Add LICENSE file (MIT recommended for portfolio projects)
  - Pin repository on GitHub profile (top 6 repos)
- [ ] LinkedIn post:
  - "Excited to share my latest project: Hybrid Cloud Resource Optimizer! 🚀"
  - "Built a K8s operator in Go that optimizes workload placement between on-premise Proxmox and AWS based on cost/latency constraints."
  - "Tech stack: K8s, Go, Karpenter, Terraform, WireGuard VPN, AWS Free Tier"
  - "Check out the demo video and code: [GitHub link]"
  - Tags: #Kubernetes #Golang #DevOps #CostOptimization #HybridCloud
- [ ] Resume update:
  - Add "Hybrid Cloud Resource Optimizer" to Projects section
  - Bullet points:
    - "Developed K8s operator in Go to optimize workload placement across hybrid cloud (Proxmox + AWS)"
    - "Integrated Karpenter for dynamic node provisioning, AWS Pricing API for cost estimation"
    - "Deployed WireGuard VPN on AWS EC2 t2.micro (Free Tier) using Terraform"
    - "Reduced infrastructure costs by X% through intelligent placement decisions"
  - Link to GitHub repo + demo video
- [ ] **English practice (2h)**: Self-review LinkedIn post and resume updates

**Deliverable**: GitHub portfolio ready, LinkedIn post published, resume updated.

### Day 41 (Sun, Apr 19) - Rest & Reflection
**Weekend (Recovery)**
- [ ] No coding! Rest day before job search begins
- [ ] Optional: Review Viaene Go course chapters completed so far
- [ ] Optional: Reflect on English progress (compare Week 1 vs Week 5 writing quality)
- [ ] Optional: Plan job application strategy (which companies, when to apply after residence permit)

**Deliverable**: Mental rest, ready for job search phase.

---

## Post-Roadmap: Job Application Phase (April 20 - June 2026)

**Timeline**: Residence permit expected ~June 2026, job applications start immediately after.

**Strategy**: Use HCRO project as portfolio centerpiece during interviews.

### Pre-Interview Checklist
- [ ] Review INTERVIEW.md before each interview
- [ ] Practice demo walkthrough (5-min version for phone screens, 15-min version for technical rounds)
- [ ] Prepare to run live demo (ensure Proxmox + AWS infrastructure ready to spin up)
- [ ] Practice English speaking (record yourself explaining architecture, listen for clarity)

### Interview Scenarios

**Phone Screen (30 min)**:
- Problem: "Tell me about a recent project"
- Response: Hybrid Cloud Resource Optimizer elevator pitch (from INTERVIEW.md Section 1)
- Follow-up: "What was the biggest challenge?"
- Response: WireGuard VPN + Cilium CNI integration, Free Tier cost optimization

**Technical Round (60 min)**:
- Task: "Show me your code"
- Action: Screen share, walk through controller code (reconciliation loop, decision engine)
- Task: "How would you handle failure scenario X?"
- Action: Reference error handling patterns in code, explain retry logic

**System Design Round (60 min)**:
- Task: "Design a multi-region cost optimizer"
- Action: Draw architecture on whiteboard, reference HCRO as starting point, explain trade-offs

### Target Companies (DevSecOps roles)
- AWS-native companies (show AWS expertise via HCRO)
- K8s-focused companies (show operator development skills)
- Cost-conscious startups (show cost optimization mindset)
- Hybrid cloud enterprises (show hybrid architecture understanding)

---

## Summary: Key Milestones

| Week | Dates         | Focus                          | Deliverables                                      | English Integration                     |
|------|---------------|--------------------------------|---------------------------------------------------|-----------------------------------------|
| 0    | Mar 10-15     | Setup + Recovery               | Project scaffold, Tailscale VPN, CRD design       | Commit messages in English              |
| 1    | Mar 16-22     | Core Controller                | Reconciliation loop, decision engine, E2E test    | `docs/architecture.md` (500 words)      |
| 2    | Mar 23-29     | Karpenter Integration          | NodePool manager, AWS node provisioning           | Test reports in English                 |
| 3    | Mar 30-Apr 5  | Decision Enhancement           | Capacity checks, Pricing API, metrics             | `docs/capacity-management.md`           |
| 4    | Apr 6-12      | WireGuard + Docs               | WireGuard VPN, OpenSpec component specs (2x1000w) | OpenSpec docs (English practice focus)  |
| 5    | Apr 13-19     | Testing + Demo                 | E2E suite, demo video, INTERVIEW.md, GitHub polish| Video transcript, README refinement     |

**Total Effort**: 40 days, ~6-8 hours/day coding + 2 hours/day English = **320-400 hours**

**Post-April 19**: Job search preparation, wait for residence permit (~June 2026), start interviews.

---

## Risk Mitigation

**Risk 1: Burnout (age 51)**
- Mitigation: Recovery days (Day 1, Day 6, weekends), sustainable 6-8h/day pace (not 12h sprints)

**Risk 2: AWS Free Tier overage**
- Mitigation: CloudWatch cost alerts, monitor VPN traffic weekly, t2.micro = $0 year 1

**Risk 3: English writing quality**
- Mitigation: Self-review protocol (5 min before commit), gradual improvement over 5 weeks, AI-assisted refinement as needed

**Risk 4: Proxmox infrastructure issues**
- Mitigation: User already has working K8s cluster, separate Terraform/Ansible management project

**Risk 5: Job search timeline pressure**
- Mitigation: Residence permit expected June 2026, 6 weeks buffer after HCRO completion (April 19 → June 1), can start applying immediately after permit

---

## Next Steps (March 10, 2026)

1. **Today (after SAA exam)**: Rest, celebrate 🎉
2. **Tomorrow (March 11)**: Start Week 0 Day 2 checklist (environment validation)
3. **March 16**: Begin full-focus mode (6-8h/day after SAA recovery)
4. **Daily routine**:
   - Morning: Review day's checklist (from this ROADMAP.md)
   - Coding: 6-8 hours (use go-expert skill, refer to MVP.md for component details)
   - English practice: 2 hours (write docs in broken English, self-review 5 min, commit)
   - Evening: Update todos, commit work, plan next day

**Files to reference daily**:
- `/home/abevz/github/hybrid-cloud-optimizer/ROADMAP.md` (this file)
- `/home/abevz/github/hybrid-cloud-optimizer/MVP.md` (technical spec)
- `/home/abevz/Obsidian/moonbase/BASE/English-Integration-Plan-2026.md` (English sprint context)

Прорвемся! 💪 Let's build this! 🚀
