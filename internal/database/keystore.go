package keystore

import (
	"go.etcd.io/bbolt"
)

var defaultBucket = []byte("kv")

// KeyValueStore defines the interface for key-value storage operations.
// This enables mocking in tests and decouples business logic from the storage implementation.
type KeyValueStore interface {
	Set(key, value string) error
	Get(key string) (string, bool, error)
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
		_, err := tx.CreateBucketIfNotExists(defaultBucket)
		return err
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
