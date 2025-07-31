package awsssolib

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache()

	// Test Put and Get
	key := "test-key"
	data := []byte("test-data")

	err := cache.Put(key, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved) != string(data) {
		t.Errorf("Expected %s, got %s", string(data), string(retrieved))
	}

	// Test Delete
	err = cache.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	retrieved, err = cache.Get(key)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected nil after delete")
	}
}

func TestFileCache(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cache := NewFileCache(tempDir)

	// Test Put and Get
	key := "test-key"
	data := []byte("test-data")

	err = cache.Put(key, data)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if string(retrieved) != string(data) {
		t.Errorf("Expected %s, got %s", string(data), string(retrieved))
	}

	// Test Delete
	err = cache.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	retrieved, err = cache.Get(key)
	if err != nil {
		t.Fatalf("Get after delete failed: %v", err)
	}
	if retrieved != nil {
		t.Error("Expected nil after delete")
	}
}

func TestTokenCaching(t *testing.T) {
	// Test SSO token caching (uses real file paths for AWS CLI compatibility)
	startURL := "https://test.awsapps.com/start"

	// Test token caching
	token := &Token{
		AccessToken: "test-access-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		StartURL:    startURL,
		Region:      "us-east-1",
	}

	// Clean up any existing token first
	DeleteCachedToken(nil, startURL)

	err := PutCachedToken(nil, startURL, token)
	if err != nil {
		t.Fatalf("PutCachedToken failed: %v", err)
	}

	retrieved, err := GetCachedToken(nil, startURL)
	if err != nil {
		t.Fatalf("GetCachedToken failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected token, got nil")
	}

	if retrieved.AccessToken != token.AccessToken {
		t.Errorf("Expected access token %s, got %s", token.AccessToken, retrieved.AccessToken)
	}

	// Clean up the previous token before testing expired token
	DeleteCachedToken(nil, startURL)

	// Test expired token (expired by more than 5-minute buffer)
	expiredToken := &Token{
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().UTC().Add(-10 * time.Minute), // Expired beyond buffer, ensure UTC
		StartURL:    startURL,
		Region:      "us-east-1",
	}

	err = PutCachedToken(nil, startURL, expiredToken)
	if err != nil {
		t.Fatalf("PutCachedToken failed: %v", err)
	}

	retrieved, err = GetCachedToken(nil, startURL)
	if err != nil {
		t.Fatalf("GetCachedToken failed: %v", err)
	}

	if retrieved != nil {
		t.Errorf("Expected nil for expired token, but got token with expiry: %s (current time: %s)", retrieved.ExpiresAt, time.Now())
	}

	// Clean up
	DeleteCachedToken(nil, startURL)
}

func TestAWSCLICompatibility(t *testing.T) {
	startURL := "https://test.awsapps.com/start"

	// Clean up first
	DeleteCachedToken(nil, startURL)

	// Test that our cache file path matches expected SHA1 format
	cachePath := GetSSOCacheFilePath(startURL)
	expectedHash := "bfe9e37c85cc299e34d8c03b631672483f78cd01" // SHA1 of the test URL
	if !strings.Contains(cachePath, expectedHash) {
		t.Errorf("Cache path %s doesn't contain expected hash %s", cachePath, expectedHash)
	}

	// Clean up
	DeleteCachedToken(nil, startURL)
}

func TestGenerateProfileName(t *testing.T) {
	account := &Account{
		AccountID:   "123456789012",
		AccountName: "Test Account",
	}
	role := &Role{
		RoleName: "Test_Role",
	}
	region := "us-west-2"

	tests := []struct {
		template string
		expected string
	}{
		{
			template: "",
			expected: "test-account.test-role.us-west-2",
		},
		{
			template: "{account_id}-{role_name}",
			expected: "123456789012-test-role",
		},
		{
			template: "{account_name}_{region}",
			expected: "test-account_us-west-2",
		},
	}

	for _, tt := range tests {
		result := GenerateProfileName(tt.template, account, role, region)
		if result != tt.expected {
			t.Errorf("Template %s: expected %s, got %s", tt.template, tt.expected, result)
		}
	}
}

func TestFormatAccountID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"123456789012", "123456789012"},
		{"1234-5678-9012", "123456789012"},
		{"1234 5678 9012", "123456789012"},
		{"123-456-789-012", "123456789012"},
	}

	for _, tt := range tests {
		result := formatAccountID(tt.input)
		if result != tt.expected {
			t.Errorf("Input %s: expected %s, got %s", tt.input, tt.expected, result)
		}
	}
}
