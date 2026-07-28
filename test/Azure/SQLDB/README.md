# Manual test suite — Azure SQL DB policy/assignment

Each `caseN/` folder is a self-contained `sqldb_policy.yaml` +
`sqldb_assignment.yaml` pair, runnable directly with `--dir
test/Azure/SQLDB/caseN`. Cases target two real databases:

- `rg-bpac` / server `bpac-sql` / database `prod` (primary — used by most cases)
- `rg-bpac` / server `sql-db-dk-east` / database `prod-db-2` (secondary — used
  by case9's multi-target fan-out)

Each case uses a unique policy name (`matrix-sqldb-caseN`) so they don't
collide with each other or with any pre-existing policies.

```
./backup-policy-as-code validate --dir test/Azure/SQLDB/case1
./backup-policy-as-code diff     --dir test/Azure/SQLDB/case1
./backup-policy-as-code apply    --dir test/Azure/SQLDB/case1
```

Unlike the VM suite, every case here reassigns the retention policy directly
on the target database (STR/LTR/redundancy are properties of the database
itself, not a Recovery Services Vault) — so applying case2 after case1
overwrites `prod`'s backup config in place. There's no separate "protected
item" pointer to move.

## Important gap found while building this matrix

`validateAzureSqlDbRules()` in `internal/engine/azure_engine/validator.go` used
to be an **unimplemented stub**. Rules 1–3 (retention bounds) are covered by
struct tags on `AzureSqlDbLtr` (`sqldb.go`); rule 4 ("week_of_year required if
yearly_retention is set") is now **enforced at validate time** — see case11,
which fails at `validate` rather than only at `apply`.

Separately, `cmd/templates/policy/azure_sql_db.yml` documents
`differential_backup_frequency`'s default as `24`, but no defaulting logic
exists in `AzureSqlDbSTR` or `ApplySqlDbSTR` — omitting the field sends `0`
to Azure, which isn't a valid `DiffBackupIntervalInHours`. See case12. This
looks like a real discrepancy between the template's documentation and the
implementation, worth a closer look independent of this test matrix.

## Cases

| Case | Matrix ID | Config | Expected outcome |
|---|---|---|---|
| case1 | A1 | STR only, no `long_term_retention` block | Clean create; `ApplySqlDbLTR` never called |
| case2 | A2 | STR + LTR weekly only | Clean create |
| case3 | A3 | STR + LTR weekly + monthly | Clean create |
| case4 | A4 | STR + LTR weekly + monthly + yearly + `week_of_year` | Clean create — valid counterpart to case11 |
| case5 | A5 | `redundancy: Zone` | Clean create/update |
| case6 | A6 | `redundancy: Geo` | Clean create/update |
| case7 | A7 | `redundancy: GeoZone` | Clean create/update — tier/region-dependent, may reject at Azure if unsupported |
| case8 | B8 | `differential_backup_frequency: 12` (non-default) | Clean create; STR diff picks up the change |
| case9 | B9 | One policy, two targets on two servers | Clean create on both; confirms fan-out |
| case10 | B10 | Assignment target sets `overrides.short_term_retention.retention_days: 30` | **Rejected at `validate`** — per-target overrides are not supported for `azure_sql_db`; `ResolveSQLDBs` errors instead of silently applying the policy's `14`. |
| case11 | D11 | `long_term_retention.yearly_retention` set, `week_of_year` omitted | **Fails at `validate`** — rule 4 in `validateAzureSqlDbRules()` now catches the missing `week_of_year` locally (previously only Azure rejected it at `apply`) |
| case12 | D12 | `differential_backup_frequency` omitted entirely | **Clean create** — `ResolveSQLDBs` defaults the omitted frequency to `24` (the template's documented default), so a valid value reaches Azure instead of `0`. |
| case13 | D13 | `resource_name: does-not-exist-db` | Passes `validate`, **fails at `diff` and `apply`** — "SQL database does not exist" |
| case14 | D14 | `long_term_retention.ltr_immutability: true` (with weekly retention) | **Fails at `validate`** — immutability is rejected in `validateAzureSqlDbRules` ("not currently supported, pending armsql v2"). No Azure call. |

Negative cases split by where they fail: case10 (unsupported override) and
case11 (missing `week_of_year`) now fail at **`validate`**, with no Azure call.
case13 and case14 require the target database to actually exist (`GetSqlDb` runs
first in both `diff` and `apply`), so they pass `validate` and must be exercised
with `diff`/`apply` to see the intended failure.

## Extended matrix (case15–case26)

Cases 15–18 exercise change/update paths and a dead field; 19–23 are pure
validate-time negatives (struct-tag bounds); 24–26 cover resolver/structural
paths. Cases marked "after caseN" are **procedural** — they assert an
update/transition, so apply the named base case first.

| Case | Matrix ID | Config | Expected outcome |
|---|---|---|---|
| case15 | E15 | LTR increase (weekly 8 / monthly 6 / yearly 7, wk 20) — **run after case4** | `diff` shows N=>M on all three LTR keys; `apply` overwrites existing LTR. Exercises update-in-place (not just 0=>N create). |
| case16 | E16 | LTR reduce + drop yearly (weekly 2 / monthly 1) — **run after case15** | `diff` shows weekly/monthly lowered and `yearly 7 => 0`; confirms Azure accepts an LTR downgrade and yearly-only removal. |
| case17 | E17 | STR `retention_days: 35` (from 14) | Clean update; `str.retentionDays` diff key deliberately exercised. Also valid upper-boundary (max=35). |
| case18 | E18 | `long_term_retention.lock_immutability: true` (+ weekly) | **Fails at `validate`** — `lock_immutability` is now rejected in `validateAzureSqlDbRules` ("not currently supported, pending armsql v2"), instead of being silently ignored at apply. |
| case19 | F19 | `redundancy: GRS` | **Fails at `validate`** — `oneof=Local Zone Geo GeoZone`. |
| case20 | F20 | `retention_days: 36` | **Fails at `validate`** — STR `max=35`. |
| case21 | F21 | `differential_backup_frequency: 6` | **Fails at `validate`** — `oneof=12 24` (non-zero invalid; distinct from case12's omitted/defaulted). |
| case22 | F22 | `yearly_retention: 5`, `week_of_year: 53` | **Fails at `validate`** — LTR `week_of_year` `max=52` struct tag (value present but out of range; distinct from case11's omitted). |
| case23 | F23 | `retention_days` omitted | **Fails at `validate`** — STR `required` (no default, unlike `differential_backup_frequency`). |
| case24 | G24 | Assignment references a policy name not present | **Fails at `validate`** (resolve) — "references unknown policy". |
| case25 | G25 | Two distinct policies (A→`prod`, B→`prod-db-2`) in one dir | Clean create; both resolve/apply independently. Multi-policy isolation (case9 was single-policy/two-target). |
| case26 | G26 | Two policy files share `metadata.name` | **Fails at `validate`** (resolve) — "duplicate policy name". |

## Procedural scenarios (not static folders)

- **Idempotency** — `apply` any case, then `apply` it again unchanged.
  Second run must report `already in desired state` / `Applied 0 change(s)`.
- **STR-only change** — after applying e.g. case2, edit only
  `short_term_retention.retention_days` and re-`diff`. Only `str.*` keys
  should show a change.
- **Redundancy-only change** — after applying e.g. case1, edit only
  `redundancy` and re-`diff`. Only `db.redundancy` should show a change.
- **LTR removal** — after applying e.g. case2 (which sets LTR), remove the
  `long_term_retention` block entirely and re-`diff`/`apply`. This now clears
  the remote LTR policy (`✔ Cleared LTR policy`) and is idempotent on re-run.
  (This is exactly what case1 exercises against a DB that already has LTR.)
- **Serverless auto-pause** — if either target database is serverless with
  auto-pause enabled, applying any case with LTR set (case2–4, 8, 14–16, 18)
  should hit the `LtrConfigPolicyUnsupportedIfAutoPauseEnabled` branch in
  `ApplySqlDb` and emit a warning instead of failing.

## Dead-field decisions (resolved)

Two fields were accepted in YAML but did nothing at `diff`/`apply` — the same
silent-intent-drop class as the (now-fixed) per-target override gap. Both are
now resolved:

- **`spec.schedule` — removed.** `AzureSqlDbSchedule` (frequency / time /
  timezone) was `validate:"required"` yet read nowhere. Azure SQL PITR/LTR is
  not schedule-configurable, so the field and type were dropped from the SQL DB
  model, the `init` template, and every fixture. (VM/disk keep their own
  separate schedule types — unaffected.)
- **Immutability — rejected at `validate`.** `lock_immutability` was fully dead
  and `ltr_immutability` was only caught at `apply`. Both are now rejected in
  `validateAzureSqlDbRules` while the SDK (armsql v1.2.0) can't support them,
  so intent fails loud. The fields are kept (they map to a real armsql v2
  feature) rather than deleted, which would reintroduce silent-ignore. The
  old apply-time guard in `ApplySqlDbLTR` was removed as now-unreachable.
