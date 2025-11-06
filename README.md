# GCP Validation Pipeline - Demo

This demo showcases a Tekton Pipeline that validates GCP cluster configurations with 3 validators that have dependency relationships.

## Architecture

### Validation Chain

```
        API Validator (Go)
              ↓
    ┌─────────┴─────────┐
    ↓                   ↓
Quota Validator    Network Validator
   (Bash)              (Bash)
    └─────────┬─────────┘
              ↓
    Results Aggregator
      (Finally Task)
```

The API validator must complete first, then Quota and Network validators run **in parallel**.

### Validators

All validators support a `force-failure` parameter to easily simulate success/failure scenarios.

1. **API Validator** (Go-based)
   - Validates required GCP APIs are enabled
   - Default: Fails (iam.googleapis.com not enabled)
   - Control: `api-force-failure` parameter

2. **Quota Validator** (Bash-based)
   - Validates CPU, Memory, IP quotas
   - Depends on API Validator, runs in parallel with Network Validator
   - Default: Passes
   - Control: `quota-force-failure` parameter

3. **Network Validator** (Bash-based)
   - Validates VPC, subnet, IP availability
   - Depends on API Validator, runs in parallel with Quota Validator
   - Default: Passes
   - Control: `network-force-failure` parameter

## Prerequisites

- Tekton Pipelines installed on Kubernetes cluster
- `kubectl` configured
- `tkn` CLI installed (optional, for easier log viewing)
- Podman (for building the Go validator image)

## Quick Start

### 1. Build the API Validator image, and push image to Quay

Open Makefile, update IMAGE_REGISTRY and IMAGE_NAME if you want to build a new image for api-validator. (Here it uses my quay registry as an example)
```bash
# build image
make build-image

# push image
podman push $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

# Then go to the image setting page, set the image as public
# Below link is an example that using my quay registry
https://quay.io/repository/rh-ee-dawang/gcp-api-validator?tab=settings
```

**Note:** If you use different image, please don't forget to update the image used in tekton/tasks/api-validator-task.yaml 

### 2. Apply Tekton Resources

```bash
# Apply all tasks
kubectl apply -f tekton/tasks/

# Apply pipeline
kubectl apply -f tekton/pipelines/gcp-validation-pipeline.yaml
```

### 4. Run the Pipeline

```bash
kubectl create -f tekton/pipelines/validation-pipelinerun.yaml

# Watch logs (requires tkn CLI)
# View all pipelinerun
tkn pipelinerun logs -f -L
# View one
tkn pipelinerun logs <pipelinerun name>
```

### 5. View Results

```bash
# Get the latest PipelineRun results
PIPELINERUN=$(kubectl get pipelinerun -l app=gcp-validation --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}')
kubectl get pipelinerun $PIPELINERUN -o jsonpath='{.status.results}' | jq .
```

## Expected Results

With default parameters (API fails, Quota and Network pass):

- **Overall Status**: FAILED
- **Passed**: 2/3 validations
- **Failed**: 1/3 validations (API - iam.googleapis.com not enabled)

The aggregation task (in the `finally` block) shows detailed results from all three validations.

## Demo Customization

### Change Validation Behavior

You can easily simulate different scenarios by modifying the `force-failure` parameters in the PipelineRun:

#### Scenario 1: API fails, Quota and Network pass (Default)
```yaml
# tekton/pipelines/validation-pipelinerun.yaml
params:
  - name: api-force-failure
    value: "true"
  - name: quota-force-failure
    value: "false"
  - name: network-force-failure
    value: "false"
```

#### Scenario 2: All validators pass
```yaml
params:
  - name: api-force-failure
    value: "false"
  - name: quota-force-failure
    value: "false"
  - name: network-force-failure
    value: "false"
```

#### Scenario 3: All validators fail
```yaml
params:
  - name: api-force-failure
    value: "true"
  - name: quota-force-failure
    value: "true"
  - name: network-force-failure
    value: "true"
```

#### Scenario 4: API passes, Quota fails, Network passes
```yaml
params:
  - name: api-force-failure
    value: "false"
  - name: quota-force-failure
    value: "true"
  - name: network-force-failure
    value: "false"
```

After modifying the parameters, simply run the pipeline again:
```bash
kubectl create -f tekton/pipelines/validation-pipelinerun.yaml
```

## Project Structure

```
├── validators/             # Validator implementations
│   ├── api-validator.go   # Go-based API validator
│   └── go.mod
├── docker/                 # Container images
│   └── Dockerfile.api-validator
├── tekton/
│   ├── tasks/             # Task definitions (including bash validators)
│   │   ├── api-validator-task.yaml
│   │   ├── quota-validator-task.yaml
│   │   └── network-validator-task.yaml
│   └── pipelines/
│       ├── gcp-validation-pipeline.yaml
│       └── validation-pipelinerun.yaml
├── Makefile
├── README.md              # This file - Quick start guide
└── ARCHITECTURE.md        # Technical details
```

## Troubleshooting

**View Logs**: Check pod logs for failures
```bash
kubectl logs -l app=gcp-validation
```

## Cleanup

```bash
kubectl delete pipelinerun -l app=gcp-validation
kubectl delete pipeline gcp-validation-pipeline
kubectl delete task gcp-api-validator gcp-quota-validator gcp-network-validator
```

## Additional Documentation

- **ARCHITECTURE.md** - Detailed technical architecture and data flow diagrams
- [Tekton Pipelines Documentation](https://tekton.dev/docs/pipelines/)
