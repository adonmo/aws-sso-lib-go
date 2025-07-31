package awsssolib

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// SSOInstance represents an AWS SSO instance configuration
type SSOInstance struct {
	StartURL       string
	Region         string
	StartURLSource string
	RegionSource   string
}

// Token represents an SSO access token
type Token struct {
	AccessToken      string    `json:"accessToken"`
	ExpiresAt        time.Time `json:"expiresAt"`
	RefreshToken     string    `json:"refreshToken,omitempty"`
	ClientID         string    `json:"clientId,omitempty"`
	ClientSecret     string    `json:"clientSecret,omitempty"`
	RegistrationTime time.Time `json:"registrationTime,omitempty"`
	Region           string    `json:"region,omitempty"`
	StartURL         string    `json:"startUrl,omitempty"`
}

// Account represents an AWS account accessible through SSO
type Account struct {
	AccountID    string
	AccountName  string
	EmailAddress string
}

// Role represents a role within an AWS account
type Role struct {
	RoleName    string
	AccountID   string
	AccountName string
}

// GetAWSConfigInput contains parameters for getting AWS SDK config
type GetAWSConfigInput struct {
	StartURL  string
	SSORegion string
	AccountID string
	RoleName  string
	Region    string
	Login     bool
	// Optional caches
	SSOCache        Cache
	CredentialCache Cache
}

// LoginInput contains parameters for SSO login
type LoginInput struct {
	StartURL       string
	SSORegion      string
	ForceRefresh   bool
	ExpiryWindow   time.Duration
	DisableBrowser bool
	Message        string
	// Optional auth handler for custom auth flow
	UserAuthHandler AuthHandler
	// Optional cache
	SSOCache Cache
}

// LoginOutput contains the result of SSO login
type LoginOutput struct {
	Token     *Token
	ExpiresAt time.Time
}

// ListAccountsInput contains parameters for listing accounts
type ListAccountsInput struct {
	StartURL  string
	SSORegion string
	Login     bool
	// Optional cache
	SSOCache Cache
}

// ListRolesInput contains parameters for listing roles
type ListRolesInput struct {
	StartURL   string
	SSORegion  string
	AccountIDs []string // Optional: filter by account IDs
	Login      bool
	// Optional cache
	SSOCache Cache
}

// Cache defines the interface for caching tokens and credentials
type Cache interface {
	Get(key string) ([]byte, error)
	Put(key string, data []byte) error
	Delete(key string) error
}

// AuthHandler is called during the authentication flow
type AuthHandler func(ctx context.Context, params AuthHandlerParams) error

// AuthHandlerParams contains parameters passed to the auth handler
type AuthHandlerParams struct {
	VerificationURI         string
	UserCode                string
	VerificationURIComplete string
	ExpiresAt               time.Time
}

// CredentialProvider provides AWS credentials
type CredentialProvider interface {
	Retrieve(ctx context.Context) (aws.Credentials, error)
}

// Error types
type AuthenticationNeededError struct {
	Message string
}

func (e AuthenticationNeededError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "authentication needed"
}

type InvalidConfigError struct {
	Message string
}

func (e InvalidConfigError) Error() string {
	return "invalid configuration: " + e.Message
}
