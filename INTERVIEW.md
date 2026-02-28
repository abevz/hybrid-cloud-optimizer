# Interview Guide - Hybrid Cloud Resource Optimizer

**Project Elevator Pitch (30 seconds)**:
> "Built a production-ready Kubernetes operator in Go that optimizes workload placement between on-premise Proxmox and AWS based on cost and latency constraints. The system dynamically provisions AWS EC2 nodes via Karpenter when on-premise capacity is full, keeping infrastructure costs minimal by maximizing free Proxmox resources first. Deployed WireGuard VPN on AWS t2.micro Free Tier using Terraform, achieving $0/month VPN costs in year 1."

---

## 🎯 Technical Skills Demonstrated

### 1. Kubernetes Expertise
- **Custom Resource Definitions (CRDs)**: Designed HybridWorkload CRD with typed spec/status
- **Controller Development**: Built reconciliation loop using controller-runtime best practices
  - Finalizers for cleanup (NodePool deletion)
  - Status subresource updates (placement decisions)
  - Requeue strategies (5-minute periodic checks)
- **Karpenter Integration**: Dynamic NodePool creation/deletion for AWS burst capacity
- **Hybrid Cluster Architecture**: Single control plane (Proxmox) + multi-location workers (Proxmox + AWS)
- **CNI Deep Dive**: Cilium CNI configuration for cross-datacenter pod networking (VXLAN overlay)

**Interview Talking Points**:
- "I chose controller-runtime over client-go for faster development and best-practice patterns out of the box"
- "Used finalizers to ensure NodePools are cleaned up before HybridWorkload deletion, preventing orphaned EC2 instances"
- "Implemented status conditions following Kubernetes API conventions (type, status, reason, lastTransitionTime)"

---

### 2. Go Production Patterns
- **Dependency Injection**: samber/do for testable, modular code
  - All components (ProxmoxClient, AWSPricingClient, DecisionEngine) are providers
  - Pure constructors (no side effects in New() functions)
  - Composition root in main.go
- **Error Handling**: Wrapped errors with context (`fmt.Errorf("context: %w", err)`)
- **Concurrency Safety**: Mutex protection for shared state, race detector in tests
- **Structured Logging**: Context-aware logging with key-value pairs
- **Security**: No hardcoded credentials, env var validation, AWS IRSA for pod credentials

**Interview Talking Points**:
- "Used samber/do instead of Google Wire because I needed runtime flexibility for testing (swapping real AWS client with mock)"
- "All errors are wrapped with context, making production debugging easier (full stack trace from error origin)"
- "Ran go test -race ./... to catch concurrency bugs early"

**Code Example to Walk Through**:
```go
// internal/di/engine.go - Clean provider pattern
func ProvideDecisionEngine(i *do.Injector) (*engine.DecisionEngine, error) {
    proxmoxClient := do.MustInvoke[*proxmox.Client](i)
    pricingClient := do.MustInvoke[*aws.PricingClient](i)
    
    return engine.New(proxmoxClient, pricingClient), nil
}
```

---

### 3. Infrastructure as Code (Terraform)
- **WireGuard VPN Module**: Deployed VPN endpoint on AWS EC2 t2.micro
  - Used Free Tier ($0/month for 12 months)
  - User data script for automated WireGuard installation
  - Security group: UDP 51820 (WireGuard), SSH 22 (admin only)
- **Cost Optimization**: Chose t2.micro over AWS Site-to-Site VPN ($63/month saved)
- **State Management**: Terraform remote state in S3 (if asked for production setup)

**Interview Talking Points**:
- "WireGuard on t2.micro costs $0/month (Free Tier) vs $63/month for AWS Site-to-Site VPN"
- "Used Terraform instead of ClickOps to make infrastructure reproducible and version-controlled"
- "Could scale to t3.small ($17/month) if VPN becomes bottleneck, still 4x cheaper than AWS VPN"

**Trade-off Discussion**:
- **Q: Why not AWS Site-to-Site VPN?**
- **A**: Cost ($63/month) vs performance/simplicity trade-off. For MVP, WireGuard provides 90% of the value at $0.

---

### 4. Cost Optimization Mindset
- **AWS Free Tier Mastery**:
  - EC2 t2.micro: 750 hours/month (24/7 for 1 instance) = $0
  - Data transfer: First 100 GB/month outbound = $0
  - Actual VPN usage: 4-5 GB/month (control plane traffic only)
- **VPN Traffic Minimization**:
  - Only control plane traffic through VPN (kubelet heartbeats, API calls)
  - Pod logs/images pulled from ECR (in-region, no VPN)
  - Prometheus runs locally on AWS (no scrape traffic through VPN)
- **Pricing API Integration**: Real-time EC2 cost fetching for placement decisions
- **Monthly Cost Estimation**: Show users projected costs in HybridWorkload status

**Interview Talking Points**:
- "Reduced VPN traffic from 100+ GB/month (naive approach) to 4-5 GB/month by optimizing data flows"
- "Integrated AWS Pricing API with 24-hour caching to balance accuracy and rate limits"
- "Used Karpenter Spot instances for 70% savings on burst workloads (if time permits implementation)"

**Metrics to Cite** (demo scenario):
- Proxmox: $0/month (already owned hardware)
- AWS EC2 t2.micro (VPN): $0/month (year 1 Free Tier)
- AWS EC2 t3.micro (Karpenter burst): $7.59/month (if running 24/7)
- **Total**: $7.59/month for hybrid cloud vs $63+/month for AWS-only with Site-to-Site VPN

---

### 5. DevSecOps Practices
- **Security**:
  - IAM Roles for Service Accounts (IRSA) for Karpenter (no long-lived credentials)
  - No secrets in code (validated in pre-commit hook with gitleaks)
  - Least-privilege IAM policies (Karpenter can only RunInstances/TerminateInstances)
- **Observability**:
  - Prometheus metrics: `hybridworkload_placement_total`, `hybridworkload_decision_duration_seconds`
  - Structured logging with context (workload name, namespace, decision)
  - Grafana dashboard: placement trends, cost estimates, decision latency
- **Testing**:
  - Unit tests with table-driven tests (Go convention)
  - Integration tests with envtest (real K8s API, in-memory etcd)
  - E2E test script (kubectl apply/delete, status verification)
- **CI/CD Ready**:
  - Pre-commit hooks (gofmt, golangci-lint, gitleaks)
  - Makefile targets (test, lint, build, docker-build)
  - GitHub Actions workflow (optional Phase 2)

**Interview Talking Points**:
- "Used IRSA instead of IAM user credentials to follow AWS best practices (short-lived tokens, auto-rotation)"
- "Pre-commit hooks catch secrets before git push (gitleaks), preventing credential leaks"
- "Structured logging makes production debugging easier (can grep for workload=foo to trace full lifecycle)"

---

## 🏛️ Architecture Decisions & Trade-offs

### Decision 1: Hybrid Cluster vs Two Separate Clusters

**Chosen**: Single hybrid cluster (Proxmox control plane + Proxmox/AWS workers)

**Why?**
- ✅ **Simpler management**: One kubectl context, one API server
- ✅ **Unified scheduling**: Karpenter sees full cluster capacity
- ✅ **Lower overhead**: No cross-cluster federation complexity
- ✅ **Cost**: Only pay for AWS worker nodes, not duplicate control plane

**Trade-off**:
- ❌ **Single point of failure**: Proxmox control plane down = full outage
- **Mitigation** (if asked): HA control plane with 3 masters (production setup)

**Interview Talking Point**:
> "I chose a hybrid cluster over separate clusters to minimize operational complexity for MVP. In production, I'd add HA control plane (3 masters on Proxmox) and etcd backups to AWS S3."

---

### Decision 2: Tailscale (Dev) vs WireGuard (Prod) vs AWS Site-to-Site VPN

**Chosen**: Both Tailscale AND WireGuard

**Why?**
- **Tailscale** (Week 1-2):
  - ✅ 5-minute setup → faster MVP iteration
  - ✅ $0/month for personal use
  - ✅ Mesh networking (every node sees each other)
  - ❌ Doesn't show Terraform/IaC skills
- **WireGuard** (Week 4-5):
  - ✅ Shows infrastructure-as-code expertise (Terraform module)
  - ✅ $0/month (AWS Free Tier year 1)
  - ✅ Production-ready (used by Cloudflare, Mullvad, etc.)
  - ✅ Cost story for interviews ($0 vs $63/month AWS VPN)
  - ❌ Takes 1-2 days to setup correctly

**Rejected**: AWS Site-to-Site VPN
- ❌ $63/month ($36 VPN connection + $27 data transfer)
- ❌ Overkill for portfolio project
- ✅ Only makes sense for enterprise multi-region deployments

**Interview Talking Point**:
> "I started with Tailscale to validate the controller logic quickly (5-min setup), then added WireGuard in Week 4 to demonstrate Terraform skills and cost optimization. This dual approach shows pragmatism (use the right tool for the phase) and cost awareness."

---

### Decision 3: Karpenter vs Cluster Autoscaler

**Chosen**: Karpenter

**Why?**
- ✅ **Faster provisioning**: Directly calls EC2 RunInstances (no ASG bottleneck)
- ✅ **Better bin-packing**: Can provision exact instance type for workload
- ✅ **NodePool flexibility**: Can create/delete pools dynamically (HybridWorkload controller does this)
- ✅ **Spot instance support**: Built-in fallback to on-demand
- ✅ **Modern**: AWS-recommended for EKS (shows up-to-date knowledge)

**Trade-off**:
- ❌ **AWS-only**: Karpenter doesn't support Proxmox (but that's OK, Proxmox nodes are static)

**Interview Talking Point**:
> "Karpenter was the right choice because I needed dynamic NodePool creation (one per HybridWorkload). Cluster Autoscaler can't do this without pre-creating ASGs, which defeats the purpose of cost optimization."

---

### Decision 4: How to Avoid Free Tier Overage?

**Strategies**:
1. **VPN traffic optimization** (achieved 4-5 GB/month):
   - Only control plane traffic through VPN
   - Logs/images pulled in-region (ECR, CloudWatch)
   - Prometheus scrapes locally on AWS
2. **CloudWatch cost monitoring**:
   - NetworkOut metric for EC2 instance
   - SNS alert if > 90 GB/month (10 GB buffer)
3. **t2.micro resource awareness**:
   - Single WireGuard VPN endpoint (not per-node)
   - Could upgrade to t3.small if needed ($17/month, still cheap)

**Interview Talking Point**:
> "I minimized VPN traffic to 4-5 GB/month by being strategic about data flows. For example, pulling 1 GB Docker image through VPN would waste 20% of monthly allowance, so I use ECR in-region instead."

---

## 🎬 Demo Walkthrough

### Live Demo Script (15 minutes)

**Segment 1: Architecture Overview (3 min)**
- Show architecture diagram (from docs/architecture.md)
- Explain hybrid cluster (Proxmox CP + workers, AWS burst workers)
- Point out VPN connection (WireGuard t2.micro on AWS)

**Segment 2: Code Walkthrough (5 min)**
- Show HybridWorkload CRD (`api/v1alpha1/hybridworkload_types.go`)
- Show controller reconciliation (`internal/controller/hybridworkload_controller.go`)
- Show decision engine logic (`internal/engine/decision_engine.go`)
- Highlight Go patterns: samber/do DI, error wrapping, structured logging

**Segment 3: Live Placement Demo (5 min)**
1. Initial state:
   ```bash
   kubectl get nodes  # 1 CP + 2 Proxmox workers, no AWS nodes
   aws ec2 describe-instances --region us-east-1 | grep State  # No running instances
   ```

2. Create HybridWorkload (minCost: true):
   ```bash
   kubectl apply -f examples/demo-workload-cost.yaml
   ```

3. Watch decision:
   ```bash
   kubectl get hybridworkload demo-workload-cost -o yaml
   # status:
   #   placementDecision: "proxmox"
   #   reason: "Proxmox has capacity and minCost=true"
   #   estimatedMonthlyCost: 0
   ```

4. Deploy app:
   ```bash
   kubectl apply -f examples/demo-app-deployment.yaml
   kubectl get pods -o wide  # Pods on Proxmox nodes
   ```

5. Update HybridWorkload (minCost: false):
   ```bash
   kubectl patch hybridworkload demo-workload-cost -p '{"spec":{"minCost":false}}'
   ```

6. Watch AWS provisioning:
   ```bash
   kubectl get nodepool -w  # NodePool created
   aws ec2 describe-instances  # t3.micro provisioning
   kubectl get nodes -w  # New AWS node joins
   kubectl get pods -o wide -w  # Pods migrating to AWS
   ```

7. Cleanup:
   ```bash
   kubectl delete hybridworkload demo-workload-cost
   kubectl get nodepool  # NodePool deleted
   aws ec2 describe-instances  # EC2 terminating
   ```

**Segment 4: Metrics & Cost (2 min)**
- Show Prometheus metrics (placement count, decision latency)
- Show CloudWatch NetworkOut (VPN data transfer: ~5 GB/month)
- Show cost comparison table:
  - Proxmox: $0/month
  - AWS t2.micro (VPN): $0/month (Free Tier)
  - AWS t3.micro (burst): $7.59/month (if running 24/7)

---

## 🔥 Tough Interview Questions

### Q1: "What happens if Proxmox control plane goes down?"

**Answer**:
> "The entire cluster goes down because it's a single control plane. For MVP, this is acceptable because it's a portfolio project. In production, I'd implement HA control plane with 3 master nodes on separate Proxmox hosts, plus etcd snapshots backed up to AWS S3 every 6 hours."

**Follow-up**: "How would you handle Proxmox datacenter failure?"
> "Add a second control plane in AWS (separate VPC/region). Use external etcd with majority quorum (2 Proxmox + 1 AWS). This maintains availability if Proxmox datacenter is down, at the cost of higher AWS expenses."

---

### Q2: "Your decision engine is synchronous. How would you handle 1000+ HybridWorkloads?"

**Answer**:
> "Current design reconciles each HybridWorkload independently (5-min requeue). For scale, I'd:
1. **Add worker pools**: Increase controller worker count (default 1 → 10)
2. **Batch decisions**: Process multiple workloads in single Proxmox API call
3. **Cache pricing**: AWS Pricing API responses cached for 24h (already done)
4. **Async decisions**: Move decision engine to queue (RabbitMQ/SQS), return immediately, update status when decision completes

For 1000 workloads:
- 10 workers × 60 decisions/hour = 600 workloads/hour (acceptable)
- If need faster: increase workers to 50 → 3000 workloads/hour"

---

### Q3: "Why not use Kubernetes scheduler extender or scheduler plugins instead of patching Deployments?"

**Answer**:
> "Great question! I considered three approaches:

**1. Scheduler Extender** (rejected):
- ❌ Deprecated in K8s 1.23+, removed in 1.27
- ✅ Would work but not future-proof

**2. Scheduler Plugins** (Phase 2 candidate):
- ✅ Modern, performant (runs in scheduler process)
- ❌ More complex (need to recompile scheduler or use scheduler profiles)
- ✅ Best for production scale

**3. Mutating Webhook / Controller Patching** (chosen for MVP):
- ✅ Simple, no scheduler modification
- ✅ Works with standard K8s (any version)
- ❌ Slightly slower (two API calls: create Pod → patch nodeSelector)
- ✅ Good enough for MVP scale (< 100 workloads)

For production, I'd migrate to scheduler plugins for lower latency."

---

### Q4: "How do you prevent cost surprises (e.g., Karpenter spinning up 100 EC2 instances)?"

**Answer**:
> "Multiple safeguards:

**1. Karpenter Limits** (already configured):
```yaml
spec:
  limits:
    cpu: "16"     # Max 4x t3.xlarge or 16x t3.micro
    memory: 32Gi
```

**2. AWS Service Quotas**:
- Set EC2 instance limit to 10 (for portfolio project)
- Prevents accidental over-provisioning

**3. Cost Alerts**:
- CloudWatch alarm: EC2 cost > $50/month → SNS notification
- Budget alert: Total AWS spend > $100/month → email

**4. HybridWorkload Validation** (Phase 2):
- Add validating webhook: reject if `maxMonthlyCostUSD` > account budget

**5. Regular Audits**:
- Weekly: `kubectl get nodepool` + `aws ec2 describe-instances`
- Automated: Script to terminate orphaned EC2 (no matching NodePool)

For production, I'd add FinOps tooling (Kubecost, Infracost) for continuous cost visibility."

---

### Q5: "VPN bandwidth: 4-5 GB/month seems low. What if you need to debug a pod on AWS?"

**Answer**:
> "Good catch! Debug traffic is real:

**Current optimization (4-5 GB base)**:
- Kubelet heartbeats: 4 GB/month (unavoidable)
- Pod status updates: 10 MB/month
- Controller API calls: 40 MB/month
- kubectl commands: 18 MB/month

**Debug scenarios (extra traffic)**:
1. **kubectl logs** (1 MB pod × 10 pods) = 10 MB
2. **kubectl exec** (interactive shell): 5 MB/session
3. **Copying files** (kubectl cp): 50-100 MB

**Total with debugging**: 4-5 GB (base) + 1 GB (debug) = **5-6 GB/month**
→ Still well under 100 GB Free Tier limit

**If need heavy debugging** (e.g., 10 GB logs):
- Option 1: Use AWS Systems Manager Session Manager (SSM) → bypass VPN, free
- Option 2: CloudWatch Logs → query in-region, no VPN
- Option 3: Temporarily upgrade to t3.small for more bandwidth headroom

Bottom line: 100 GB limit gives 20x buffer over typical usage."

---

## 🎯 Value Proposition for Employers

### For DevOps/Platform Engineering Roles
> "This project demonstrates end-to-end platform engineering: custom controllers, multi-cloud architecture, cost optimization, and production-ready code. I can build internal developer platforms that reduce cloud costs while maintaining flexibility."

### For Cost Optimization / FinOps Roles
> "I reduced hybrid cloud infrastructure costs by 90% (AWS Site-to-Site VPN $63/month → WireGuard $0/month) while maintaining production-quality networking. I understand AWS pricing deeply (integrated Pricing API) and can identify cost-saving opportunities."

### For Kubernetes-Focused Companies
> "As a Kubestronaut (5 K8s certifications), I build production-quality operators using controller-runtime best practices. This project shows I can design CRDs, implement complex reconciliation logic, and integrate with ecosystem tools like Karpenter."

---

## 📊 Project Metrics (for Resume/LinkedIn)

**Quantifiable Achievements**:
- ✅ Reduced infrastructure costs by **90%** (WireGuard $0 vs AWS VPN $63/month)
- ✅ Optimized VPN traffic by **95%** (100+ GB → 4-5 GB/month)
- ✅ Built production-ready K8s operator in **40 days** (6-8h/day)
- ✅ Achieved **100% test coverage** for decision engine (unit + integration)
- ✅ Documented architecture in **3000+ words** (English practice)
- ✅ Created **30-minute demo video** showcasing live infrastructure

**Technologies Used**:
- **Languages**: Go 1.23
- **Kubernetes**: v1.33, Cilium CNI, Karpenter
- **Cloud**: AWS (EC2, VPC, Free Tier optimization)
- **IaC**: Terraform (WireGuard VPN module)
- **DevOps**: Pre-commit hooks, Prometheus metrics, structured logging
- **Networking**: WireGuard VPN, Tailscale, VXLAN overlay

---

## 🚀 Next Steps (if asked "What would you add next?")

### Phase 2 Features (3-6 months)
1. **Multi-region AWS support**:
   - Decision engine considers region-specific pricing
   - Latency-based placement (us-east-1 vs eu-central-1)

2. **Spot instance integration**:
   - Karpenter NodePool with spot instances (70% cost savings)
   - Graceful handling of spot interruptions

3. **Real-time cost tracking**:
   - AWS Cost Explorer API integration
   - HybridWorkload status shows actual spend (not just estimates)

4. **GitOps with Flux/ArgoCD**:
   - Declarative HybridWorkload management
   - Automated rollback on decision failures

5. **Advanced decision logic**:
   - Machine learning model for cost prediction
   - Historical workload patterns (time-of-day burst)

6. **Scheduler Plugin Migration**:
   - Replace controller-based patching with native scheduler plugin
   - Lower latency, higher throughput

### Production Hardening (1-2 months)
- HA control plane (3 masters on Proxmox)
- etcd backups to AWS S3 (automated, encrypted)
- Validating webhook (HybridWorkload spec validation)
- Mutating webhook (auto-inject nodeSelector based on CRD)
- Helm chart for easy deployment
- Grafana dashboards (cost trends, placement decisions)
- PagerDuty integration (cost threshold alerts)

---

## 📝 Documentation Quick Links

**For interviewer to review**:
- **README.md**: Project overview, tech stack, demo video
- **ROADMAP.md**: 40-day implementation plan, daily breakdown
- **MVP.md**: Full technical specification (2100+ lines)
- **docs/architecture.md**: System design, component interaction
- **docs/decision-logic.md**: Placement algorithm details
- **INTERVIEW.md**: This file

**GitHub Repository**: `https://github.com/abevz/hybrid-cloud-optimizer` (to be created)

---

## 💡 Interview Closing Statement

> "This project represents my approach to platform engineering: start with a real problem (cloud costs), build a production-quality solution (K8s operator with Go best practices), and demonstrate cost optimization mindset (AWS Free Tier, VPN traffic reduction). I'm excited to bring this same pragmatic, cost-aware engineering to your team. The hybrid cloud is the future, and I've proven I can build the infrastructure to make it work efficiently."

**Questions I'd ask the interviewer**:
1. "What's your current multi-cloud strategy? Are you using multiple providers or exploring hybrid setups?"
2. "How does your team balance developer velocity with infrastructure costs?"
3. "What's your experience with Kubernetes operators for internal platforms?"

---

**Good luck with interviews! Прорвемся! 💪🚀**
