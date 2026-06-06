# Models

The `models` package holds the internal representation of policies and assignments.
There are separate abstraction hierarchies for Policy and Assignment.

---

## Policy — 3 layers

```
Layer 1  models/policy.go
         Policy{ APIVersion, Kind, Provider, Metadata, Resource yaml.Node, Spec yaml.Node }

         Provider-agnostic envelope. Resource and Spec are held as raw yaml.Node
         and decoded only once the Provider field is known.

Layer 2  models/azure/azure.go
         AzureResource{ SubscriptionID, Type, Vault }

         Common to all Azure resource kinds. Identifies WHERE the backup lives
         (vault, subscription). The Type field routes to the correct Layer 3 spec.

Layer 3  models/azure/policies/vm.go        → AzureVMSpec
         models/azure/policies/disk.go      → AzureDiskSpec
         models/azure/policies/sqlvm.go     → AzureSQLVMSpec

         Resource-kind-specific spec — HOW the backup is configured
         (retention counts, schedule frequency/time, instant RP settings, etc.).
         Shared building blocks (AzureRetention, AzureSchedule, etc.) live in
         models/azure/policies/azure_policies.go.
```

---

## Assignment — 2 layers

```
Layer 1  models/assignment.go
         Assignment{ APIVersion, Kind, Provider, Metadata, Assignments yaml.Node }

         Provider-agnostic envelope. Assignments is held as raw yaml.Node
         and decoded only once the Provider field is known.
         No resource block — vault and subscription are inherited from the
         referenced policy at resolve time.

Layer 2  models/azure/assignments/vm_assignment.go
         VMConfig → []VMAssignment → []VMTarget{ ResourceGroup, ResourceName, *DiskSelection }

         Azure-specific assignment targets. Each VMAssignment names a policy
         and lists the VM targets that should be assigned to it.
         DiskSelection is optional; omitting it backs up all disks.
```

---

## Resolved types

`models/azure/resolved.go` — `ResolvedVM`

After the engine resolves an assignment against its policy, all data is merged
into a flat `ResolvedVM`. This is the type passed to `external/azure` for diffing
and applying. It contains no yaml.Node fields — everything is fully decoded.

---

## Rules

- Layer 1 files must not import any provider package.
- Layer 2 and 3 files must not import `models` (no upward dependency).
- New providers follow the same pattern: add `models/<provider>/` with the
  same Layer 2 + Layer 3 split.
- New Azure resource kinds add a Layer 3 file in `models/azure/policies/` and
  a corresponding assignment file in `models/azure/assignments/`.
