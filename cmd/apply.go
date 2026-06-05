package cmd

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure_models/azure_policies"
)

// TODO: Refactor apply such that it only determines the provider and outsources to external

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply backup policies idempotently to Azure",
	Long: `Validates policies, diffs desired state against live Azure, and applies
only the resources that are out of desired state. Safe to run repeatedly.`,
	RunE: runApply,
}

func runApply(cmd *cobra.Command, args []string) error {
	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	dir := viper.GetString("dir")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	resolved, _, err := loadAndValidate(dir, true)
	if err != nil {
		return err
	}

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		subscriptionID = viper.GetString("subscription")
	}
	if subscriptionID == "" {
		return fmt.Errorf("subscription ID not set (use --subscription or AZURE_SUBSCRIPTION_ID)")
	}

	cred, err := azure_engine.Authenticate()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	var applyErrors []error
	applied := 0

	for _, rt := range resolved {
		subID := rt.SubscriptionID
		if subID == "" {
			subID = subscriptionID
		}

		switch rt.Target.Type {
		case "azure_vm":
			// Step 1: ensure the backup policy exists in the vault with correct retention.
			backupClient, err := armrecoveryservicesbackup.NewProtectionPoliciesClient(subID, cred, nil)
			if err != nil {
				applyErrors = append(applyErrors, fmt.Errorf("%s/%s: failed to create policy client: %w",
					rt.Target.ResourceGroup, rt.Target.ResourceName, err))
				continue
			}
			if rt.Policy.Spec.Schedule.Full == nil {
				applyErrors = append(applyErrors, fmt.Errorf("%s/%s: schedule.full is required for azure_vm",
					rt.Target.ResourceGroup, rt.Target.ResourceName))
				continue
			}

			azurePolicy := azure_policies.AzureRecoveryVMPolicy{
				ResourceGroup:                rt.Policy.Spec.Vault.ResourceGroup,
				VaultName:                    rt.Policy.Spec.Vault.Name,
				PolicyName:                   rt.Policy.Metadata.Name,
				BackupTime:                   rt.Policy.Spec.Schedule.Full.ParsedTime, // see note below
				Retention:                    rt.Policy.Spec.Retention,
				InstantRPRetentionDays:       rt.Policy.Spec.InstantRPRetentionDays,
				InstantRPResourceGroupPrefix: rt.Policy.Spec.InstantRPResourceGroupPrefix,
				InstantRPResourceGroupSuffix: rt.Policy.Spec.InstantRPResourceGroupSuffix,
			}

			if dryRun {
				fmt.Printf("[dry-run] would create/update backup policy %q in vault %s/%s\n",
					rt.Policy.Metadata.Name, rt.Policy.Spec.Vault.ResourceGroup, rt.Policy.Spec.Vault.Name)
			} else {
				if _, err := engine.CreateOrUpdateAzureBackupPolicy(cmd.Context(), backupClient, azurePolicy); err != nil {
					applyErrors = append(applyErrors, fmt.Errorf("%s/%s: policy upsert failed: %w",
						rt.Target.ResourceGroup, rt.Target.ResourceName, err))
					continue
				}
			}

			// Step 2: diff the VM's current protected item assignment.
			remote, err := engine.GetBackupPolicyVM(
				cred, subID,
				rt.Target.ResourceGroup,
				rt.Policy.Spec.Vault.Name,
				rt.Target.ResourceName,
			)
			if err != nil {
				applyErrors = append(applyErrors, fmt.Errorf("%s/%s: get remote failed: %w",
					rt.Target.ResourceGroup, rt.Target.ResourceName, err))
				continue
			}

			_, diff, _, _, err := engine.DiffVMPolicy(rt, remote)
			if err != nil {
				applyErrors = append(applyErrors, fmt.Errorf("%s/%s: diff failed: %w",
					rt.Target.ResourceGroup, rt.Target.ResourceName, err))
				continue
			}

			if len(diff) == 0 {
				fmt.Println(green(fmt.Sprintf("✔ %s/%s — already in desired state",
					rt.Target.ResourceGroup, rt.Target.ResourceName)))
				continue
			}

			if dryRun {
				fmt.Printf("[dry-run] would apply policy %q to %s/%s\n",
					rt.Policy.Metadata.Name, rt.Target.ResourceGroup, rt.Target.ResourceName)
				continue
			}

			// Step 3: apply the VM assignment.
			if err := engine.ApplyVM(cred, subID, rt); err != nil {
				applyErrors = append(applyErrors, fmt.Errorf("%s/%s: apply failed: %w",
					rt.Target.ResourceGroup, rt.Target.ResourceName, err))
				fmt.Fprintln(os.Stderr, red(fmt.Sprintf("✖ %s/%s: %v",
					rt.Target.ResourceGroup, rt.Target.ResourceName, err)))
			} else {
				applied++
				fmt.Println(green(fmt.Sprintf("✔ Applied policy %q to %s/%s",
					rt.Policy.Metadata.Name, rt.Target.ResourceGroup, rt.Target.ResourceName)))
			}
		case "azure_disk":
			azureClient, err := azure.NewClient(subID)
			if err != nil {
				applyErrors = append(applyErrors, fmt.Errorf("%s/%s: failed to create azure client: %w",
					rt.Target.ResourceGroup, rt.Target.ResourceName, err))
				continue
			}

			diskCfg := azure.AzureRecoveryDiskPolicy{
				ResourceGroup:               rt.Policy.Spec.Vault.ResourceGroup,
				VaultName:                   rt.Policy.Spec.Vault.Name,
				PolicyName:                  rt.Policy.Metadata.Name,
				BackupIntervalHours:         rt.Policy.Spec.Schedule.Full.Interval,
				RetentionDays:               rt.Policy.Spec.Retention.Operational.Count,
				SnapshotResourceGroupPrefix: rt.Policy.Spec.SnapshotResourceGroupPrefix,
				SnapshotResourceGroupSuffix: rt.Policy.Spec.SnapshotResourceGroupSuffix,
			}

			if dryRun {
				fmt.Printf("[dry-run] would create/update disk backup policy %q in vault %s/%s\n",
					rt.Policy.Metadata.Name, rt.Policy.Spec.Vault.ResourceGroup, rt.Policy.Spec.Vault.Name)
			} else {
				if _, err := azureClient.CreateOrUpdateBackupPolicyDisk(cmd.Context(), diskCfg); err != nil {
					applyErrors = append(applyErrors, fmt.Errorf("%s/%s: policy upsert failed: %w",
						rt.Target.ResourceGroup, rt.Target.ResourceName, err))
					continue
				}
			}

			remote, err := azureClient.GetBackupPolicyDisk(
				cmd.Context(),
				rt.Target.ResourceGroup,
				rt.Policy.Spec.Vault.Name,
				rt.Target.ResourceName,
			)
			if err != nil {
				applyErrors = append(applyErrors, fmt.Errorf("%s/%s: get remote failed: %w",
					rt.Target.ResourceGroup, rt.Target.ResourceName, err))
				continue
			}

			_, diff, _, _, err := engine.DiffDiskPolicy(rt, remote)
			if err != nil {
				applyErrors = append(applyErrors, fmt.Errorf("%s/%s: diff failed: %w",
					rt.Target.ResourceGroup, rt.Target.ResourceName, err))
				continue
			}

			if len(diff) == 0 {
				fmt.Println(green(fmt.Sprintf("✔ %s/%s — already in desired state",
					rt.Target.ResourceGroup, rt.Target.ResourceName)))
				continue
			}

			if dryRun {
				fmt.Printf("[dry-run] would apply disk policy %q to %s/%s\n",
					rt.Policy.Metadata.Name, rt.Target.ResourceGroup, rt.Target.ResourceName)
				continue
			}

			if err := azureClient.ApplyDisk(cmd.Context(), rt); err != nil {
				applyErrors = append(applyErrors, fmt.Errorf("%s/%s: apply failed: %w",
					rt.Target.ResourceGroup, rt.Target.ResourceName, err))
				fmt.Fprintln(os.Stderr, red(fmt.Sprintf("✖ %s/%s: %v",
					rt.Target.ResourceGroup, rt.Target.ResourceName, err)))
			} else {
				applied++
				fmt.Println(green(fmt.Sprintf("✔ Applied disk policy %q to %s/%s",
					rt.Policy.Metadata.Name, rt.Target.ResourceGroup, rt.Target.ResourceName)))
			}

		case "azure_sql", "azure_sql_vm":
			fmt.Fprintf(os.Stderr, "WARN: apply for %s not yet implemented, skipping %s/%s\n",
				rt.Target.Type, rt.Target.ResourceGroup, rt.Target.ResourceName)

		default:
			applyErrors = append(applyErrors, fmt.Errorf("%s/%s: unsupported type %q",
				rt.Target.ResourceGroup, rt.Target.ResourceName, rt.Target.Type))
		}
	}

	fmt.Printf("\nApplied %d change(s) with %d error(s)\n", applied, len(applyErrors))

	if len(applyErrors) > 0 {
		for i, err := range applyErrors {
			fmt.Printf("Error number: %d, %s\n", i, err)
		}
		return fmt.Errorf("apply completed with %d error(s)", len(applyErrors))
	}
	return nil
}
