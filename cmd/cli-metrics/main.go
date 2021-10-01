package main

import (
	"go-marathon-team-3/internal/app/cli-metrics"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	for i := 0; i < 2; i++ {
		basepath = filepath.Dir(basepath)
	}
	app := cli_metrics.CreateMetricsApp(&basepath)
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

