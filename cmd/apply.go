package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dbjorklund/backup-policy-as-code/internal/engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/engine/azure_engine"
	"github.com/dbjorklund/backup-policy-as-code/internal/models"
	"github.com/dbjorklund/backup-policy-as-code/internal/models/azure"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply backup policies idempotently",
	Long: `Validates policies, diffs desired state against live environment, and applies
only the resources that are out of desired state. Safe to run repeatedly.`,
	RunE: runApply,
}

func runApply(cmd *cobra.Command, args []string) error {
	fmt.Println(colorKadaBold("──────────────── kåda — backup policy as code ────────────────"))
	dir := viper.GetString("dir")

	var azureTargets []models.ResolvedTarget
	var awsTargets []models.ResolvedTarget
	var errs []error

	resolved, err := loadAndValidate(dir)
	if err != nil {
		fmt.Println(err.Error())
		return fmt.Errorf("Exit Code: 2 - Error")
	}

	for _, rt := range resolved {
		switch rt.Provider {
		case "azure":
			azureTargets = append(azureTargets, rt)
		case "aws":
			awsTargets = append(awsTargets, rt)
		default:
			errs = append(errs, fmt.Errorf("Provider %q is not yet supported", rt.Provider))
		}
	}

	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, colorRed("✖ "+err.Error()))
		}
	}

	azureApplied, azureApplyErrs := applyAzure(cmd, resolved)
	// apply aws
	// sum azureApplied with awsApplied and so on
	errs = append(errs, azureApplyErrs...)

	fmt.Println(colorKadaBold(fmt.Sprintf("\nApplied %d change(s) with %d error(s)", azureApplied, len(errs))))
	if len(errs) > 0 {
		fmt.Printf("Apply completed with %d error(s)\n", len(errs))
		return fmt.Errorf("Exit Code: 2 - Error")
	}

	return nil
}

func applyAzure(cmd *cobra.Command, targets []models.ResolvedTarget) (int, []error) {
	var errs []error
	applied := 0

	credAny, err := engine.Authenticate("azure")
	if err != nil {
		return 0, []error{fmt.Errorf("Authentication failed: %w", err)}
	}

	cred, ok := credAny.(*azidentity.DefaultAzureCredential)
	if !ok {
		return 0, []error{fmt.Errorf("Authentication failed: %w", err)}
	}

	for _, rt := range targets {
		var n int
		var messages []string
		var err error
		var resourceGroup, resourceName, resourceType string

		switch payload := rt.Payload.(type) {
		case azure.ResolvedVM:
			resourceGroup, resourceName, resourceType = payload.ResourceGroup, payload.ResourceName, payload.Resource.Type
			n, messages, err = azure_engine.ApplyVM(cmd.Context(), cred, payload)
		case azure.ResolvedDisk:
			resourceGroup, resourceName, resourceType = payload.ResourceGroup, payload.ResourceName, payload.Resource.Type
			n, messages, err = azure_engine.ApplyDisk(cmd.Context(), cred, payload)
		case azure.ResolvedSqlDb:
			resourceGroup, resourceName, resourceType = payload.ResourceGroup, payload.ResourceName, payload.Resource.Type
			n, messages, err = azure_engine.ApplySqlDb(cmd.Context(), cred, payload)
		default:
			errs = append(errs, fmt.Errorf("Provider %q is not yet supported", rt.Provider))
			continue
		}

		fmt.Println(providerColor(rt.Provider)(fmt.Sprintf("\n──── %s/%s (%s) ────", resourceGroup, resourceName, resourceType)))
		for _, msg := range messages {
			fmt.Println("  " + colorizeApplyMessage(msg))
		}
		applied += n
		if err != nil {
			errs = append(errs, err)
			fmt.Fprintln(os.Stderr, colorRed("✖ "+err.Error()))
		}
	}

	return applied, errs
}

// colorizeApplyMessage colors a message returned from an Apply* function based
// on its content: "✔ " prefix means a change was made
func colorizeApplyMessage(msg string) string {
	switch {
	case strings.HasPrefix(msg, "✔"):
		return colorGreen(msg)
	default:
		return colorYellow(msg)
	}
}
