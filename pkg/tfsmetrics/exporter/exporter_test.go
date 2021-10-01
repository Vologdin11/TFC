package exporter

import (
	"go-marathon-team-3/pkg/tfsmetrics/repointerface"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type testItertor struct {
	index   int
	commits []repointerface.Commit
}

func (ti *testItertor) Next() (*repointerface.Commit, error) {
	if ti.index < len(ti.commits) {
		ti.index++
		return &ti.commits[ti.index-1], nil
	}
	return nil, repointerface.ErrNoMoreItems
}

func Test_exporter_PrometheusMetrics(t *testing.T) {
	project1 := "project1"
	iter1 := testItertor{
		index: 0,
		commits: []repointerface.Commit{
			{
				Author:      "Ivan",
				Email:       "ivan@email.com",
				AddedRows:   5,
				DeletedRows: 10,
				Date:        time.Now(),
			},
			{
				Author:      "Ivan",
				Email:       "ivan@email.com",
				AddedRows:   5,
				DeletedRows: 10,
				Date:        time.Now(),
			},
		},
	}

	exporter := exporter{
		metrics: newMetrics(),
	}
	exporter.PrometheusMetrics(&iter1, project1)
	assert.Equal(t, float64(2), testutil.ToFloat64(exporter.metrics.commits.With(prometheus.Labels{"project": project1,
		"author": "Ivan", "email": "ivan@email.com"})))
	assert.Equal(t, float64(10), testutil.ToFloat64(exporter.metrics.addedRows.With(prometheus.Labels{"project": project1,
		"author": "Ivan", "email": "ivan@email.com"})))
	assert.Equal(t, float64(20), testutil.ToFloat64(exporter.metrics.deletedRows.With(prometheus.Labels{"project": project1,
		"author": "Ivan", "email": "ivan@email.com"})))

	project2 := "project2"
	iter2 := testItertor{
		index: 0,
		commits: []repointerface.Commit{
			{
				Author:      "Pety",
				Email:       "pety@email.com",
				AddedRows:   5,
				DeletedRows: 10,
				Date:        time.Now(),
			},
			{
				Author:      "Pety",
				Email:       "pety@email.com",
				AddedRows:   5,
				DeletedRows: 10,
				Date:        time.Now(),
			},
			{
				Author:      "Ivan",
				Email:       "ivan@email.com",
				AddedRows:   5,
				DeletedRows: 10,
				Date:        time.Now(),
			},
		},
	}

	exporter.PrometheusMetrics(&iter2, project2)
	assert.Equal(t, float64(1), testutil.ToFloat64(exporter.metrics.commits.With(prometheus.Labels{"project": project2,
		"author": "Ivan", "email": "ivan@email.com"})))
	assert.Equal(t, float64(2), testutil.ToFloat64(exporter.metrics.commits.With(prometheus.Labels{"project": project2,
		"author": "Pety", "email": "pety@email.com"})))

	assert.Equal(t, float64(5), testutil.ToFloat64(exporter.metrics.addedRows.With(prometheus.Labels{"project": project2,
		"author": "Ivan", "email": "ivan@email.com"})))
	assert.Equal(t, float64(10), testutil.ToFloat64(exporter.metrics.addedRows.With(prometheus.Labels{"project": project2,
		"author": "Pety", "email": "pety@email.com"})))

	assert.Equal(t, float64(10), testutil.ToFloat64(exporter.metrics.deletedRows.With(prometheus.Labels{"project": project2,
		"author": "Ivan", "email": "ivan@email.com"})))
	assert.Equal(t, float64(20), testutil.ToFloat64(exporter.metrics.deletedRows.With(prometheus.Labels{"project": project2,
		"author": "Pety", "email": "pety@email.com"})))
}

func Test_exporter_GetDataByProject(t *testing.T) {
	iter1 := testItertor{
		index: 0,
		commits: []repointerface.Commit{
			{
				Author:      "Ivan",
				Email:       "ivan@email.com",
				AddedRows:   5,
				DeletedRows: 10,
				Date:        time.Now(),
			},
			{
				Author:      "Ivan",
				Email:       "ivan@email.com",
				AddedRows:   5,
				DeletedRows: 10,
				Date:        time.Now(),
			},
			{
				Author:      "Pety",
				Email:       "pety@email.com",
				AddedRows:   5,
				DeletedRows: 10,
				Date:        time.Now(),
			},
		},
	}
	exporter := exporter{}
	data := exporter.GetDataByProject(&iter1)
	assert.Equal(t, &ByProject{
		Commits:     2,
		AddedRows:   10,
		DeletedRows: 20,
	}, data["Ivan"])

	assert.Equal(t, &ByProject{
		Commits:     1,
		AddedRows:   5,
		DeletedRows: 10,
	}, data["Pety"])
}

func Test_exporter_GetDataByAuthor(t *testing.T) {
	project1 := "project1"
	iter1 := testItertor{
		index: 0,
		commits: []repointerface.Commit{
			{
				Author:      "Ivan",
				Email:       "ivan@email.com",
				AddedRows:   5,
				DeletedRows: 10,
				Date:        time.Now(),
			},
			{
				Author:      "Ivan",
				Email:       "ivan@email.com",
				AddedRows:   5,
				DeletedRows: 10,
				Date:        time.Now(),
			},
			{
				Author:      "Pety",
				Email:       "pety@email.com",
				AddedRows:   5,
				DeletedRows: 10,
				Date:        time.Now(),
			},
		},
	}
	exporter := exporter{
		dataByAuthor: make(map[string]*ByAuthor),
	}
	data := exporter.GetDataByAuthor(&iter1, "Ivan", project1)
	assert.Equal(t, &ByAuthor{
		Commits:     2,
		AddedRows:   10,
		DeletedRows: 20,
	}, data[project1])

	project2 := "project2"
	iter2 := testItertor{
		index: 0,
		commits: []repointerface.Commit{
			{
				Author:      "Pety",
				Email:       "pety@email.com",
				AddedRows:   5,
				DeletedRows: 10,
				Date:        time.Now(),
			},
			{
				Author:      "Ivan",
				Email:       "ivan@email.com",
				AddedRows:   5,
				DeletedRows: 10,
				Date:        time.Now(),
			},
		},
	}
	exporter.GetDataByAuthor(&iter2, "Ivan", project2)
	assert.Equal(t, &ByAuthor{
		Commits:     1,
		AddedRows:   5,
		DeletedRows: 10,
	}, data[project2])
}
