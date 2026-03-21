package keystore

import (
	"sort"
	"strconv"
	"strings"

	"go.etcd.io/bbolt"
)

var defaultBucket = []byte("kv")

// Entry represents a key-value pair in the local store.
type Entry struct {
	Key   string
	Value string
}

// KeyValueStore defines the interface for key-value storage operations.
// This enables mocking in tests and decouples business logic from the storage implementation.
type KeyValueStore interface {
	Set(key, value string) error
	Get(key string) (string, bool, error)
	List(limit int) ([]Entry, error)
	ListPage(offset, limit int) ([]Entry, error)
	ListNumericRange(minID, maxID int) ([]Entry, error)
	Count() (int, error)
	SearchPage(query string, offset, limit int) ([]Entry, error)
	CountSearch(query string) (int, error)
	Delete(key string) error
	Close() error
}

// Store implements KeyValueStore using bbolt.
type Store struct {
	db *bbolt.DB
}

// NewStore opens or creates the database file and ensures the default bucket exists.
func NewStore(path string) (*Store, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(defaultBucket); err != nil {
			return err
		}
		if err := tx.DeleteBucket([]byte("on_market")); err != nil && err != bbolt.ErrBucketNotFound {
			return err
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Set stores a key-value pair.
func (s *Store) Set(key, value string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(defaultBucket).Put([]byte(key), []byte(value))
	})
}

// Get retrieves a value by key. Returns (value, true, nil) if found, ("", false, nil) otherwise.
func (s *Store) Get(key string) (string, bool, error) {
	var val []byte
	err := s.db.View(func(tx *bbolt.Tx) error {
		val = tx.Bucket(defaultBucket).Get([]byte(key))
		return nil
	})
	if err != nil {
		return "", false, err
	}
	if val == nil {
		return "", false, nil
	}
	return string(val), true, nil
}

// List returns up to limit key-value entries from the default bucket.
func (s *Store) List(limit int) ([]Entry, error) {
	return s.ListPage(0, limit)
}

func compareEntries(a, b Entry) bool {
	ai, aErr := strconv.Atoi(a.Key)
	bi, bErr := strconv.Atoi(b.Key)

	if aErr == nil && bErr == nil {
		return ai < bi
	}
	if aErr == nil {
		return true
	}
	if bErr == nil {
		return false
	}
	return a.Key < b.Key
}

func (s *Store) collectSortedEntries(query string) ([]Entry, error) {
	query = strings.TrimSpace(query)
	entries := make([]Entry, 0)

	err := s.db.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket(defaultBucket).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			key := string(k)
			if query != "" && !strings.Contains(key, query) {
				continue
			}
			entries = append(entries, Entry{Key: key, Value: string(v)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return compareEntries(entries[i], entries[j])
	})

	return entries, nil
}

func paginateEntries(entries []Entry, offset, limit int) []Entry {
	if limit <= 0 {
		return []Entry{}
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(entries) {
		return []Entry{}
	}
	end := offset + limit
	if end > len(entries) {
		end = len(entries)
	}
	return entries[offset:end]
}

// ListPage returns up to limit entries starting from offset.
func (s *Store) ListPage(offset, limit int) ([]Entry, error) {
	entries, err := s.collectSortedEntries("")
	if err != nil {
		return nil, err
	}

	return paginateEntries(entries, offset, limit), nil
}

// ListNumericRange returns entries whose key is a numeric ID in [minID, maxID].
func (s *Store) ListNumericRange(minID, maxID int) ([]Entry, error) {
	if minID > maxID {
		minID, maxID = maxID, minID
	}

	entries, err := s.collectSortedEntries("")
	if err != nil {
		return nil, err
	}

	filtered := make([]Entry, 0)
	for _, entry := range entries {
		id, convErr := strconv.Atoi(entry.Key)
		if convErr != nil {
			continue
		}
		if id >= minID && id <= maxID {
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}

// Count returns the number of entries in the default bucket.
func (s *Store) Count() (int, error) {
	entries, err := s.collectSortedEntries("")
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

// SearchPage returns up to limit entries whose key contains query,
// starting from offset within the filtered result set.
func (s *Store) SearchPage(query string, offset, limit int) ([]Entry, error) {
	entries, err := s.collectSortedEntries(query)
	if err != nil {
		return nil, err
	}

	return paginateEntries(entries, offset, limit), nil
}

// CountSearch returns how many entries match query by key containment.
func (s *Store) CountSearch(query string) (int, error) {
	entries, err := s.collectSortedEntries(query)
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

// Delete removes a key from the store.
func (s *Store) Delete(key string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(defaultBucket).Delete([]byte(key))
	})
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}
