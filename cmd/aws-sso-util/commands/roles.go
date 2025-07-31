package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/adonmo/aws-sso-lib-go/awsssolib"
	"github.com/spf13/cobra"
)

// NewRolesCommand creates the roles command
func NewRolesCommand() *cobra.Command {
	var accountIDs []string
	var login bool
	var format string

	cmd := &cobra.Command{
		Use:   "roles",
		Short: "List available AWS SSO roles",
		Long: `List all roles available through AWS SSO.

This command shows all the accounts and roles you have access to through SSO.

Examples:
  # List all available roles
  aws-sso-util roles

  # List roles for specific accounts
  aws-sso-util roles --account 123456789012 --account 234567890123

  # List roles and login if needed
  aws-sso-util roles --login

  # Output in different formats
  aws-sso-util roles --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Get SSO configuration
			startURL, _ := cmd.Flags().GetString("start-url")
			ssoRegion, _ := cmd.Flags().GetString("sso-region")

			// Try to find configuration if not provided
			if startURL == "" || ssoRegion == "" {
				instance, err := awsssolib.FindInstance("")
				if err != nil {
					return fmt.Errorf("no SSO configuration found. Please provide --start-url and --sso-region or set AWS_DEFAULT_SSO_START_URL and AWS_DEFAULT_SSO_REGION")
				}
				if startURL == "" {
					startURL = instance.StartURL
				}
				if ssoRegion == "" {
					ssoRegion = instance.Region
				}
			}

			// List roles
			roles, err := awsssolib.ListAvailableRoles(ctx, awsssolib.ListRolesInput{
				StartURL:   startURL,
				SSORegion:  ssoRegion,
				AccountIDs: accountIDs,
				Login:      login,
			})
			if err != nil {
				return fmt.Errorf("failed to list roles: %w", err)
			}

			// Output results
			switch format {
			case "json":
				// TODO: Implement JSON output
				return fmt.Errorf("JSON output not yet implemented")
			default:
				// Table output
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "ACCOUNT ID\tACCOUNT NAME\tROLE NAME")
				fmt.Fprintln(w, "----------\t------------\t---------")

				for _, role := range roles {
					fmt.Fprintf(w, "%s\t%s\t%s\n", role.AccountID, role.AccountName, role.RoleName)
				}

				w.Flush()
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&accountIDs, "account", []string{}, "Filter by account ID (can be specified multiple times)")
	cmd.Flags().BoolVar(&login, "login", false, "Login if needed")
	cmd.Flags().StringVar(&format, "format", "table", "Output format (table, json)")

	return cmd
}
