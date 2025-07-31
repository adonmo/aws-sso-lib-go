package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/adonmo/aws-sso-lib-go/awsssolib"
	"github.com/spf13/cobra"
)

// NewRunAsCommand creates the run-as command
func NewRunAsCommand() *cobra.Command {
	var accountID string
	var roleName string
	var region string
	var login bool

	cmd := &cobra.Command{
		Use:   "run-as -- <command> [args...]",
		Short: "Run a command with specific AWS credentials",
		Long: `Run a command with AWS credentials for a specific account and role.

This command sets up AWS environment variables for the specified account/role
combination and then executes the provided command.

Examples:
  # Run AWS CLI command
  aws-sso-util run-as --account 123456789012 --role MyRole -- aws s3 ls

  # Run with specific region
  aws-sso-util run-as --account 123456789012 --role MyRole --region us-west-2 -- aws ec2 describe-instances

  # Run any command that uses AWS credentials
  aws-sso-util run-as --account 123456789012 --role MyRole -- terraform plan`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Validate required flags
			if accountID == "" || roleName == "" {
				return fmt.Errorf("--account and --role are required")
			}

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

			// Default region if not specified
			if region == "" {
				region = os.Getenv("AWS_DEFAULT_REGION")
				if region == "" {
					region = "us-east-1"
				}
			}

			// Get AWS config
			cfg, err := awsssolib.GetAWSConfig(ctx, awsssolib.GetAWSConfigInput{
				StartURL:  startURL,
				SSORegion: ssoRegion,
				AccountID: accountID,
				RoleName:  roleName,
				Region:    region,
				Login:     login,
			})
			if err != nil {
				return fmt.Errorf("failed to get AWS config: %w", err)
			}

			// Get credentials
			creds, err := cfg.Credentials.Retrieve(ctx)
			if err != nil {
				return fmt.Errorf("failed to get credentials: %w", err)
			}

			// Set up environment
			env := os.Environ()
			env = setEnv(env, "AWS_ACCESS_KEY_ID", creds.AccessKeyID)
			env = setEnv(env, "AWS_SECRET_ACCESS_KEY", creds.SecretAccessKey)
			env = setEnv(env, "AWS_SESSION_TOKEN", creds.SessionToken)
			env = setEnv(env, "AWS_DEFAULT_REGION", region)
			env = setEnv(env, "AWS_REGION", region)

			// Execute command
			command := args[0]
			commandArgs := args[1:]

			execCmd := exec.Command(command, commandArgs...)
			execCmd.Env = env
			execCmd.Stdin = os.Stdin
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr

			err = execCmd.Run()
			if err != nil {
				// Try to get the exit code
				if exitErr, ok := err.(*exec.ExitError); ok {
					if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
						os.Exit(status.ExitStatus())
					}
				}
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&accountID, "account", "", "AWS account ID")
	cmd.Flags().StringVar(&roleName, "role", "", "SSO role name")
	cmd.Flags().StringVar(&region, "region", "", "AWS region")
	cmd.Flags().BoolVar(&login, "login", true, "Login if needed")

	return cmd
}

// setEnv sets or updates an environment variable in the env slice
func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if len(e) > len(prefix) && e[:len(prefix)] == prefix {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
