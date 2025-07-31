package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewConsoleCommand creates the console command group
func NewConsoleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "console",
		Short: "Commands for launching AWS Console in browser",
		Long: `Commands for launching the AWS Management Console in a browser.

This feature is in beta and provides functionality to open the AWS Console
for specific accounts and roles.`,
	}

	cmd.AddCommand(newConsoleLaunchCommand())

	return cmd
}

// newConsoleLaunchCommand creates the console launch command
func newConsoleLaunchCommand() *cobra.Command {
	var accountID string
	var roleName string
	var service string

	cmd := &cobra.Command{
		Use:   "launch",
		Short: "Launch AWS Console in browser",
		Long: `Launch the AWS Management Console in your browser for a specific account and role.

Examples:
  # Open console for specific account/role
  aws-sso-util console launch --account 123456789012 --role MyRole

  # Open specific service console
  aws-sso-util console launch --account 123456789012 --role MyRole --service ec2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement console launch functionality
			return fmt.Errorf("console launch functionality not yet implemented")
		},
	}

	cmd.Flags().StringVar(&accountID, "account", "", "AWS account ID")
	cmd.Flags().StringVar(&roleName, "role", "", "SSO role name")
	cmd.Flags().StringVar(&service, "service", "", "AWS service to open (e.g., ec2, s3)")

	return cmd
}
