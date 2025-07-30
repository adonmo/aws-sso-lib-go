package awsssolib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// getCacheFilename generates a cache filename from a key
func (c *FileCache) getCacheFilename(key string) string {
	// Create a hash of the key for the filename
	hasher := sha256.New()
	hasher.Write([]byte(key))
	hash := hex.EncodeToString(hasher.Sum(nil))
	return filepath.Join(c.directory, hash+".json")
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

// Token cache helpers

// GetCachedToken retrieves a cached SSO token
func GetCachedToken(cache Cache, startURL string) (*Token, error) {
	if cache == nil {
		cache = NewFileCache(DefaultSSOCacheDir)
	}

	key := generateTokenCacheKey(startURL)
	data, err := cache.Get(key)
	if err != nil || data == nil {
		return nil, err
	}

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

// PutCachedToken stores an SSO token in the cache
func PutCachedToken(cache Cache, startURL string, token *Token) error {
	if cache == nil {
		cache = NewFileCache(DefaultSSOCacheDir)
	}

	key := generateTokenCacheKey(startURL)
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}

	return cache.Put(key, data)
}

// DeleteCachedToken removes an SSO token from the cache
func DeleteCachedToken(cache Cache, startURL string) error {
	if cache == nil {
		cache = NewFileCache(DefaultSSOCacheDir)
	}

	key := generateTokenCacheKey(startURL)
	return cache.Delete(key)
}

// generateTokenCacheKey creates a cache key for an SSO token
func generateTokenCacheKey(startURL string) string {
	// Normalize the URL
	url := strings.TrimRight(startURL, "/")
	return fmt.Sprintf("sso-token-%s", url)
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