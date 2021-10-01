package exporter

import (
	"go-marathon-team-3/pkg/tfsmetrics/repointerface"

	"github.com/prometheus/client_golang/prometheus"
)

type Exporter interface {
	// Возвращяет данные по проекту
	GetDataByProject(iterator repointerface.CommitIterator) map[string]*ByProject
	// Возвращает данные по автору
	GetDataByAuthor(iterator repointerface.CommitIterator, author string, project string) map[string]*ByAuthor
	// Принимает итератор и создает по нему метрики Prometheus
	PrometheusMetrics(iterator repointerface.CommitIterator, project string)
}

type metrics struct {
	commits     prometheus.CounterVec
	addedRows   prometheus.CounterVec
	deletedRows prometheus.CounterVec
}

func newMetrics() *metrics {
	m := &metrics{
		commits: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "commits",
		}, []string{"project", "author", "email"}),
		addedRows: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "added_rows",
		}, []string{"project", "author", "email"}),
		deletedRows: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "deleted_rows",
		}, []string{"project", "author", "email"}),
	}
	prometheus.MustRegister(m.commits, m.addedRows, m.deletedRows)
	return m
}

type exporter struct {
	metrics      *metrics
	dataByAuthor map[string]*ByAuthor
}

func NewExporter() Exporter {
	return &exporter{
		metrics:      newMetrics(),
		dataByAuthor: make(map[string]*ByAuthor),
	}
}

func (e *exporter) PrometheusMetrics(iterator repointerface.CommitIterator, project string) {
	for commit, err := iterator.Next(); err == nil; commit, err = iterator.Next() {
		e.metrics.commits.With(prometheus.Labels{"project": project,
			"author": commit.Author, "email": commit.Email}).Inc()
		e.metrics.addedRows.With(prometheus.Labels{"project": project,
			"author": commit.Author, "email": commit.Email}).Add(float64(commit.AddedRows))
		e.metrics.deletedRows.With(prometheus.Labels{"project": project,
			"author": commit.Author, "email": commit.Email}).Add(float64(commit.DeletedRows))

	}
}

type ByAuthor struct {
	Commits     int
	AddedRows   int
	DeletedRows int
}

type ByProject struct {
	Commits     int
	AddedRows   int
	DeletedRows int
}

func (e *exporter) GetDataByProject(iterator repointerface.CommitIterator) map[string]*ByProject {
	res := make(map[string]*ByProject)
	for commit, err := iterator.Next(); err == nil; commit, err = iterator.Next() {
		if author, ok := res[commit.Author]; ok {
			author.Commits += 1
			author.AddedRows += commit.AddedRows
			author.DeletedRows += commit.DeletedRows
		} else {
			res[commit.Author] = &ByProject{
				Commits:     1,
				AddedRows:   commit.AddedRows,
				DeletedRows: commit.DeletedRows,
			}
		}
	}
	return res
}

func (e *exporter) GetDataByAuthor(iterator repointerface.CommitIterator, author string, project string) map[string]*ByAuthor {
	for commit, err := iterator.Next(); err == nil; commit, err = iterator.Next() {
		if auth, ok := e.dataByAuthor[project]; ok {
			if commit.Author == author {
				auth.Commits += 1
				auth.AddedRows += commit.AddedRows
				auth.DeletedRows += commit.DeletedRows
			}
		} else {
			if commit.Author == author {
				e.dataByAuthor[project] = &ByAuthor{
					Commits:     1,
					AddedRows:   commit.AddedRows,
					DeletedRows: commit.DeletedRows,
				}
			}
		}
	}
	return e.dataByAuthor
}
