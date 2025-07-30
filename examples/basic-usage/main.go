package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/adonmo/aws-sso-lib-go/awsssolib"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func main() {
	// Example configuration - replace with your values
	startURL := os.Getenv("AWS_DEFAULT_SSO_START_URL")
	ssoRegion := os.Getenv("AWS_DEFAULT_SSO_REGION")
	
	if startURL == "" || ssoRegion == "" {
		log.Fatal("Please set AWS_DEFAULT_SSO_START_URL and AWS_DEFAULT_SSO_REGION environment variables")
	}

	ctx := context.Background()

	// Example 1: Login to SSO
	fmt.Println("=== Example 1: Login to SSO ===")
	loginExample(ctx, startURL, ssoRegion)

	// Example 2: List available accounts and roles
	fmt.Println("\n=== Example 2: List Available Accounts and Roles ===")
	listAccessExample(ctx, startURL, ssoRegion)

	// Example 3: Get AWS SDK config for specific account/role
	fmt.Println("\n=== Example 3: Use AWS SDK with SSO Credentials ===")
	// You'll need to replace these with actual values from your SSO
	// accountID := "123456789012"
	// roleName := "MyRole"
	// awsSDKExample(ctx, startURL, ssoRegion, accountID, roleName)
}

func loginExample(ctx context.Context, startURL, ssoRegion string) {
	output, err := awsssolib.Login(ctx, awsssolib.LoginInput{
		StartURL:  startURL,
		SSORegion: ssoRegion,
	})
	if err != nil {
		log.Printf("Login failed: %v", err)
		return
	}

	fmt.Printf("Successfully logged in!\n")
	fmt.Printf("Token expires at: %s\n", output.ExpiresAt.Format("2006-01-02 15:04:05"))
}

func listAccessExample(ctx context.Context, startURL, ssoRegion string) {
	// List accounts
	accounts, err := awsssolib.ListAvailableAccounts(ctx, awsssolib.ListAccountsInput{
		StartURL:  startURL,
		SSORegion: ssoRegion,
		Login:     true,
	})
	if err != nil {
		log.Printf("Failed to list accounts: %v", err)
		return
	}

	fmt.Printf("\nAvailable Accounts:\n")
	for _, account := range accounts {
		fmt.Printf("  - %s (%s)\n", account.AccountName, account.AccountID)
	}

	// List first 5 roles
	roles, err := awsssolib.ListAvailableRoles(ctx, awsssolib.ListRolesInput{
		StartURL:  startURL,
		SSORegion: ssoRegion,
	})
	if err != nil {
		log.Printf("Failed to list roles: %v", err)
		return
	}

	fmt.Printf("\nAvailable Roles (first 5):\n")
	count := 0
	for _, role := range roles {
		fmt.Printf("  - %s in %s (%s)\n", role.RoleName, role.AccountName, role.AccountID)
		count++
		if count >= 5 {
			fmt.Printf("  ... and more\n")
			break
		}
	}
}

func awsSDKExample(ctx context.Context, startURL, ssoRegion, accountID, roleName string) {
	// Get AWS config for specific account/role
	cfg, err := awsssolib.GetAWSConfig(ctx, awsssolib.GetAWSConfigInput{
		StartURL:  startURL,
		SSORegion: ssoRegion,
		AccountID: accountID,
		RoleName:  roleName,
		Region:    "us-east-1",
		Login:     true,
	})
	if err != nil {
		log.Printf("Failed to get AWS config: %v", err)
		return
	}

	// Example: Get caller identity
	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Printf("Failed to get caller identity: %v", err)
		return
	}

	fmt.Printf("\nCaller Identity:\n")
	fmt.Printf("  Account: %s\n", *identity.Account)
	fmt.Printf("  ARN: %s\n", *identity.Arn)
	fmt.Printf("  UserID: %s\n", *identity.UserId)

	// Example: List S3 buckets
	s3Client := s3.NewFromConfig(cfg)
	buckets, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		log.Printf("Failed to list buckets: %v", err)
		return
	}

	fmt.Printf("\nS3 Buckets:\n")
	for _, bucket := range buckets.Buckets {
		fmt.Printf("  - %s\n", *bucket.Name)
	}
}