<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/kada-banner-dark.svg">
    <img alt="kåda" src="assets/kada-banner.svg" width="420">
  </picture>
</p>

<p align="center">
  <strong>Declare, version, and apply cloud backup policies as code.</strong><br>
  The GitOps workflow, applied to backup administration.
</p>

<p align="center">
  <a href="https://kadacli.io"><strong>kadacli.io</strong></a>
  &nbsp;·&nbsp;
  <a href="#quickstart">Quickstart</a>
  &nbsp;·&nbsp;
  <a href="#cli-reference">CLI reference</a>
  &nbsp;·&nbsp;
  <a href="https://github.com/kadacli/kada/releases">Releases</a>
</p>

<p align="center">
  <a href="https://github.com/kadacli/kada/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/kadacli/kada/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://golang.org/"><img alt="Go Version" src="https://img.shields.io/badge/go-1.26+-00ADD8.svg"></a>
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/badge/license-Apache--2.0-blue.svg"></a>
</p>

---

## What is this?

**kåda** (`kada`) brings the declarative, GitOps-driven workflow of infrastructure
as code to backup administration. Instead of configuring backup policies through
cloud consoles or GUI wizards, you define them in YAML, commit them to Git,
review them in pull requests, and let `kada` diff and apply them against your
cloud.

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
  instant_rp_retention_days: 5
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
| **Azure Backup** | `azure_vm`, `azure_disk`, `azure_sql_db` | ✅ Supported |
| Azure SQL on VM | `azure_sql_vm` | 🚧 Planned |
| AWS Backup | — | 🚧 Planned |
| Google Cloud Backup | — | 🔭 On roadmap |
| Veeam | — | 🔭 On roadmap |
| Commvault | — | 🔭 On roadmap |

`azure_vm` and `azure_sql_db` use a Recovery Services Vault; `azure_disk` uses a
Backup Vault (`Microsoft.DataProtection`).

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

kåda uses the standard Azure SDK credential chain (`DefaultAzureCredential`).
Log in with the Azure CLI, or provide a service principal via environment
variables:

```bash
az login   # or set AZURE_TENANT_ID / AZURE_CLIENT_ID / AZURE_CLIENT_SECRET
```

The subscription is read from each Policy's `resource.subscription_id`, not from
the environment — one repository can span many subscriptions.

> **Never commit credentials or `.env` files.** Use least-privilege service
> principals and your CI/CD platform's secret store.

### 2. Scaffold a Policy and an Assignment

```bash
mkdir -p policies
kada init policy azure_vm policies/daily-vm-policy.yaml
kada init assignment azure_vm policies/daily-vm-assignment.yaml
```

`init` takes `<kind> <resource-type> <output-path>`, where kind is `policy` or
`assignment` and resource type is one of `azure_vm`, `azure_disk`,
`azure_sql_db`. It will not overwrite an existing file.

Edit the generated files to describe your desired backup configuration. See
[`test/examples/azure/`](test/examples/azure/) for complete Policy and
Assignment examples covering VMs, disks, and SQL.

### 3. Validate (offline)

```bash
kada validate --dir policies
```

No network calls — this checks schema, field ranges, and the cross-field rules
Azure would otherwise only reject at apply time.

### 4. Diff against live Azure

```bash
kada diff --dir policies
```

### 5. Apply

```bash
kada apply --dir policies
```

`apply` is idempotent: it diffs first and only writes the resources that are out
of desired state. It applies without prompting, so gate it behind your CD
pipeline's approval rather than expecting a confirmation step.

---

## CLI reference

```
Usage:
  kada [command]

Available Commands:
  validate    Validate Policy and Assignment YAML files without connecting to Azure
  diff        Compare local backup policies against live Azure configuration
  apply       Apply backup policies idempotently to Azure
  init        Scaffold a new Policy or Assignment YAML file
  completion  Generate the shell autocompletion script for the specified shell
  help        Help about any command

Persistent Flags:
      --dir string   Directory containing Policy and Assignment YAML files (default ".")
  -h, --help         Help for kada
```

`--dir` is scanned recursively. Hidden directories (`.git`, `.github`) and those
starting with `_` are skipped, as are YAML files that aren't kåda documents — a
policy directory can live alongside compose files and CI config without
tripping the tool up.

### Exit codes

| Code | Meaning | Emitted by |
|---|---|---|
| `0` | Compliant — files valid, nothing to change | `validate`, `diff`, `apply` |
| `1` | Drift found — live state differs from desired state | `diff` |
| `2` | Error — invalid input, auth failure, API error, bad usage | all commands |

These make `kada validate` and `kada diff` safe to gate pull requests on. A run
that finds no Policy or Assignment documents at all exits `2` rather than `0`,
so a wrong `--dir` fails loudly instead of reporting a green pipeline over an
empty scan.

```bash
# Fail the build when the environment has drifted
kada diff --dir policies || exit $?
```

### Shell completion

`kada completion <shell>` writes an autocompletion script to stdout for `bash`,
`zsh`, `fish`, or `powershell`. It completes subcommands and flags, and stays in
sync with the CLI automatically.

```bash
# bash — current shell only
source <(kada completion bash)

# bash — persistent
kada completion bash | sudo tee /etc/bash_completion.d/kada > /dev/null

# zsh
kada completion zsh > "${fpath[1]}/_kada"

# fish
kada completion fish > ~/.config/fish/completions/kada.fish
```

Run `kada completion <shell> --help` for the per-shell install notes.

---

## GitOps workflow

Treat your policy directory as infrastructure code:

```
┌──────────────────────────────────────────────────┐
│  Engineer edits Policy / Assignment YAML         │
│  and opens a pull request                        │
└────────────────────┬─────────────────────────────┘
                     ▼
┌──────────────────────────────────────────────────┐
│  CI runs: kada validate                          │
│           kada diff                              │
│  Diff output posted as a PR comment              │
└────────────────────┬─────────────────────────────┘
                     ▼
┌──────────────────────────────────────────────────┐
│  PR reviewed and merged to main                  │
└────────────────────┬─────────────────────────────┘
                     ▼
┌──────────────────────────────────────────────────┐
│  CD runs: kada apply                             │
│  Changes applied idempotently to Azure           │
└──────────────────────────────────────────────────┘
```

Because policy changes go through pull requests, Git *is* your approval
workflow, change history, and segregation of duties.

---

## Known limitations

- **Azure disk backup policies are immutable.** Azure refuses to update a disk
  policy in place, and refuses to delete one while a disk is protected by it. To
  change a disk policy, give it a new `metadata.name` and re-apply.
- **A VM's policy type cannot be changed in place.** Azure does not support
  moving an existing policy between Standard (V1) and Enhanced (V2), or
  downgrading a protected VM from Enhanced to Standard.
- **Paused backups must be resumed first.** A VM with "stop backup with retain
  data" cannot have its policy assignment managed until protection is resumed.
- **SQL time-based immutability is not yet supported**, pending `armsql` v2.

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

---

<p align="center">
  <a href="https://kadacli.io">kadacli.io</a>
  &nbsp;·&nbsp;
  <a href="mailto:hello@kadacli.io">hello@kadacli.io</a>
</p>
