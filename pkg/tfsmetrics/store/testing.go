package store

import (
	"go-marathon-team-3/pkg/tfsmetrics/repointerface"

	bolt "go.etcd.io/bbolt"
)

type TestIterator struct {
	Index   int
	Commits []repointerface.Commit
}

func (ti *TestIterator) Next() (*repointerface.Commit, error) {
	if ti.Index < len(ti.Commits) {
		ti.Index++
		return &ti.Commits[ti.Index-1], nil
	}
	return nil, repointerface.ErrNoMoreItems
}

func TestStore() (*DB, error) {
	bolt, err := bolt.Open("pkg", 0600, nil)
	if err != nil {
		return nil, err
	}
	store := DB{
		DB: bolt,
	}
	return &store, nil
}
