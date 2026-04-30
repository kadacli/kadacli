# backup-policy-as-code

> Declare, version, and apply backup policies across cloud providers using YAML.

[![Build](https://github.com/yourname/backup-policy-as-code/actions/workflows/build.yml/badge.svg)](https://github.com/yourname/backup-policy-as-code/actions)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)

---

## What is this?

`backup-policy-as-code` brings the declarative, GitOps-driven workflow of tools like Terraform to backup administration. Instead of configuring backup policies through cloud consoles or GUI-based tools, you define them in YAML files, commit them to Git, and let the tool handle the rest.

```yaml
apiVersion: backupctl/v1
kind: BackupPolicy
metadata:
  name: production-vms
  environment: prod
  owner: backup-team@company.com

spec:
  targets:
    - type: azure_vm
      selector:
        resource_group: rg-production
        tags:
          backup: "true"

  schedule:
    full:
      cron: "0 2 * * 0"
    incremental:
      cron: "0 2 * * 1-6"
    timezone: "Europe/Copenhagen"

  retention:
    daily: 7
    weekly: 4
    monthly: 12
    yearly: 1

  storage:
    primary:
      provider: azure
      vault: backup-vault-prod
      redundancy: GRS
```

---

## Why?

Backup configuration is one of the last areas of infrastructure that hasn't been properly codified. Today, backup admins manage policies through GUIs, tribal knowledge, and undocumented configurations. This leads to:

- **Configuration drift** — no single source of truth across environments
- **No auditability** — who changed what policy, and when?
- **Manual toil** — onboarding a new workload means clicking through wizards instead of copying a YAML block
- **Inconsistency** — different mental models for Azure, AWS, GCP, and on-prem tools

`backup-policy-as-code` solves all of this. Your Git repository becomes the authoritative source of truth for every backup policy in your organisation.

---

## Features

- **Declarative YAML policies** — describe what you want, not how to do it
- **`plan` before you `apply`** — see exactly what will change before it happens, Terraform-style
- **Drift detection** — compare live provider state against your declared policies
- **Pluggable provider system** — one interface, multiple backends
- **Git-native** — every change is a commit, every review is a pull request
- **CI/CD ready** — run `validate` and `plan` in your pipelines

---

## Supported Providers

| Provider | Status |
|---|---|
| Azure Backup (Recovery Services Vault) | 🚧 Planned  |
| AWS Backup | 🚧 Planned |
| Google Cloud Backup | 🚧 Planned |
| OpenStack | 🚧 Planned |
| Veeam | 🔭 On roadmap |
| Commvault | 🔭 On roadmap |

---

## Installation

### Download binary

Download the latest release for your platform from the [releases page](https://github.com/yourname/backup-policy-as-code/releases).

```bash
# macOS (Apple Silicon)
curl -L https://github.com/yourname/backup-policy-as-code/releases/latest/download/bpac_darwin_arm64.tar.gz | tar xz
sudo mv bpac /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/yourname/backup-policy-as-code/releases/latest/download/bpac_linux_amd64.tar.gz | tar xz
sudo mv bpac /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/yourname/backup-policy-as-code.git
cd backup-policy-as-code
make build
```

Requires Go 1.21 or later.

---

## Quickstart

### 1. Set up your credentials

```bash
# Azure
export AZURE_SUBSCRIPTION_ID="your-subscription-id"
az login  # or use a service principal / managed identity
```

### 2. Create your first policy

```bash
mkdir policies
cat > policies/production-vms.yaml << EOF
apiVersion: backupctl/v1
kind: BackupPolicy
metadata:
  name: production-vms
  environment: prod
  owner: you@company.com
spec:
  targets:
    - type: azure_vm
      selector:
        resource_group: rg-production
  schedule:
    full:
      cron: "0 2 * * 0"
    incremental:
      cron: "0 2 * * 1-6"
    timezone: "Europe/Copenhagen"
  retention:
    daily: 7
    weekly: 4
    monthly: 12
  storage:
    primary:
      provider: azure
      vault: backup-vault-prod
      redundancy: GRS
EOF
```

### 3. Validate your policy

```bash
bpac validate
```

```
✔  production-vms   policies/production-vms.yaml

1 policy validated successfully.
```

### 4. Preview changes

```bash
bpac plan
```

```
bpac plan
─────────────
  + production-vms   (new policy — will be created)

Plan: 1 to add, 0 to change, 0 to destroy.
```

### 5. Apply

```bash
bpac apply
```

```
bpac plan
─────────────
  + production-vms   (new policy — will be created)

Do you want to apply these changes? (yes/no): yes

  ✔  production-vms   created in 3.2s

Apply complete. 1 added, 0 changed, 0 destroyed.
```

---

## CLI Reference

```
Usage:
  bpac [command]

Available Commands:
  validate    Validate policy YAML files without connecting to any provider
  plan        Show what changes would be applied
  apply       Apply policy changes to the provider
  diff        Compare local policies against live provider state

Flags:
  --policies-dir string   Directory containing policy YAML files (default "policies")
  --auto-approve          Skip confirmation prompt on apply
  -h, --help              Help for bpac

Environment Variables:
  AZURE_SUBSCRIPTION_ID   Required for Azure provider
```

---

## YAML Policy Reference

| Field | Type | Required | Description |
|---|---|---|---|
| `apiVersion` | string | ✅ | Must be `backupctl/v1` |
| `kind` | string | ✅ | Must be `BackupPolicy` |
| `metadata.name` | string | ✅ | Unique policy name |
| `metadata.environment` | string | ✅ | One of: `prod`, `staging`, `dev` |
| `metadata.owner` | email | ✅ | Owning team or person |
| `metadata.tags` | map | — | Arbitrary key/value tags |
| `spec.targets` | list | ✅ | Resources this policy applies to |
| `spec.targets[].type` | string | ✅ | One of: `azure_vm`, `aws_ec2`, `gcp_vm` |
| `spec.targets[].selector` | map | ✅ | Provider-specific resource selector |
| `spec.schedule.full.cron` | string | ✅ | Cron expression for full backups |
| `spec.schedule.incremental.cron` | string | — | Cron expression for incremental backups |
| `spec.schedule.timezone` | string | ✅ | IANA timezone string |
| `spec.retention.daily` | int | ✅ | Days to retain daily backups |
| `spec.retention.weekly` | int | — | Weeks to retain weekly backups |
| `spec.retention.monthly` | int | — | Months to retain monthly backups |
| `spec.retention.yearly` | int | — | Years to retain yearly backups |
| `spec.storage.primary.provider` | string | ✅ | One of: `azure`, `aws`, `gcp` |
| `spec.storage.primary.vault` | string | — | Target vault or bucket name |
| `spec.storage.primary.redundancy` | string | — | e.g. `LRS`, `GRS`, `ZRS` |
| `spec.storage.copies` | list | — | Additional offsite copies |
| `spec.restore_test.enabled` | bool | — | Enable automated restore testing |
| `spec.restore_test.schedule.cron` | string | — | Cron expression for restore tests |
| `spec.restore_test.notify.email` | string | — | Notification recipient |
| `spec.restore_test.notify.on` | list | — | One or more of: `success`, `failure` |

---

## Project Structure

```
backup-policy-as-code/
├── main.go
├── go.mod
├── Makefile
├── cmd/
│   ├── root.go
│   ├── validate.go
│   ├── plan.go
│   ├── apply.go
│   └── diff.go
├── internal/
│   ├── models/
│   │   └── policy.go          # YAML schema as Go structs
│   ├── engine/
│   │   ├── parser.go          # YAML loading and validation
│   │   └── planner.go         # Desired vs actual diff
│   ├── state/
│   │   └── state.go           # Local state management
│   └── providers/
│       ├── provider.go        # Provider interface
│       └── azure/
│           └── azure.go       # Azure Backup integration
└── policies/
    └── examples/
        └── example-policy.yaml
```

---

## GitOps Workflow

The recommended workflow is to treat your `policies/` directory as infrastructure code:

```
┌─────────────────────────────────────────────────┐
│  Developer edits policies/*.yaml                │
│  Opens a pull request                           │
└────────────────────┬────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────┐
│  CI runs: bpac validate                         │
│           bpac plan                             │
│  Plan output posted as PR comment               │
└────────────────────┬────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────┐
│  PR reviewed and merged to main                 │
└────────────────────┬────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────┐
│  CD runs: bpac apply --auto-approve             │
│  Changes applied to provider                    │
│  State updated                                  │
└─────────────────────────────────────────────────┘
```

A starter GitHub Actions workflow is available in [`.github/workflows/`](.github/workflows/).

---

## Contributing

Contributions are welcome. Please open an issue before submitting a pull request for significant changes.

```bash
# Run tests
make test

# Run linter
make lint

# Build binary
make build
```

---

## Licence


---

> Built by backup engineers who were tired of clicking through GUIs.
