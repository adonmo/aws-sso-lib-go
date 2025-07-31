package awsssolib

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Default cache directories
var (
	DefaultSSOCacheDir = filepath.Join(os.Getenv("HOME"), ".aws", "sso", "cache")
	DefaultCLICacheDir = filepath.Join(os.Getenv("HOME"), ".aws", "cli", "cache")
)

// FileCache implements the Cache interface using the filesystem
type FileCache struct {
	directory string
}

// NewFileCache creates a new file-based cache
func NewFileCache(directory string) *FileCache {
	return &FileCache{
		directory: directory,
	}
}

// Get retrieves data from the cache
func (c *FileCache) Get(key string) ([]byte, error) {
	filename := c.getCacheFilename(key)
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

// Put stores data in the cache
func (c *FileCache) Put(key string, data []byte) error {
	if err := os.MkdirAll(c.directory, 0700); err != nil {
		return err
	}

	filename := c.getCacheFilename(key)
	return os.WriteFile(filename, data, 0600)
}

// Delete removes data from the cache
func (c *FileCache) Delete(key string) error {
	filename := c.getCacheFilename(key)
	err := os.Remove(filename)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// getCacheFilename generates a cache filename from a key (NOT used for SSO tokens)
func (c *FileCache) getCacheFilename(key string) string {
	// This is only used for non-SSO token caching
	// For SSO tokens, we use GetSSOCacheFilePath
	return filepath.Join(c.directory, key+".json")
}

// MemoryCache implements an in-memory cache
type MemoryCache struct {
	data map[string][]byte
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		data: make(map[string][]byte),
	}
}

// Get retrieves data from the cache
func (c *MemoryCache) Get(key string) ([]byte, error) {
	data, ok := c.data[key]
	if !ok {
		return nil, nil
	}
	return data, nil
}

// Put stores data in the cache
func (c *MemoryCache) Put(key string, data []byte) error {
	c.data[key] = data
	return nil
}

// Delete removes data from the cache
func (c *MemoryCache) Delete(key string) error {
	delete(c.data, key)
	return nil
}

// AWS CLI Compatible Token Format
// This matches the exact format used by AWS CLI and aws-sso-util
type AWSCLIToken struct {
	StartURL              string `json:"startUrl"`
	Region                string `json:"region"`
	AccessToken           string `json:"accessToken"`
	ExpiresAt             string `json:"expiresAt"`
	ReceivedAt            string `json:"receivedAt,omitempty"`
	ClientID              string `json:"clientId,omitempty"`
	ClientSecret          string `json:"clientSecret,omitempty"`
	RegistrationExpiresAt string `json:"registrationExpiresAt,omitempty"`
}

// GetSSOCacheFilePath returns the cache file path for the given start URL (AWS CLI compatible)
func GetSSOCacheFilePath(startURL string) string {
	// Use SHA1 hashing like AWS CLI and aws-sso-util for compatibility
	hash := sha1.Sum([]byte(startURL))
	filename := fmt.Sprintf("%x.json", hash)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fall back to HOME env var
		homeDir = os.Getenv("HOME")
	}

	cacheDir := filepath.Join(homeDir, ".aws", "sso", "cache")
	return filepath.Join(cacheDir, filename)
}

// Token cache helpers

// GetCachedToken retrieves a cached SSO token (AWS CLI compatible)
func GetCachedToken(cache Cache, startURL string) (*Token, error) {
	// Always use file system for SSO tokens to ensure AWS CLI compatibility
	cachePath := GetSSOCacheFilePath(startURL)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Try to parse as AWS CLI token format first
	var awsToken AWSCLIToken
	if err := json.Unmarshal(data, &awsToken); err != nil {
		// Fall back to our format
		var token Token
		if err := json.Unmarshal(data, &token); err != nil {
			return nil, err
		}

		// Check if token is expired
		if time.Now().After(token.ExpiresAt) {
			return nil, nil
		}

		return &token, nil
	}

	// Convert AWS CLI token to our format
	expiresAt, err := time.Parse("2006-01-02T15:04:05Z", awsToken.ExpiresAt)
	if err != nil {
		// Try RFC3339 format as fallback
		expiresAt, err = time.Parse(time.RFC3339, awsToken.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse token expiry: %w", err)
		}
	}

	// Check if token is expired (with 5-minute buffer)
	if time.Now().After(expiresAt.Add(-5 * time.Minute)) {
		return nil, nil
	}

	// Convert to our Token format
	token := &Token{
		AccessToken:  awsToken.AccessToken,
		ExpiresAt:    expiresAt,
		ClientID:     awsToken.ClientID,
		ClientSecret: awsToken.ClientSecret,
		Region:       awsToken.Region,
		StartURL:     awsToken.StartURL,
	}

	// Handle ReceivedAt if present
	if awsToken.ReceivedAt != "" {
		if registrationTime, err := time.Parse("2006-01-02T15:04:05Z", awsToken.ReceivedAt); err == nil {
			token.RegistrationTime = registrationTime
		}
	}

	return token, nil
}

// PutCachedToken stores an SSO token in the cache (AWS CLI compatible format)
func PutCachedToken(cache Cache, startURL string, token *Token) error {
	// Always use file system for SSO tokens to ensure AWS CLI compatibility
	cachePath := GetSSOCacheFilePath(startURL)

	// Ensure cache directory exists
	cacheDir := filepath.Dir(cachePath)
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create SSO cache directory: %w", err)
	}

	// Convert to AWS CLI format
	awsToken := AWSCLIToken{
		StartURL:     startURL,
		Region:       token.Region,
		AccessToken:  token.AccessToken,
		ExpiresAt:    token.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		ReceivedAt:   time.Now().Format("2006-01-02T15:04:05Z"),
		ClientID:     token.ClientID,
		ClientSecret: token.ClientSecret,
	}

	// Set registration expiry if we have client credentials
	if token.ClientID != "" && token.ClientSecret != "" {
		// Client registration typically expires in 90 days
		registrationExpiry := time.Now().Add(90 * 24 * time.Hour)
		awsToken.RegistrationExpiresAt = registrationExpiry.Format("2006-01-02T15:04:05Z")
	}

	// Marshal with indentation to match AWS CLI format
	data, err := json.MarshalIndent(awsToken, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Write with proper permissions
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cached token: %w", err)
	}

	return nil
}

// DeleteCachedToken removes an SSO token from the cache
func DeleteCachedToken(cache Cache, startURL string) error {
	cachePath := GetSSOCacheFilePath(startURL)
	err := os.Remove(cachePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// generateTokenCacheKey creates a cache key for an SSO token
// DEPRECATED: Use GetSSOCacheFilePath for AWS CLI compatibility
func generateTokenCacheKey(startURL string) string {
	return startURL
}

// Credential cache helpers

// CachedCredentials represents cached AWS credentials
type CachedCredentials struct {
	AccessKeyID     string    `json:"AccessKeyId"`
	SecretAccessKey string    `json:"SecretAccessKey"`
	SessionToken    string    `json:"SessionToken"`
	Expiration      time.Time `json:"Expiration"`
}

// GetCachedCredentials retrieves cached credentials
func GetCachedCredentials(cache Cache, cacheKey string) (*CachedCredentials, error) {
	if cache == nil {
		cache = NewMemoryCache()
	}

	data, err := cache.Get(cacheKey)
	if err != nil || data == nil {
		return nil, err
	}

	var creds CachedCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	// Check if credentials are expired
	if time.Now().After(creds.Expiration) {
		return nil, nil
	}

	return &creds, nil
}

// PutCachedCredentials stores credentials in the cache
func PutCachedCredentials(cache Cache, cacheKey string, creds *CachedCredentials) error {
	if cache == nil {
		cache = NewMemoryCache()
	}

	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	return cache.Put(cacheKey, data)
}

// generateCredentialCacheKey creates a cache key for credentials
func generateCredentialCacheKey(startURL, accountID, roleName string) string {
	return fmt.Sprintf("aws-sso-creds-%s-%s-%s", startURL, accountID, roleName)
}
