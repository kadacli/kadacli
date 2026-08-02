# Manual test suite — Azure Disk policy/assignment

Each `caseN/` folder is a self-contained `disk_policy.yaml` +
`disk_assignment.yaml` pair, runnable directly with `--dir
test/Azure/Disks/caseN`. Cases target two real disks:

- `rg-bpac` / disk `datalun2` (primary — used by most cases)
- `rg-bpac` / disk `data` (secondary — used by case5's fan-out and case6's
  second policy)

Every case uses a unique policy name (`matrix-disk-caseN`) so they don't
collide with each other or with any pre-existing policies in the vault.

```
./backup-policy-as-code validate --dir test/Azure/Disks/case1
./backup-policy-as-code diff     --dir test/Azure/Disks/case1
./backup-policy-as-code apply    --dir test/Azure/Disks/case1
```

**The vault is different from the VM/SQL suites.** Disk backup runs on
`Microsoft.DataProtection` (a **Backup Vault**), not the Recovery Services
Vault the VM suite uses — hence `vault.name: backup-vault` everywhere here.

### Prerequisite: vault identity role assignments

Without these, every case fails at `apply` with
`UserErrorNotEnoughPermissionOnDisk` — *after* the policy definition has
already been created in the vault (kåda upserts the policy first, then
registers the backup instance). The error comes from Azure's async operation
status, so there is nothing kåda can validate up front.

```sh
SUB=6a0bb3b0-b6fd-4690-94b0-67907ebe1e20
MSI=$(az dataprotection backup-vault show -g rg-bpac -v backup-vault \
        --query identity.principalId -o tsv)

# Read the source disks. RG scope covers both datalun2 and data.
az role assignment create --assignee "$MSI" --role "Disk Backup Reader" \
  --scope "/subscriptions/$SUB/resourceGroups/rg-bpac"

# Write snapshots. Same RG, since snapshot_resource_group is omitted here.
az role assignment create --assignee "$MSI" --role "Disk Snapshot Contributor" \
  --scope "/subscriptions/$SUB/resourceGroups/rg-bpac"
```

If `MSI` comes back empty the vault has no system-assigned identity yet —
enable it first (`az dataprotection backup-vault update -g rg-bpac -v
backup-vault --type SystemAssigned`, or via the portal). Role assignments take
a few minutes to propagate. Re-running `apply` after the grant is safe and
also exercises the idempotent path: the policy already exists with matching
retention, so only the backup instance registration runs.

`spec.snapshot_resource_group` is omitted in every case, so snapshots land in
the disk's own resource group (`ApplyDisk` defaults `snapshotRG` to
`disk.ResourceGroup`).

Like the SQL DB suite and unlike the VM suite, applying case3 after case1
does **not** leave case1 protecting the disk — each case is a distinct policy
name, so the backup instance for `datalun2` is repointed to the newest policy.
Old policy definitions stay behind in the vault, unused; clean them up
manually if that matters. Note that `CreateOrUpdateBackupPolicyDisk` **deletes
the policy before recreating it**, so a policy still attached to a backup
instance cannot be updated in place (see the gaps section).

## Cases

| Case | Matrix ID | Config | Expected outcome |
|---|---|---|---|
| case1 | A1 | Daily 04:00, `default: true` + weekly retention 1 | Clean create — baseline. Policy definition upserted, `datalun2` registered as a backup instance |
| case2 | A2 | Hourly, `interval: 4`, daily retention 7 | Clean create. Hourly is the only frequency that permits a `daily` retention rule (see case7). `RepeatingTimeIntervals` becomes `R/2023-01-01T00:00:00Z/PT4H` |
| case3 | A3 | Daily 04:00, weekly retention 4 | Clean create. Longer retention (28 days, inside the operational-store ceiling); run after case1 to see the disk move to a new policy |
| case4 | A4 | `retention.default: false` + weekly retention 2 | Clean create. Legal because `retention_rules` is non-empty. The mandatory `Default` rule is still emitted and keeps its **P7D fallback** — `default: false` does not remove it |
| case5 | B5 | One policy, two targets (`datalun2` + `data`) | Clean create on both — policy upserted once, two backup instances against the same policy ID. Confirms fan-out |
| case6 | B6 | Two distinct policies (A→`datalun2`, B→`data`) in one dir | Clean create; both resolve/apply independently. Multi-policy isolation (case5 was single-policy/two-target) |
| case7 | D7 | `frequency: daily` + `rule: daily` | **Fails at `validate`** — disk rule 2 ("RetentionRule[0] is daily and your backup schedule is daily") |
| case8 | D8 | `frequency: hourly`, `interval` omitted | **Fails at `validate`** — disk rule 3. The `omitempty,oneof=…` tag lets `0` through, so the rule catches it |
| case9 | D9 | `frequency: hourly` + `time: "04:00"` | **Fails at `validate`** — disk rule 3. `time` is meaningless for hourly (`buildDiskScheduleInterval` ignores it), so intent is rejected rather than dropped |
| case10 | D10 | `frequency: daily`, `time` omitted | **Fails at `validate`** — disk rule 4. Without it the interval string would be `R/2023-01-01T:00Z/P1D` |
| case11 | F11 | `frequency: weekly` | **Fails at `validate`** — `oneof=hourly daily`. Azure Disk Backup has no weekly *schedule*, only weekly *retention* |
| case12 | F12 | `interval: 3` | **Fails at `validate`** — `oneof=1 2 4 6 8 12`. The tag is wider than Azure itself (see gaps) |
| case13 | F13 | `retention.default: false`, no `retention_rules` | **Fails at `validate`** — disk rule 1, also caught by `parseDiskSpec`. Valid counterpart is case4 |
| case14 | F14 | Weekly retention `duration: 500` | **Fails at `validate`** — `max=436` |
| case15 | G15 | `resource_name: does-not-exist-disk` | Passes `validate`, **fails at `diff` and `apply`** — "Disk does not exist". `GetDisk` runs first in both, so nothing is written to the vault |
| case16 | G16 | Assignment references a policy name not present | **Fails at `validate`** (resolve) — "unknown policy". Note this comes from `resolveAzure`'s routing, not `ResolveDisks` (see gaps) |

case7–case14 and case16 are validate-only by design — `diff`/`apply` fail at
the same step before reaching Azure. case15 is the one negative case worth
running through `diff`/`apply`.

## Procedural scenarios (not static folders)

- **Idempotency** — `apply` any case, then `apply` it again unchanged. Second
  run must report `already assigned correctly` / `Applied 0 change(s)`.
- **Retention change** — copy case1 to a new `metadata.name`, change
  `retention_rules[0].duration`, and `apply`. `diff` shows
  `retention.weekly.duration N => M` and the disk moves to the new policy.
  Doing the same edit **in place** on an already-applied case fails instead —
  see gap 6.
- **Schedule change on an applied policy** — after applying case3, change only
  `schedule.time` (e.g. `04:00` → `22:00`) and re-`diff`. Because the policy is
  already protecting `datalun2`, this **fails with an explanatory error**
  telling you to rename the policy (gap 6) — it does *not* silently no-op, which
  is what it used to do (gap 1).
- **Schedule change via rename** — the working version of the above: change
  `schedule.time` *and* `metadata.name` (plus the assignment's `policy:`
  reference) and apply. `✔ Created policy`, then the disk moves to it. This is
  the only way to change a disk policy's schedule or retention.
- **Timezone change** — same, via rename. After `apply`, the Azure portal's
  "Backup frequency" should read e.g. "Every day at 04:00, W. Europe Standard
  Time" rather than UTC.
- **Reassignment** — after applying case1, apply case3. `datalun2`'s backup
  instance repoints at `matrix-disk-case3`; case1's definition stays in the
  vault, unused.
- **Fan-out then split** — apply case5 (both disks on one policy), then case6
  (each disk on its own policy) and confirm both instances move.
- **First-time registration on an unprotected disk** — case1 against a disk
  that has never been backed up exercises `ApplyDisk`'s create path
  (`remote == nil`) rather than the update path.

## Known gaps found while building this matrix

Listed roughly in order of how much they can bite a user; the ones since fixed
are struck through and kept for the record. Gaps 1–3 are the silent-intent-drop
class — the same class as the SQL DB per-target-override gap, which was also
fixed to fail loud.

1. ~~**Schedule changes are invisible to `diff` and `apply`.**~~ **Fixed.**
   `diskRetentionToMap` mapped only retention rules, so changing `frequency`,
   `time` or `interval` on an applied policy produced no diff and `apply`
   reported success without doing anything. `DiffDiskRetention` is now
   `DiffDiskPolicyDefinition` and compares `schedule.frequency`,
   `schedule.time` / `schedule.interval` and `schedule.timezone` alongside
   retention, parsing the remote side back out of the policy's ISO 8601
   `RepeatingTimeIntervals`. Only the keys meaningful for the frequency are
   emitted, so `time` (invalid for hourly) and `interval` (unused for daily)
   never compare against a value they don't control.

2. ~~**Declared retention rules other than `Default` never apply.**~~ **Fixed,
   confirmed against the live API** — the operational datastore accepts tagged
   retention rules, and a re-`diff` after apply reports `NO CHANGES`.
   `buildDiskBackupPolicy` emitted exactly one tagging criterion (`Default`),
   though its own comment said "one entry per retention rule plus Default" —
   the per-rule loop was never written. Azure applies a retention rule only to
   backups carrying a tag of the same name, so the `daily`/`weekly` rules were
   inert: present in the policy, retaining nothing. Only `Default` did any
   work, which meant `retention.default: false` silently fell back to 7 days —
   **case4's `weekly: 2` really retained 7 days.** The rule names were also
   lower-case where Azure's tags are capitalised, so they could not have
   matched a tag even if one had existed.

   Now each declared rule gets a tagging criterion (`FirstOfDay` at priority
   25, `FirstOfWeek` at 20, against `Default`'s 99) and the rule is named after
   its tag via `TagName()`, which both the builder and the differ use. The two
   flag states now mean something concrete:
   - `default: false` → `Default` keeps Azure's 7-day base and the declared
     rule adds a longer tier on top. Real GFS; case4 retains 7 days for every
     snapshot and 2 weeks for the first of each week.
   - `default: true` → `Default` mirrors the declared rule, so every snapshot
     is kept for the declared duration. Unchanged from before.

   **Migration note:** policies already in the vault were written with
   lower-case, untagged rules. They will now show a retention diff, and
   because an applied policy is immutable (gap 6) they need a new
   `metadata.name` — or delete the old policy once nothing is protected by it.

   ~~*Previously filed as: `retention.default: true` with no `retention_rules`
   breaks `diff` with `mapOne must not be nil or empty`.*~~ **Fixed.** The
   local map is no longer empty (it always carries schedule keys), and
   `diskRetentionToMap` now emits `retention.Default.*` so the rule that
   actually governs retention is compared — including when it is the only
   retention that exists. Durations are compared unit-aware (`4W`, `7D`)
   rather than as bare integers, which previously read "1 week" and "1 day"
   as the same value. `DefaultISODuration()` is now the single source of that
   value for both the policy builder and the differ.

3. ~~**Per-target overrides are parsed and dropped.**~~ **Fixed.**
   `DiskTarget.Overrides` (`overrides.retention.operational.count`) decoded
   fine, but `ResolvedDisk` has no `Overrides` field, so `ResolveDisks` could
   not carry it and a user writing an override silently got the policy's
   value. `ResolveDisks` now rejects the block at validate time with the same
   wording used for `azure_sql_db`.

4. ~~**The `init` template documents fields that don't exist.**~~ **Fixed.**
   `cmd/templates/policy/azure_disk.yml` showed
   `snapshot_resource_group_prefix` / `snapshot_resource_group_suffix` where
   the model field is `snapshot_resource_group` (singular, a full RG name),
   and set `interval: 12` next to `frequency: daily` where `interval` is
   unused — three inert keys. The template now uses the real field, comments
   `interval` out as hourly-only, and documents what `retention.default`
   actually does.

   Verifying this turned up a bigger one: **`kada init` never worked at all.**
   `cmd/init.go` declared `var templatesFS embed.FS` with no `//go:embed`
   directive, so the filesystem was empty and every kind/resource-type
   combination failed with "No template found". The directive is now present
   and all six combinations render.

5. ~~**`metadata.timezone` is ignored for disks.**~~ **Fixed.**
   `ResolveDisks` never populated `ResolvedDisk.Timezone` and
   `buildDiskBackupPolicy` never set `BackupSchedule.TimeZone`, so a `"04:00"`
   daily backup under `timezone: "W. Europe Standard Time"` ran at 04:00
   **UTC** — an hour or two off the declared intent, silently. The zone is now
   carried through and sent (falling back to an explicit `UTC` if the policy
   somehow has none), and it participates in the diff via gap 1's fix.
   The `init` templates for disk and SQL DB also shipped IANA zone IDs
   (`Europe/Copenhagen`) where Azure wants Windows IDs; both now match the VM
   template. Round-trip confirmed against the live API: after an apply, the
   zone comes back on the GET and a re-`diff` reports `NO CHANGES`, so
   `armdataprotection` v1.x does persist `BackupSchedule.TimeZone`.

6. **An applied disk policy is immutable** — confirmed against the live API,
   not fixable in kåda.
   Azure rejects `CreateOrUpdate` on an existing policy with
   `UserErrorBMSUpdatePolicyNotSupported` ("Update of existing policy is not
   supported. Please create a new policy."), and rejects deleting one that a
   backup instance is using with `UserErrorDppPolicyObjectInUse`. Together
   those make an applied policy read-only: **the only way to change a disk
   policy's schedule or retention is to give it a new `metadata.name`.**
   `CreateOrUpdateBackupPolicyDisk` therefore deletes-then-creates (the update
   call is a guaranteed 400 and is not attempted), and both `diff` and `apply`
   now detect this case and return a one-line instruction to rename instead of
   two nested Azure error blobs. Delete-then-create is only reachable for a
   policy nothing is protecting, which bounds the non-atomic window: a failure
   after the delete can only lose an unused policy.

   This is a real product constraint worth surfacing in user docs — it means
   disk policies are effectively versioned by name, unlike VM policies, which
   Azure does allow updating in place.

7. **Interval tag is wider than Azure.**
   `oneof=1 2 4 6 8 12` accepts `1`, `2` and `8`; Azure Disk Backup only
   supports 4-, 6- and 12-hour intervals. Those values pass `validate` and are
   rejected at `apply` (case12 uses `3` to hit the tag itself).

8. **`max=436` is applied to two different units.**
   The same tag bounds daily rules (days) and weekly rules (weeks), and
   Azure's operational datastore caps disk snapshot retention far lower.
   Values well under 436 can still be rejected at `apply`. Also
   `duration` is `omitempty`, so a rule with no `duration` passes `validate`
   and sends `P0D`/`P0W` to Azure.

9. ~~**The `Default` retention rule is never diffed.**~~ **Fixed** as part of
   gap 2 — `diskRetentionToMap` now emits `retention.Default.*`, so the rule
   that actually governs retention is compared. Flipping `retention.default`
   alone produces a real diff.

10. **`ResolveDisks`' unknown-policy guard is mostly unreachable.**
    `resolveAzure` routes each assignment by its *first* policy reference and
    fails there with `Assignment %q: unknown policy %q`. `ResolveDisks`' own
    `references unknown policy` message only fires when a *later* entry in the
    same assignment file is bad. case16 exercises the routing guard.

11. **`snapshot_resource_group` changes were a silent no-op** — **fixed.**
    The field was read only when building the backup instance and appeared in
    no comparison map, so editing it on a protected disk produced `NO CHANGES`
    and an apply that did nothing. It is now compared as
    `snapshotResourceGroupID` in the protected-item map, sourced from
    `ResolvedDisk.SnapshotResourceGroupID()` so the value compared is the value
    sent. Confirmed round-tripping against the live API.

12. **Duplicate retention rules of the same type were accepted** — **fixed.**
    Two `weekly` rules validated cleanly, then produced two Azure rules and two
    tagging criteria all named `Weekly`, while the local map key collided so
    only the last was compared. Now rejected at validate.

13. **A malformed assignment silently protected nothing** — **fixed**, and the
    worst thing this suite turned up.
    `ResolveDisks` (and `ResolveSQLDBs`) skipped an assignment whose block
    failed to decode, commented "not a disk assignment" — but `resolveAzure`
    has already routed by resource type, so the only way to reach it is a
    malformed file. The result was `All files valid — 0 resolved target(s)`
    and exit 0: in CI, a green pipeline protecting nothing. Both resolvers now
    report the decode error, and `loadAndValidate` fails when assignments are
    loaded but nothing resolves, so no future silent skip can pass as success.

14. ~~**Cosmetic: numeric validation errors print as characters.**~~ **Fixed.**
    `WrapValidationError` formatted the offending value with `%q`, so case12
    reported `'\x03'` instead of `3` and case14 `'Ǵ'` instead of `500`; disk
    rule 1 was also the only rule that omitted the `sourcePath` prefix, so
    case13's failure didn't say which file it came from. Both are corrected,
    and failed files now render like successful ones —
    `✖ Policy "name" (path): reason`.
