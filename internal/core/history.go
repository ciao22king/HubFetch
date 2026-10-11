package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// MaxHistoryEntries bounds the persisted history; the oldest entries fall off.
const MaxHistoryEntries = 100

// Entry is one saved fetch: where it came from and where it lives on disk.
type Entry struct {
	Repository
	Target    string    `json:"target"`
	FetchedAt time.Time `json:"fetched_at"`
}

// Label is the display name of an entry, falling back to the folder name for
// rows written by older versions without a parsed repository.
func (e Entry) Label() string {
	if e.Name != "" {
		return e.Name
	}
	return filepath.Base(strings.TrimRight(e.Target, string(os.PathSeparator)))
}

// RelativeTime formats t as a short "ago" string for the history rows.
func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	delta := time.Since(t)
	switch {
	case delta < time.Minute:
		return "adesso"
	case delta < time.Hour:
		return fmt.Sprintf("%dm fa", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh fa", int(delta.Hours()))
	case delta < 30*24*time.Hour:
		return fmt.Sprintf("%dg fa", int(delta.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

// Store is a JSON file holding the fetch history. Reads are served from an
// in-memory cache so navigating the list never touches the disk; writes go
// through atomically and refresh the cache. The zero value is not usable —
// call NewStore.
type Store struct {
	path     string
	mu       sync.Mutex
	cache    []Entry
	cachedAt time.Time
}

// NewStore points a store at dir/history.json (creating nothing yet).
func NewStore(dir string) *Store {
	return &Store{path: filepath.Join(dir, "history.json")}
}

// DefaultStoreDir is ~/.config/gofetch (%AppData%\gofetch on Windows), shared
// with the config file so everything lives next to each other.
func DefaultStoreDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "gofetch"), nil
}

// Load returns the history, using the cached copy when the file has not
// changed since it was read. Missing or corrupt files degrade to an empty
// history instead of failing: GoFetch must always start.
func (s *Store) Load() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	stat, err := os.Stat(s.path)
	if err == nil && s.cache != nil && stat.ModTime().Equal(s.cachedAt) {
		return s.cache
	}

	var entries []Entry
	if data, err := os.ReadFile(s.path); err == nil {
		_ = json.Unmarshal(data, &entries) // invalid JSON degrades to empty
	}
	s.cache, s.cachedAt = entries, time.Now()
	return s.cache
}

// Save replaces the history atomically: the data goes to a temp file in the
// same directory and is renamed over the target, so a crash or a full disk can
// never leave a truncated history.json behind.
func (s *Store) Save(entries []Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(s.path)
	if err := writeAtomic(dir, "history-*.tmp", s.path, data); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = entries
	if stat, statErr := os.Stat(s.path); statErr == nil {
		s.cachedAt = stat.ModTime()
	}
	return nil
}

// Upsert moves (or adds) the entry matching target to the front with a fresh
// timestamp, keeping the newest-first order and the size bound.
func Upsert(entries []Entry, entry Entry) []Entry {
	filtered := make([]Entry, 0, len(entries)+1)
	for _, existing := range entries {
		if !SameTarget(existing.Target, entry.Target) {
			filtered = append(filtered, existing)
		}
	}
	entry.FetchedAt = time.Now()
	return Bound(append([]Entry{entry}, filtered...))
}

// Remove drops every entry pointing at target.
func Remove(entries []Entry, target string) []Entry {
	filtered := make([]Entry, 0, len(entries))
	for _, existing := range entries {
		if !SameTarget(existing.Target, target) {
			filtered = append(filtered, existing)
		}
	}
	return filtered
}

// Bound enforces the history size limit.
func Bound(entries []Entry) []Entry {
	if len(entries) > MaxHistoryEntries {
		return entries[:MaxHistoryEntries]
	}
	return entries
}

// SameTarget compares two paths after cleaning and home expansion, case-folded
// on Windows where the filesystem is case-insensitive.
func SameTarget(a, b string) bool {
	cleanA := filepath.Clean(ExpandHome(a))
	cleanB := filepath.Clean(ExpandHome(b))
	if runtime.GOOS == "windows" {
		return strings.EqualFold(cleanA, cleanB)
	}
	return cleanA == cleanB
}

// Search returns the indexes of the entries matching query, preserving the
// newest-first order. An empty query matches everything. Matching is done per
// field instead of joining one big haystack string, which keeps it allocation
// free for the common no-match case.
func Search(entries []Entry, query string) []int {
	query = strings.ToLower(strings.TrimSpace(query))
	indexes := make([]int, 0, len(entries))
	for index, entry := range entries {
		if query == "" || matches(entry, query) {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

func matches(entry Entry, query string) bool {
	for _, field := range [...]string{entry.Name, entry.Provider, entry.Target, entry.URL} {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

// RecentDirectories returns the distinct parent directories seen in the
// history, alphabetically sorted, to power destination suggestions.
func RecentDirectories(entries []Entry, limit int) []string {
	seen := make(map[string]struct{}, len(entries))
	directories := make([]string, 0, min(len(entries), max(limit, 1)))
	for _, entry := range entries {
		if entry.Target == "" {
			continue
		}
		dir := filepath.Dir(ExpandHome(entry.Target))
		key := dir
		if runtime.GOOS == "windows" {
			key = strings.ToLower(dir)
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		directories = append(directories, dir)
	}
	sort.Strings(directories)
	if limit > 0 && len(directories) > limit {
		directories = directories[:limit]
	}
	return directories
}

// writeAtomic writes data to a temp file in dir and renames it over target.
// Shared by the history and the config so both get identical crash safety.
func writeAtomic(dir, pattern, target string, data []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, target); err != nil {
		// Windows refuses to rename over an existing file. Fall back to a
		// direct write; it is still better than dropping the update entirely.
		if runtime.GOOS == "windows" {
			return os.WriteFile(target, data, 0o644)
		}
		return err
	}
	return nil
}
