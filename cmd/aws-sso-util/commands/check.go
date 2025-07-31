package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/adonmo/aws-sso-lib-go/awsssolib"
	"github.com/spf13/cobra"
)

// NewCheckCommand creates the check command
func NewCheckCommand() *cobra.Command {
	var accountID string
	var roleName string

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check SSO configuration and access",
		Long: `Check SSO configuration and validate access to specific accounts/roles.

This command helps diagnose SSO configuration issues and verify access.

Examples:
  # Check SSO configuration
  aws-sso-util check

  # Check access to specific account
  aws-sso-util check --account 123456789012

  # Check access to specific role
  aws-sso-util check --account 123456789012 --role MyRole`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Get SSO configuration
			startURL, _ := cmd.Flags().GetString("start-url")
			ssoRegion, _ := cmd.Flags().GetString("sso-region")

			fmt.Fprintln(os.Stderr, "Checking SSO configuration...")

			// Try to find configuration
			var instance *awsssolib.SSOInstance
			if startURL == "" || ssoRegion == "" {
				var err error
				instance, err = awsssolib.FindInstance("")
				if err != nil {
					fmt.Fprintln(os.Stderr, "❌ No SSO configuration found")
					fmt.Fprintln(os.Stderr, "   Please provide --start-url and --sso-region or set AWS_DEFAULT_SSO_START_URL and AWS_DEFAULT_SSO_REGION")
					return err
				}
				if startURL == "" {
					startURL = instance.StartURL
				}
				if ssoRegion == "" {
					ssoRegion = instance.Region
				}
			}

			fmt.Fprintf(os.Stderr, "✓ SSO Start URL: %s\n", startURL)
			fmt.Fprintf(os.Stderr, "✓ SSO Region: %s\n", ssoRegion)
			if instance != nil {
				fmt.Fprintf(os.Stderr, "  (configured via %s)\n", instance.StartURLSource)
			}

			// Check cached token
			fmt.Fprintln(os.Stderr, "\nChecking authentication status...")
			token, err := awsssolib.GetCachedToken(nil, startURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error checking token: %v\n", err)
			} else if token == nil {
				fmt.Fprintln(os.Stderr, "❌ Not logged in")
				fmt.Fprintln(os.Stderr, "   Run: aws-sso-util login")
			} else {
				fmt.Fprintln(os.Stderr, "✓ Logged in")
				fmt.Fprintf(os.Stderr, "  Token expires: %s\n", token.ExpiresAt.Format("2006-01-02 15:04:05"))
			}

			// If logged in, check access
			if token != nil {
				fmt.Fprintln(os.Stderr, "\nChecking account access...")

				// List accounts
				accounts, err := awsssolib.ListAvailableAccounts(ctx, awsssolib.ListAccountsInput{
					StartURL:  startURL,
					SSORegion: ssoRegion,
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "❌ Failed to list accounts: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "✓ Access to %d accounts\n", len(accounts))

					// Check specific account if provided
					if accountID != "" {
						found := false
						for _, acc := range accounts {
							if acc.AccountID == accountID {
								found = true
								fmt.Fprintf(os.Stderr, "✓ Access to account %s (%s)\n", accountID, acc.AccountName)
								break
							}
						}
						if !found {
							fmt.Fprintf(os.Stderr, "❌ No access to account %s\n", accountID)
						}
					}
				}

				// Check roles if account specified
				if accountID != "" && roleName != "" {
					fmt.Fprintln(os.Stderr, "\nChecking role access...")

					roles, err := awsssolib.ListAvailableRoles(ctx, awsssolib.ListRolesInput{
						StartURL:   startURL,
						SSORegion:  ssoRegion,
						AccountIDs: []string{accountID},
					})
					if err != nil {
						fmt.Fprintf(os.Stderr, "❌ Failed to list roles: %v\n", err)
					} else {
						found := false
						for _, role := range roles {
							if role.RoleName == roleName {
								found = true
								fmt.Fprintf(os.Stderr, "✓ Access to role %s in account %s\n", roleName, accountID)
								break
							}
						}
						if !found {
							fmt.Fprintf(os.Stderr, "❌ No access to role %s in account %s\n", roleName, accountID)
						}
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&accountID, "account", "", "Check access to specific account")
	cmd.Flags().StringVar(&roleName, "role", "", "Check access to specific role (requires --account)")

	return cmd
}
