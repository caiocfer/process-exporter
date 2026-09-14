package collector_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/caiocfer/process-exporter/pkg/collector"
	"github.com/caiocfer/process-exporter/pkg/procfs"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func testFiles() map[string][]byte {
	return map[string][]byte{
		"proc/1234/stat":    []byte("1234 (bash) S 1 1 1 0 -1 4194304 100 20 0 0 100 20 0 0 20 0 1 0 5000 67890123 1234 0 0 0 0 0 0 0 0 0 0 0 0 0\n"),
		"proc/1234/status":  []byte("Name:   bash\nThreads:        4\nVmSize:  102400 kB\nVmRSS:   51200 kB\n"),
		"proc/1234/io":      []byte("read_bytes: 1000\nwrite_bytes: 2000\n"),
		"proc/1234/cmdline": []byte("bash\000"),
		"proc/2/stat":       []byte("2 (kthreadd) S 0 0 0 0 -1 4194304 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0\n"),
		"proc/2/status":     []byte("Name:   kthreadd\nThreads:        1\nVmSize: 0 kB\nVmRSS: 0 kB\n"),
	}
}

const expectedOutput = `
# HELP process_cpu_seconds_total Total CPU time consumed (utime + stime)
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total{cmdline="bash",cpu_id="0",pid="1234",process_name="bash",state="S"} 120
# HELP process_resident_memory_bytes Resident memory size (VmRSS)
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes{cmdline="bash",pid="1234",process_name="bash"} 52428800
# HELP process_virtual_memory_bytes Virtual memory size (VmSize)
# TYPE process_virtual_memory_bytes gauge
process_virtual_memory_bytes{cmdline="bash",pid="1234",process_name="bash"} 104857600
# HELP process_threads_total Number of active threads
# TYPE process_threads_total gauge
process_threads_total{pid="1234",process_name="bash"} 4
# HELP process_io_read_bytes_total Disk read bytes from /proc/<pid>/io
# TYPE process_io_read_bytes_total counter
process_io_read_bytes_total{pid="1234",process_name="bash"} 1000
# HELP process_io_write_bytes_total Disk write bytes from /proc/<pid>/io
# TYPE process_io_write_bytes_total counter
process_io_write_bytes_total{pid="1234",process_name="bash"} 2000
`

func TestProcessCollectorEmitsAllMetrics(t *testing.T) {
	c := collector.NewProcessCollector(procfs.NewMapFSReader(testFiles()), collector.Options{
		Path:         "/proc",
		ProcMinRSSMB: 10,
		CacheTTL:     time.Minute,
	})

	err := testutil.CollectAndCompare(c, strings.NewReader(expectedOutput),
		"process_cpu_seconds_total",
		"process_resident_memory_bytes",
		"process_virtual_memory_bytes",
		"process_threads_total",
		"process_io_read_bytes_total",
		"process_io_write_bytes_total",
	)
	if err != nil {
		t.Fatalf("unexpected metric mismatch: %v", err)
	}
}

func TestProcessCollectorRSSFilter(t *testing.T) {
	c := collector.NewProcessCollector(procfs.NewMapFSReader(testFiles()), collector.Options{
		Path:         "/proc",
		ProcMinRSSMB: 10,
		CacheTTL:     time.Minute,
	})

	expected := `
# HELP process_cpu_seconds_total Total CPU time consumed (utime + stime)
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total{cmdline="bash",cpu_id="0",pid="1234",process_name="bash",state="S"} 120
# HELP process_resident_memory_bytes Resident memory size (VmRSS)
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes{cmdline="bash",pid="1234",process_name="bash"} 52428800
`

	err := testutil.CollectAndCompare(c, strings.NewReader(expected),
		"process_cpu_seconds_total",
		"process_resident_memory_bytes",
	)
	if err != nil {
		t.Fatalf("unexpected metric mismatch after RSS filter: %v", err)
	}
}

type recordingReader struct {
	procfs.ProcReader
	mu             sync.Mutex
	cmdlineCalls   int
}

func (r *recordingReader) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cmdlineCalls
}

func (r *recordingReader) ReadFile(name string) ([]byte, error) {
	if strings.HasSuffix(name, "cmdline") {
		r.mu.Lock()
		r.cmdlineCalls++
		r.mu.Unlock()
	}
	return r.ProcReader.ReadFile(name)
}

func TestProcessCollectorTTLCache(t *testing.T) {
	r := &recordingReader{ProcReader: procfs.NewMapFSReader(testFiles())}
	c := collector.NewProcessCollector(r, collector.Options{
		Path:         "/proc",
		ProcMinRSSMB: 10,
		CacheTTL:     time.Minute,
	})

	reg := prometheus.NewRegistry()
	reg.MustRegister(c)
	if _, err := reg.Gather(); err != nil {
		t.Fatalf("first collect failed: %v", err)
	}
	if _, err := reg.Gather(); err != nil {
		t.Fatalf("second collect failed: %v", err)
	}

	if got := r.count(); got != 1 {
		t.Fatalf("cmdline read count = %d, want 1 (second served from cache)", got)
	}
}
