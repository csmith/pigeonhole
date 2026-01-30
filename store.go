package main

import (
	"time"

	bolt "go.etcd.io/bbolt"
)

type Message struct {
	Time    time.Time `json:"time"`
	Content string    `json:"message"`
}

type store struct {
	db *bolt.DB
}

func newStore(path string) (*store, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	return &store{db: db}, nil
}

func (s *store) insert(cubby, message string) (time.Time, error) {
	now := time.Now()
	key := now.Format(time.RFC3339Nano)

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(cubby))
		if err != nil {
			return err
		}

		return bucket.Put([]byte(key), []byte(message))
	})

	return now, err
}

func (s *store) retrieve(cubby string) ([]Message, error) {
	var messages []Message

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(cubby))
		if bucket == nil {
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			t, err := time.Parse(time.RFC3339Nano, string(k))
			if err != nil {
				return err
			}

			messages = append(messages, Message{
				Time:    t,
				Content: string(v),
			})
			return nil
		})
	})

	return messages, err
}

func (s *store) clear(cubby string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(cubby))
	})
}

func (s *store) removeOldest(cubby string, notafter time.Time) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(cubby))
		if bucket == nil {
			return nil
		}

		c := bucket.Cursor()
		prefix := notafter.Format(time.RFC3339Nano)

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if string(k) <= prefix {
				if err := c.Delete(); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (s *store) close() error {
	return s.db.Close()
}
