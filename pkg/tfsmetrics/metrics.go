package tfsmetrics

import (
	"go-marathon-team-3/pkg/tfsmetrics/azure"
	"go-marathon-team-3/pkg/tfsmetrics/repointerface"
	"go-marathon-team-3/pkg/tfsmetrics/store"
)

type commitsCollection struct {
	nameOfProject string
	azure         azure.AzureInterface

	cache bool
	store store.Store
}

// Если cache = false, то в store передаем nil
func NewCommitCollection(nameOfProject string, azure azure.AzureInterface, cache bool, store store.Store) repointerface.Repository {
	return &commitsCollection{
		nameOfProject: nameOfProject,
		azure:         azure,
		cache:         cache,
		store:         store,
	}
}

func (c *commitsCollection) Open() error {
	if c.cache {
		c.store.InitProject(c.nameOfProject)
	}
	return c.azure.TfvcClientConnection()
}

func (c *commitsCollection) GetCommitIterator() (repointerface.CommitIterator, error) {
	changeSets, err := c.azure.GetChangesets(c.nameOfProject)
	if err != nil {
		return nil, err
	}
	return &iterator{
		index:         0,
		commits:       changeSets,
		nameOfProject: c.nameOfProject,
		azure:         c.azure.Azure(),
		cache:         c.cache,
		store:         c.store,
	}, nil
}

type iterator struct {
	index   int
	commits []*int

	nameOfProject string
	azure         azure.AzureInterface

	cache bool
	store store.Store
}

func (i *iterator) Next() (*repointerface.Commit, error) {
	if i.index < len(i.commits) {
		i.index++
		if i.cache {
			changeSet, err := i.store.FindOne(*i.commits[i.index-1], i.nameOfProject)
			if err == nil {
				return changeSet, err
			}
		}
		changeSet, err := i.azure.GetChangesetChanges(i.commits[i.index-1], i.nameOfProject)
		if err != nil {
			return nil, err
		}
		commit := repointerface.Commit{
			Id:          changeSet.Id,
			Author:      changeSet.Author,
			Email:       changeSet.Email,
			AddedRows:   changeSet.AddedRows,
			DeletedRows: changeSet.DeletedRows,
			Date:        changeSet.Date,
			Message:     changeSet.Message,
			Hash:        changeSet.Hash,
		}
		if i.cache {
			if err := i.store.Write(&commit, i.nameOfProject); err != nil {
				return &commit, err
			}
		}
		return &commit, nil
	}
	return nil, repointerface.ErrNoMoreItems
}
