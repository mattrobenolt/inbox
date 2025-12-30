package cache

import (
	json "encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.withmatt.com/inbox/internal/config"
	"go.withmatt.com/inbox/internal/gmail"
)

// ThreadCache represents the cached thread data
type ThreadCache struct {
	Threads   []gmail.Thread `json:"threads"`
	Timestamp time.Time      `json:"timestamp"`
}

// InboxCache stores inbox thread IDs for an account.
type InboxCache struct {
	ThreadIDs     []string  `json:"thread_ids"`
	NextPageToken string    `json:"next_page_token,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// ThreadMetadataCache stores metadata for a single thread.
type ThreadMetadataCache struct {
	Thread    gmail.Thread `json:"thread"`
	Timestamp time.Time    `json:"timestamp"`
}

func normalizeAccountKey(email string) string {
	if email == "" {
		return "unknown"
	}
	key := strings.ToLower(email)
	replacer := strings.NewReplacer(
		"@", "_at_",
		".", "_dot_",
		"+", "_plus_",
		":", "_",
		"/", "_",
		"\\", "_",
	)
	return replacer.Replace(key)
}

// LoadThreads loads cached threads from disk
func LoadThreads() ([]gmail.Thread, error) {
	path, err := config.CachePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No cache exists yet, that's fine
			return nil, nil
		}
		return nil, err
	}

	var cache ThreadCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return cache.Threads, nil
}

// SaveThreads saves threads to disk cache
func SaveThreads(threads []gmail.Thread) error {
	path, err := config.CachePath()
	if err != nil {
		return err
	}

	cache := ThreadCache{
		Threads:   threads,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func inboxCachePath(accountKey string) (string, error) {
	cacheDir, err := config.CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "inbox_"+accountKey+".json"), nil
}

func threadMetadataCachePath(accountKey string, threadID string) (string, error) {
	cacheDir, err := config.CacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cacheDir, "thread_metadata", accountKey)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	return filepath.Join(dir, threadID+".json"), nil
}

// LoadInbox loads cached inbox thread IDs for an account.
func LoadInbox(accountEmail string, legacyIndex int) ([]string, string, error) {
	_ = legacyIndex
	accountKey := normalizeAccountKey(accountEmail)
	path, err := inboxCachePath(accountKey)
	if err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", nil
		}
		return nil, "", err
	}

	var cache InboxCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, "", err
	}

	return cache.ThreadIDs, cache.NextPageToken, nil
}

// SaveInbox saves inbox thread IDs for an account.
func SaveInbox(accountEmail string, threadIDs []string, nextPageToken string) error {
	accountKey := normalizeAccountKey(accountEmail)
	path, err := inboxCachePath(accountKey)
	if err != nil {
		return err
	}

	cache := InboxCache{
		ThreadIDs:     threadIDs,
		NextPageToken: nextPageToken,
		Timestamp:     time.Now(),
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// LoadThreadMetadata loads cached thread metadata.
func LoadThreadMetadata(
	accountEmail string,
	legacyIndex int,
	threadID string,
) (*gmail.Thread, error) {
	_ = legacyIndex
	accountKey := normalizeAccountKey(accountEmail)
	path, err := threadMetadataCachePath(accountKey, threadID)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cache ThreadMetadataCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	cache.Thread.Loaded = true
	return &cache.Thread, nil
}

// SaveThreadMetadata saves cached thread metadata.
func SaveThreadMetadata(accountEmail string, thread gmail.Thread) error {
	accountKey := normalizeAccountKey(accountEmail)
	path, err := threadMetadataCachePath(accountKey, thread.ThreadID)
	if err != nil {
		return err
	}

	cache := ThreadMetadataCache{
		Thread:    thread,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}
