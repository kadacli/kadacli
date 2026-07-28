# kåda

> Declare, version, and apply cloud backup policies as code — the GitOps workflow, applied to backup administration.

[![CI](https://github.com/kadacli/kada/actions/workflows/ci.yml/badge.svg)](https://github.com/kadacli/kada/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.26+-00ADD8.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

---

## What is this?

**kåda** (`kada`) brings the declarative, GitOps-driven workflow of tools like
IaC to backup administration. Instead of configuring backup policies
through cloud consoles or GUI wizards, you define them in YAML, commit them to
Git, review them in pull requests, and let `kada` diff and apply them against
your cloud.

```yaml
apiVersion: kadacli/v1
kind: Policy
provider: azure
metadata:
  name: daily-vm-policy
  environment: production
  timezone: "W. Europe Standard Time"
  owner: ops@company.com
resource:
  subscription_id: "00000000-0000-0000-0000-000000000000"
  type: azure_vm
  vault:
    name: recovery-vault
    resource_group: rg-backup
spec:
  policy_type: Standard
  retention:
    daily:
      count: 30
  schedule:
    full:
      frequency: daily
      time: "02:00"
```

---

## Why?

Backup configuration is one of the last areas of infrastructure that hasn't been
properly codified. Today it's managed through GUIs, tribal knowledge, and
undocumented click-ops. That leads to:

- **Configuration drift** — no single source of truth across environments
- **No auditability** — who changed which retention policy, and when?
- **Manual toil** — onboarding a workload means clicking through wizards
- **Inconsistency** — a different mental model for every provider and team

With kåda, your Git repository becomes the authoritative source of truth for
every backup policy in your organisation.

---

## Core concepts

kåda models backups as two kinds of YAML document:

| Kind | Purpose |
|---|---|
| **`Policy`** | *How* to back up — schedule, retention, vault, policy type. |
| **`Assignment`** | *What* to back up — which resources a Policy applies to. |

Assignments reference a Policy by `metadata.name` and inherit its vault and
subscription, so one Policy can protect many resources.

---

## Supported providers

| Provider | Resource types | Status |
|---|---|---|
| **Azure Backup** (Recovery Services Vault) | `azure_vm`, `azure_disk`, `azure_sql_vm`, `azure_sql_db` | ✅ v1.0 |
| AWS Backup | — | 🚧 Planned |
| Google Cloud Backup | — | 🚧 Planned |
| Veeam | — | 🔭 On roadmap |
| Commvault | — | 🔭 On roadmap |

---

## Installation

### Download a binary

Download the latest release for your platform from the
[releases page](https://github.com/kadacli/kada/releases).

```bash
# Linux (amd64)
curl -L https://github.com/kadacli/kada/releases/latest/download/kada_linux_amd64.tar.gz | tar xz
sudo mv kada /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/kadacli/kada/releases/latest/download/kada_darwin_arm64.tar.gz | tar xz
sudo mv kada /usr/local/bin/
```

### Build from source

```bash
git clone https://github.com/kadacli/kada.git
cd kada
go build -o kada .
```

Requires Go 1.26 or later.

---

## Quickstart

### 1. Authenticate to Azure

kåda uses the standard Azure SDK credential chain. Set your subscription and log
in with the Azure CLI, or provide a service principal via environment variables:

```bash
export AZURE_SUBSCRIPTION_ID="your-subscription-id"
az login   # or set AZURE_TENANT_ID / AZURE_CLIENT_ID / AZURE_CLIENT_SECRET
```

> **Never commit credentials or `.env` files.** Use least-privilege service
> principals and your CI/CD platform's secret store.

### 2. Scaffold a Policy and an Assignment

```bash
kada init Policy azure_vm policies/daily-vm-policy.yaml
kada init Assignment azure_vm policies/daily-vm-assignment.yaml
```

Edit the generated files to describe your desired backup configuration. See
[`examples/azure/`](examples/azure/) for complete Policy and Assignment examples
covering VMs, disks, and SQL.

### 3. Validate (offline)

```bash
kada validate --dir policies
```

### 4. Diff against live Azure

```bash
kada diff --dir policies
```

### 5. Apply

```bash
kada apply --dir policies
```

---

## CLI reference

```
Usage:
  kada [command]

Available Commands:
  validate    Validate policy and assignment YAML files without connecting to Azure
  diff        Compare local backup policies against live Azure configuration
  apply       Apply backup policies idempotently to Azure
  init        Scaffold a new Policy or Assignment YAML file

Persistent Flags:
  --dir string       Directory containing Policy and Assignment YAML files (default ".")
  --output string    Output format: text | json (default "text")
  --verbose          Enable verbose output
  -h, --help         Help for kada

Exit Codes:
  0   Success / compliant
  1   Violations or drift found
  2   Error
```

The `0 / 1 / 2` exit codes make `kada validate`, `diff`, and `apply` safe to gate
pull requests and pipelines on.

---

## GitOps workflow

Treat your policy directory as infrastructure code:

```
┌─────────────────────────────────────────────────┐
│  Engineer edits Policy / Assignment YAML         │
│  and opens a pull request                        │
└────────────────────┬─────────────────────────────┘
                     ▼
┌─────────────────────────────────────────────────┐
│  CI runs: kada validate                          │
│           kada diff                              │
│  Diff output posted as a PR comment              │
└────────────────────┬─────────────────────────────┘
                     ▼
┌─────────────────────────────────────────────────┐
│  PR reviewed and merged to main                  │
└────────────────────┬─────────────────────────────┘
                     ▼
┌─────────────────────────────────────────────────┐
│  CD runs: kada apply                             │
│  Changes applied idempotently to Azure           │
└──────────────────────────────────────────────────┘
```

Because policy changes go through pull requests, Git *is* your approval workflow,
change history, and segregation of duties.

---

## Contributing

Contributions, bug reports, and ideas are welcome. Please read
[CONTRIBUTING.md](CONTRIBUTING.md) and our [Code of Conduct](CODE_OF_CONDUCT.md),
and open an issue before starting significant work.

## Security

Found a vulnerability? Please **do not** open a public issue — see
[SECURITY.md](SECURITY.md) for how to report it privately.

## License

kåda is licensed under the [Apache License 2.0](LICENSE).
