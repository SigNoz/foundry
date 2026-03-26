# Examples

Deployment examples for SigNoz using Foundry. Each example includes a `casting.yaml` and a README with prerequisites, configuration, and deployment instructions.

| Deployment | Path | Description |
| --- | --- | --- |
| Docker Compose | [docker/compose](docker/compose/) | Single-node Docker deployment |
| Docker Swarm | [docker/swarm](docker/swarm/) | Multi-node Docker Swarm cluster |
| Systemd (binary) | [systemd/binary](systemd/binary/) | Bare metal Linux with systemd services |
| Kubernetes (Kustomize) | [kubernetes/kustomize](kubernetes/kustomize/) | Kubernetes with Kustomize manifests |
| Kubernetes (Kustomize + patches) | [kubernetes/kustomize-patches](kubernetes/kustomize-patches/) | Kustomize with resource, storage, and scheduling patches |
| Kubernetes (Helm) | [kubernetes/helm](kubernetes/helm/) | Kubernetes with the SigNoz Helm chart |
| Kubernetes (Helm + patches) | [kubernetes/helm-patches](kubernetes/helm-patches/) | Helm with resource, scheduling, and persistence patches |
| Render | [render/blueprint](render/blueprint/) | Render cloud platform |
| Coolify | [coolify/stack](coolify/stack/) | Coolify stack |
| Railway | [railway/template](railway/template/) | Railway template |
| AWS ECS (EC2 + Terraform) | [ecs/ec2/terraform](ecs/ec2/terraform/) | ECS on EC2 with Terraform |
