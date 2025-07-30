package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/adonmo/aws-sso-lib-go/awsssolib"
	"github.com/spf13/cobra"
)

// NewLoginCommand creates the login command
func NewLoginCommand() *cobra.Command {
	var forceRefresh bool
	var disableBrowser bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to AWS SSO",
		Long: `Log in to AWS SSO interactively.

This command will open your browser to complete the SSO login process.
Once logged in, the token will be cached for future use.

Examples:
  # Login using environment variables or config
  aws-sso-util login

  # Login with specific SSO instance
  aws-sso-util login --start-url https://my-sso.awsapps.com/start --sso-region us-east-1

  # Force re-authentication
  aws-sso-util login --force-refresh`,
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

			// Perform login
			fmt.Fprintf(os.Stderr, "Logging in to %s...\n", startURL)

			output, err := awsssolib.Login(ctx, awsssolib.LoginInput{
				StartURL:       startURL,
				SSORegion:      ssoRegion,
				ForceRefresh:   forceRefresh,
				DisableBrowser: disableBrowser,
			})
			if err != nil {
				return fmt.Errorf("login failed: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Successfully logged in!\n")
			fmt.Fprintf(os.Stderr, "Token expires at: %s\n", output.ExpiresAt.Format("2006-01-02 15:04:05"))

			return nil
		},
	}

	cmd.Flags().BoolVar(&forceRefresh, "force-refresh", false, "Force re-authentication even if valid token exists")
	cmd.Flags().BoolVar(&disableBrowser, "disable-browser", false, "Disable automatic browser opening")

	return cmd
}

// NewLogoutCommand creates the logout command
func NewLogoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out from AWS SSO",
		Long: `Log out from AWS SSO and remove cached credentials.

Examples:
  # Logout using environment variables or config
  aws-sso-util logout

  # Logout from specific SSO instance
  aws-sso-util logout --start-url https://my-sso.awsapps.com/start --sso-region us-east-1`,
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

			// Perform logout
			fmt.Fprintf(os.Stderr, "Logging out from %s...\n", startURL)

			err := awsssolib.Logout(ctx, startURL, ssoRegion, nil)
			if err != nil {
				return fmt.Errorf("logout failed: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Successfully logged out!\n")

			return nil
		},
	}

	return cmd
}