package store

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"go-marathon-team-3/pkg/tfsmetrics/repointerface"

	bolt "go.etcd.io/bbolt"
)

type Store interface {
	InitProject(projectName string) error
	Close() error
	FindOne(id int, projectName string) (*repointerface.Commit, error)
	Write(commit *repointerface.Commit, projectName string) error
}

type DB struct {
	DB *bolt.DB
}

func NewStore() (Store, error) {
	db, err := bolt.Open("assets.db", 0600, nil)
	if err != nil {
		return nil, err
	}

	return &DB{DB: db}, nil
}

func (db *DB) InitProject(projectName string) error {
	return db.DB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(projectName))
		if err != nil {
			return err
		}
		return nil
	})
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) FindOne(id int, projectName string) (*repointerface.Commit, error) {
	res := &repointerface.Commit{}
	err := db.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(projectName))
		v := b.Get(itob(id))

		if v == nil {
			return errors.New("no item")
		}
		if err := json.Unmarshal(v, res); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (db *DB) Write(commit *repointerface.Commit, projectName string) error {
	err := db.DB.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(projectName))
		if err != nil {
			return err
		}
		v := b.Get(itob(commit.Id))

		if v != nil {
			return nil
		}

		buf, err := json.Marshal(commit)
		if err != nil {
			return err
		}

		return b.Put(itob(commit.Id), buf)
	})
	if err != nil {
		return err
	}

	return nil
}

func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
