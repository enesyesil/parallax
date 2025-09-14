# parallax
Lightweight LLM Inference Load Balancer


## Motivation

As a final-year Computer Science student interested in infrastructure systems and LLM infrastructure. As part of my course **EECS 4171 (Parallel, Concurrent, and Distributed Algorithms)**, I wanted to connect the algorithms we are learning with how real companies (like OpenAI or Anthropic) run their inference systems.

In production, a big challenge is sending thousands of requests to multiple GPUs or servers, keeping latency low, and making sure resources are used efficiently. At its core, that is a **scheduling and load balancing problem**.

This project, **Parallax**, is my lightweight version of such a system. Instead of GPUs, I simulate inference with CPU-heavy tasks. The focus is on experimenting with different **scheduling algorithms**, adding **observability** (metrics, dashboards), and showing how theory links to real infrastructure.

## High-Level Plan

### 1. Scheduling Policies
I plan to implement multiple scheduling approaches and compare their trade-offs:

- **Round Robin (baseline)**
- **Least-Loaded** (with exponential moving average)
- **Power-of-Two Choices**
- **Parallel Prefix** (batch placement using prefix sums)
- **Dynamic Microbatching** (combine requests into small batches)
- **SLO-Aware Shortest Job First** (prioritize shorter jobs under latency goals)
- **Prefill / Decode Split** (separate worker pools for context vs. token generation)
- **KV-Cache Locality** (keep the same session on the same worker)
- **Fair Queuing** (per-tenant weights for fairness)
- **Hedged Requests** (duplicate slow requests to reduce tail latency)
- **Admission Control** (reject jobs when queues or memory exceed thresholds)

---

### 2. Worker Management
- Workers register and send **heartbeats** to the balancer.  
- Simulated “inference” jobs consume **CPU cycles proportional to tokens**.  
- Jobs carry **session and tenant IDs** to test fairness and cache reuse.  

---

### 3. Observability & Autoscaling
- Metrics: **throughput**, **p50/p95 latency**, **queue depth**, **admission rates**, **KV usage**.  
- **Prometheus + Grafana** dashboards to visualize results.  
- Simulated **autoscaling** (add/remove workers when latency grows too high).  

---

### 4. Deployment
- **Local**: Go binaries + Docker Compose.  
- **Optional**: Kubernetes manifests with Horizontal Pod Autoscaler (HPA).  

---

### Stretch Goals
- **Work Stealing** between workers.  
- **Canary Routing** (compare two policies side-by-side).  
- **Lightweight Web Dashboard** (Next.js) to visualize scheduling & metrics.  

