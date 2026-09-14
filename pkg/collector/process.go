package collector

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/caiocfer/process-exporter/pkg/procfs"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	cpuSecondsName  = "process_cpu_seconds_total"
	rssName         = "process_resident_memory_bytes"
	vmemName        = "process_virtual_memory_bytes"
	threadsName     = "process_threads_total"
	ioReadName      = "process_io_read_bytes_total"
	ioWriteName     = "process_io_write_bytes_total"
)

var (
	cpuDesc = prometheus.NewDesc(cpuSecondsName,
		"Total CPU time consumed (utime + stime)",
		[]string{"pid", "process_name", "cmdline", "state", "cpu_id"}, nil)
	rssDesc = prometheus.NewDesc(rssName,
		"Resident memory size (VmRSS)",
		[]string{"pid", "process_name", "cmdline"}, nil)
	vmemDesc = prometheus.NewDesc(vmemName,
		"Virtual memory size (VmSize)",
		[]string{"pid", "process_name", "cmdline"}, nil)
	threadsDesc = prometheus.NewDesc(threadsName,
		"Number of active threads",
		[]string{"pid", "process_name"}, nil)
	ioReadDesc = prometheus.NewDesc(ioReadName,
		"Disk read bytes from /proc/<pid>/io",
		[]string{"pid", "process_name"}, nil)
	ioWriteDesc = prometheus.NewDesc(ioWriteName,
		"Disk write bytes from /proc/<pid>/io",
		[]string{"pid", "process_name"}, nil)
)

// Options configures a ProcessCollector.
type Options struct {
	// Path is the root /proc directory to scan (e.g. "/proc").
	Path string
	// ProcMinRSSMB is the minimum resident memory in MB for a process to be
	// exposed. Zero exposes all processes.
	ProcMinRSSMB int
	// CacheTTL bounds how long stable metadata (cmdline, process_name) is cached
	// per process.
	CacheTTL time.Duration
}

type metaEntry struct {
	name     string
	cmdline  string
	cachedAt time.Time
}

// ProcessCollector implements prometheus.Collector, exposing per-process
// resource metrics parsed from the /proc filesystem.
type ProcessCollector struct {
	reader      procfs.ProcReader
	root        string
	minRSSBytes uint64
	cacheTTL    time.Duration

	mu    sync.Mutex
	cache map[int]*metaEntry
}

// NewProcessCollector builds a ProcessCollector that reads from r.
func NewProcessCollector(r procfs.ProcReader, opts Options) *ProcessCollector {
	var minBytes uint64
	if opts.ProcMinRSSMB > 0 {
		minBytes = uint64(opts.ProcMinRSSMB) * 1024 * 1024
	}
	return &ProcessCollector{
		reader:      r,
		root:        opts.Path,
		minRSSBytes: minBytes,
		cacheTTL:    opts.CacheTTL,
		cache:       make(map[int]*metaEntry),
	}
}

// Describe implements prometheus.Collector.
func (c *ProcessCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- cpuDesc
	ch <- rssDesc
	ch <- vmemDesc
	ch <- threadsDesc
	ch <- ioReadDesc
	ch <- ioWriteDesc
}

// Collect implements prometheus.Collector.
func (c *ProcessCollector) Collect(ch chan<- prometheus.Metric) {
	pids, err := listPIDs(c.reader, c.root)
	if err != nil {
		return
	}
	for _, pid := range pids {
		if err := c.collectOne(pid, ch); err != nil {
			continue
		}
	}
}

func (c *ProcessCollector) collectOne(pid int, ch chan<- prometheus.Metric) error {
	stat, err := procfs.ReadStat(c.reader, pid)
	if err != nil {
		return err
	}
	status, err := procfs.ReadStatus(c.reader, pid)
	if err != nil {
		return err
	}
	if status.VmRSS < c.minRSSBytes {
		return nil
	}

	name, cmdline := c.meta(pid, stat.Name)
	pidStr := strconv.Itoa(pid)

	c.emit(ch, cpuDesc, prometheus.CounterValue, float64(stat.UTime+stat.STime),
		pidStr, name, cmdline, stat.State, strconv.Itoa(stat.CPUID))
	c.emit(ch, rssDesc, prometheus.GaugeValue, float64(status.VmRSS), pidStr, name, cmdline)
	c.emit(ch, vmemDesc, prometheus.GaugeValue, float64(status.VmSize), pidStr, name, cmdline)
	c.emit(ch, threadsDesc, prometheus.GaugeValue, float64(status.Threads), pidStr, name)

	if ioStats, err := procfs.ReadIO(c.reader, pid); err == nil {
		c.emit(ch, ioReadDesc, prometheus.CounterValue, float64(ioStats.ReadBytes), pidStr, name)
		c.emit(ch, ioWriteDesc, prometheus.CounterValue, float64(ioStats.WriteBytes), pidStr, name)
	}
	return nil
}

func (c *ProcessCollector) emit(ch chan<- prometheus.Metric, desc *prometheus.Desc, typ prometheus.ValueType, val float64, labels ...string) {
	m, err := prometheus.NewConstMetric(desc, typ, val, labels...)
	if err != nil {
		return
	}
	ch <- m
}

// meta returns the cached (or freshly read) process name and cmdline for pid.
// Metadata is stable, so it is cached up to cacheTTL to reduce scrape overhead.
func (c *ProcessCollector) meta(pid int, name string) (string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if e, ok := c.cache[pid]; ok && now.Sub(e.cachedAt) < c.cacheTTL {
		return e.name, e.cmdline
	}

	args, err := procfs.ReadCmdline(c.reader, pid)
	if err != nil {
		args = nil
	}
	entry := &metaEntry{name: name, cmdline: strings.Join(args, " "), cachedAt: now}
	c.cache[pid] = entry
	return entry.name, entry.cmdline
}

func listPIDs(r procfs.ProcReader, root string) ([]int, error) {
	entries, err := r.ReadDir(root)
	if err != nil {
		return nil, err
	}
	pids := make([]int, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	sort.Ints(pids)
	return pids, nil
}
