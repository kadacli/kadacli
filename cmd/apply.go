package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
)

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

	cred, err := engine.Authenticate()
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

			_, diff, _, _, err := engine.DiffPolicy(rt, remote)
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

		case "azure_sql", "azure_sql_vm", "azure_disk":
			fmt.Fprintf(os.Stderr, "WARN: apply for %s not yet implemented, skipping %s/%s\n",
				rt.Target.Type, rt.Target.ResourceGroup, rt.Target.ResourceName)

		default:
			applyErrors = append(applyErrors, fmt.Errorf("%s/%s: unsupported type %q",
				rt.Target.ResourceGroup, rt.Target.ResourceName, rt.Target.Type))
		}
	}

	fmt.Printf("\nApplied %d change(s) with %d error(s)\n", applied, len(applyErrors))

	if len(applyErrors) > 0 {
		return fmt.Errorf("apply completed with %d error(s)", len(applyErrors))
	}
	return nil
}
