package keystore

import (
	"go.etcd.io/bbolt"
)

var defaultBucket = []byte("kv")

type Store struct {
	db *bbolt.DB
}

// NewStore abre o crea el archivo de base de datos y asegura que exista el bucket
func NewStore(path string) (*Store, error) {
	db, err := bbolt.Open(path, 0666, nil)
	if err != nil {
		return nil, err
	}

	// Aseguramos que el bucket exista
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

// Set guarda una clave-valor
func (s *Store) Set(key string, value string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(defaultBucket).Put([]byte(key), []byte(value))
	})
}

// Get obtiene una clave. Devuelve (valor, true) si existe, si no, ("", false)
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

// Delete elimina una clave
func (s *Store) Delete(key string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(defaultBucket).Delete([]byte(key))
	})
}

// Close cierra la base de datos
func (s *Store) Close() error {
	return s.db.Close()
}
