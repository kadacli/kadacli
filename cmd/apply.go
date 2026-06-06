package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	azure_external "github.com/dbjorklund/backup-policy-as-code/external/azure"
	"github.com/dbjorklund/backup-policy-as-code/internal/engine/azure_engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply backup policies idempotently to Azure",
	Long: `Validates policies, diffs desired state against live Azure, and applies
only the resources that are out of desired state. Safe to run repeatedly.`,
	RunE: runApply,
}

func runApply(cmd *cobra.Command, args []string) error {
	dir := viper.GetString("dir")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	resolved, err := loadAndValidate(dir)
	if err != nil {
		return err
	}

	// Authenticate once; a per-target Client is constructed using the same credential.
	cred, err := azure_external.Authenticate()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	var applyErrors []error
	applied := 0

	for _, rt := range resolved {
		switch rt.Provider {
		case "azure":
			n, err := applyAzureTarget(cmd.Context(), cred, rt, dryRun)
			applied += n
			if err != nil {
				applyErrors = append(applyErrors, err)
				fmt.Fprintln(os.Stderr, colorRed("✖ "+err.Error()))
			}
		default:
			applyErrors = append(applyErrors, fmt.Errorf("provider %q is not yet supported", rt.Provider))
		}
	}

	fmt.Printf("\nApplied %d change(s) with %d error(s)\n", applied, len(applyErrors))
	if len(applyErrors) > 0 {
		return fmt.Errorf("apply completed with %d error(s)", len(applyErrors))
	}
	return nil
}

func applyAzureTarget(ctx context.Context, cred *azidentity.DefaultAzureCredential, rt models.ResolvedTarget, dryRun bool) (int, error) {
	vm, ok := rt.Payload.(azure.ResolvedVM)
	if !ok {
		return 0, fmt.Errorf("unexpected payload type %T for azure provider", rt.Payload)
	}

	client := &azure_external.Client{
		Credential:     cred,
		SubscriptionID: vm.SubscriptionID,
	}

	switch vm.Resource.Type {
	case "azure_vm":
		return applyAzureVM(ctx, client, vm, dryRun)
	case "azure_disk":
		// TODO: enable once disk.go.bak is updated to the new model
		return 0, fmt.Errorf("%s/%s: azure_disk apply is not yet implemented",
			vm.ResourceGroup, vm.ResourceName)
	case "azure_sql_vm":
		// TODO: enable once sqlvm.go.bak is implemented
		return 0, fmt.Errorf("%s/%s: azure_sql_vm apply is not yet implemented",
			vm.ResourceGroup, vm.ResourceName)
	default:
		return 0, fmt.Errorf("%s/%s: unsupported resource type %q",
			vm.ResourceGroup, vm.ResourceName, vm.Resource.Type)
	}
}

func applyAzureVM(ctx context.Context, client *azure_external.Client, vm azure.ResolvedVM, dryRun bool) (int, error) {
	remote, err := client.GetBackupPolicyVM(ctx, vm.ResourceGroup, vm.VaultName, vm.ResourceName)
	if err != nil {
		return 0, fmt.Errorf("%s/%s: failed to get remote state: %w",
			vm.ResourceGroup, vm.ResourceName, err)
	}

	_, diff, _, _, err := azure_engine.DiffVMPolicy(vm, remote)
	if err != nil {
		return 0, fmt.Errorf("%s/%s: diff failed: %w",
			vm.ResourceGroup, vm.ResourceName, err)
	}

	if len(diff) == 0 {
		fmt.Println(colorGreen(fmt.Sprintf("✔ %s/%s — already in desired state",
			vm.ResourceGroup, vm.ResourceName)))
		return 0, nil
	}

	if dryRun {
		fmt.Printf("[dry-run] would apply policy %q to %s/%s\n",
			vm.PolicyName, vm.ResourceGroup, vm.ResourceName)
		return 0, nil
	}

	// TODO: upsert the policy definition in the vault before assigning (executor.go)
	if err := client.ApplyVM(ctx, vm); err != nil {
		return 0, fmt.Errorf("%s/%s: apply failed: %w",
			vm.ResourceGroup, vm.ResourceName, err)
	}

	fmt.Println(colorGreen(fmt.Sprintf("✔ Applied policy %q to %s/%s",
		vm.PolicyName, vm.ResourceGroup, vm.ResourceName)))
	return 1, nil
}
