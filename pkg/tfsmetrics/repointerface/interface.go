package repointerface

import (
	"errors"
	"time"
)

var ErrNoMoreItems error = errors.New("no more items")

type Repository interface {
	Open() error // Вызывать для каждого проекта, если включен кэш
	GetCommitIterator() (CommitIterator, error)
}

type CommitIterator interface {
	Next() (*Commit, error)
}

type Commit struct {
	Id          int
	Author      string // обязательное поле
	Email       string
	AddedRows   int       // обязательное поле
	DeletedRows int       // обязательное поле
	Date        time.Time // обязательное поле
	Message     string
	Hash        string
}
