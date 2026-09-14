package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/caiocfer/process-exporter/pkg/collector"
	"github.com/caiocfer/process-exporter/pkg/procfs"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	addr := flag.String("addr", ":9835", "Listen address")
	path := flag.String("path", "/metrics", "Metrics HTTP path")
	cacheTTL := flag.Duration("cache-ttl", 60*time.Second, "Process metadata cache TTL")
	proc := flag.String("proc", "/proc", "Path to proc filesystem")
	minRSS := flag.Int("proc-min-rss-mb", 10, "Minimum RSS threshold (MB) to expose system processes")
	flag.Parse()

	c := collector.NewProcessCollector(procfs.OSProcReader{}, collector.Options{
		Path:         *proc,
		ProcMinRSSMB: *minRSS,
		CacheTTL:     *cacheTTL,
	})

	reg := prometheus.NewRegistry()
	reg.MustRegister(c)

	http.Handle(*path, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	srv := &http.Server{Addr: *addr}
	log.Printf("system-process-exporter listening on %s%s", *addr, *path)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
