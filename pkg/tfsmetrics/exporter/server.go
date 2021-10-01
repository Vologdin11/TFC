package exporter

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusServer interface {
	// Запускает сервер на переданном порте. Порт передается в виде ":8080"
	Start(addr string)
	// Останавливает сервер с заданным таймаутом. После проверки ошибки надо дождатся WaitGroup.Wait()
	Stop() error
}

type server struct {
	exiteDoneWG      *sync.WaitGroup
	context          context.Context
	timeout          time.Duration
	prometheusServer *http.Server
}

func NewPrometheusServer(exiteDoneWaitGroup *sync.WaitGroup, timeout time.Duration) PrometheusServer {
	return &server{
		exiteDoneWG: exiteDoneWaitGroup,
		context:     context.Background(),
		timeout:     timeout,
	}
}

func (s *server) Start(addr string) {
	server := &http.Server{Addr: addr}
	http.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer,
		promhttp.HandlerOpts{EnableOpenMetrics: true}))

	s.exiteDoneWG.Add(1)

	go func() {
		defer s.exiteDoneWG.Done()
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	s.prometheusServer = server
}

func (s *server) Stop() error {
	cxt, cancel := context.WithTimeout(s.context, s.timeout)
	defer cancel()
	return s.prometheusServer.Shutdown(cxt)
}
