package links

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adrg/xdg"
	_ "modernc.org/sqlite"
)

const cacheFileName = "link_cache.sqlite"

type cacheEntry struct {
	Resolved  string    `json:"resolved"`
	NoChange  bool      `json:"no_change"`
	CheckedAt time.Time `json:"checked_at"`
}

type Cache struct {
	db *sql.DB
	mu sync.Mutex
}

type DomainEntry struct {
	Domain        string
	RedirectCount int
	LastSeen      time.Time
}

func LoadCache() (*Cache, error) {
	path, err := xdg.CacheFile(filepath.Join("inbox", cacheFileName))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if _, err := db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS link_cache (
			url TEXT PRIMARY KEY,
			resolved TEXT NOT NULL,
			no_change INTEGER NOT NULL,
			checked_at INTEGER NOT NULL
		)
	`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS link_domains (
			domain TEXT PRIMARY KEY,
			redirect_count INTEGER NOT NULL,
			last_seen INTEGER NOT NULL
		)
	`); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Cache{db: db}, nil
}

func (c *Cache) Get(rawURL string) (cacheEntry, bool) {
	if c == nil || c.db == nil {
		return cacheEntry{}, false
	}

	var entry cacheEntry
	var noChange int
	var checkedAt int64
	err := c.db.QueryRowContext(
		context.Background(),
		`SELECT resolved, no_change, checked_at FROM link_cache WHERE url = ?`,
		rawURL,
	).Scan(&entry.Resolved, &noChange, &checkedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return cacheEntry{}, false
		}
		return cacheEntry{}, false
	}

	entry.NoChange = noChange != 0
	entry.CheckedAt = time.Unix(checkedAt, 0).UTC()
	return entry, true
}

func (c *Cache) SetMany(entries map[string]cacheEntry) error {
	if c == nil || c.db == nil || len(entries) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	ctx := context.Background()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO link_cache (url, resolved, no_change, checked_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			resolved = excluded.resolved,
			no_change = excluded.no_change,
			checked_at = excluded.checked_at
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for rawURL, entry := range entries {
		checkedAt := entry.CheckedAt
		if checkedAt.IsZero() {
			checkedAt = time.Now().UTC()
		}
		noChange := 0
		if entry.NoChange {
			noChange = 1
		}
		if _, err := stmt.ExecContext(ctx, rawURL, entry.Resolved, noChange, checkedAt.Unix()); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (c *Cache) ListDomains() ([]string, error) {
	entries, err := c.ListDomainEntries()
	if err != nil {
		return nil, err
	}
	domains := make([]string, 0, len(entries))
	for _, entry := range entries {
		domains = append(domains, entry.Domain)
	}
	return domains, nil
}

func (c *Cache) ListDomainEntries() ([]DomainEntry, error) {
	if c == nil || c.db == nil {
		return nil, nil
	}

	rows, err := c.db.QueryContext(
		context.Background(),
		`SELECT domain, redirect_count, last_seen FROM link_domains
			ORDER BY redirect_count DESC, domain ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []DomainEntry
	for rows.Next() {
		var entry DomainEntry
		var lastSeen int64
		if err := rows.Scan(&entry.Domain, &entry.RedirectCount, &lastSeen); err != nil {
			return nil, err
		}
		entry.Domain = normalizeDomain(entry.Domain)
		if entry.Domain == "" {
			continue
		}
		entry.LastSeen = time.Unix(lastSeen, 0).UTC()
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (c *Cache) RecordDomains(domains []string) error {
	if c == nil || c.db == nil || len(domains) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(domains))
	normalized := make([]string, 0, len(domains))
	for _, domain := range domains {
		domain = normalizeDomain(domain)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		normalized = append(normalized, domain)
	}
	if len(normalized) == 0 {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	ctx := context.Background()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO link_domains (domain, redirect_count, last_seen)
		VALUES (?, 1, ?)
		ON CONFLICT(domain) DO UPDATE SET
			redirect_count = redirect_count + 1,
			last_seen = excluded.last_seen
	`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC().Unix()
	for _, domain := range normalized {
		if _, err := stmt.ExecContext(ctx, domain, now); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
