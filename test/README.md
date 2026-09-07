# Manual test suite — Azure VM policy/assignment

Each `caseN/` folder is a self-contained `vm_policy.yaml` + `vm_assignment.yaml`
pair, runnable directly with `--dir test/caseN`. All cases target
`rg-bpac/vm-bpac` against `recovery-vault` (the only VM confirmed to be in the
same region as the vault) and use a unique policy name (`matrix-caseN`) so
they don't collide with each other or with any pre-existing policies in the
vault.

```
./backup-policy-as-code validate --dir test/case1
./backup-policy-as-code diff     --dir test/case1
./backup-policy-as-code apply    --dir test/case1
```

Since every case reassigns the *same* VM, only the most recently applied
case is actually protecting `vm-bpac` at any given time — applying case2
after case1 moves the VM's protection over to case2's policy. Case1's policy
definition stays behind in the vault, unused but harmless. Running all 16
cases through `apply` leaves ~16 permanent-but-inert policies in
`recovery-vault`; clean those up manually if that matters to you.

## Cases

| Case | Matrix ID | Config | Expected outcome |
|---|---|---|---|
| case1 | A1 | Standard, daily, daily retention only | Clean create |
| case2 | A2 | Standard, daily, daily+weekly (GFS) | Clean create |
| case3 | A3 | Standard, daily, +monthly | Clean create, monthly retention build |
| case4 | A4 | Standard, daily, full stack +yearly | Clean create, yearly anchor |
| case5 | A5 | Standard, **weekly** frequency, weekly retention only (no daily) | Clean create — the valid counterpart to case15 |
| case6 | A6 | Enhanced, **hourly** (interval 4h / window 24h) | Clean create, V2 HourlySchedule |
| case7 | A7 | Enhanced, daily | Clean create, V2 DailySchedule |
| case8 | B8 | `disk_selection` omitted | PROTECTED ITEM: all disks |
| case9 | B9 | `disk_selection: {mode: all}` | Same as case8, explicit |
| case10 | B10 | `disk_selection: {mode: include, luns: [0]}` | PROTECTED ITEM: only LUN 0 |
| case11 | B11 | `disk_selection: {mode: exclude, luns: [1]}` | PROTECTED ITEM: all except LUN 1 |
| case12 | D17 | `frequency: weekly`, no `day` | **Fails at `validate`** |
| case13 | D18 | `retention.weekly.day` ≠ `schedule.full.day` | **Fails at `validate`** |
| case14 | D19 | `retention.yearly` with no weekly/monthly anchor | **Fails at `validate`** |
| case15 | D20 | `frequency: weekly` + `retention.daily` present | Passes `validate`/`diff`, **fails at `apply`** (Azure `BMSUserErrorInvalidPolicyInput` — known gap, not caught locally) |
| case16 | D21 | `frequency: hourly`, no `schedule`/`duration` | **Fails at `validate`** |

case12–14 and case16 are validation-only by design — running `apply` on them
will just fail at the same `validate` step before ever reaching Azure.
case15 is the one negative case worth actually running through `apply`, to
confirm the (currently un-guarded) Azure-side rejection still reproduces.

## Procedural scenarios (not static folders)

These don't map to a single fixed file — they're a sequence of edits/commands
against any of the cases above. Run them by hand:

- **Idempotency** — `apply` any case, then `apply` it again unchanged.
  Second run must report `already assigned correctly` /
  `already in desired state` and `Applied 0 change(s)`.
- **Policy-definition-only change** — after applying e.g. case2, edit only
  `retention.weekly.count` and re-`diff`. Only POLICY DEFINITION should show
  a change; PROTECTED ITEM must stay `NO CHANGES`.
- **Protected-item-only change** — after applying e.g. case10, edit only
  `disk_selection.luns` and re-`diff`. Inverse of the above.
- **Policy rename / reassignment** — after applying one case, copy it to a
  new `metadata.name` and re-`apply`. The old policy is left untouched in
  the vault; the VM moves to the new one.
- **First-time registration on a genuinely unprotected VM** — needs a VM
  that has never been protected, in the same region as the vault. No such
  VM is set up as of writing this suite; this is the one scenario still
  worth chasing down if you get access to one, since it's the only path
  that exercises `ApplyVM`'s first-time-registration code rather than an
  update.
