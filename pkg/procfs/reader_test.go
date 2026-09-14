package procfs_test

import (
	"strings"
	"testing"

	"github.com/caiocfer/process-exporter/pkg/procfs"
)

var validStatFields = []string{
	"S", "1230", "1230", "1230", "0", "-1", "4194304", "12345", "6789",
	"0", "0", "100", "20", "0", "0", "20", "0", "5", "0", "12345",
	"67890123", "1234",
	"0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0", "0",
	"0", "3",
}

func buildStat(pid, comm string, fields []string) string {
	return pid + " (" + comm + ") " + strings.Join(fields, " ")
}

func mutateField(fields []string, idx int, val string) []string {
	out := make([]string, len(fields))
	copy(out, fields)
	out[idx] = val
	return out
}

func TestParseStatValid(t *testing.T) {
	data := buildStat("1234", "bash", validStatFields) + "\n"

	stat, err := procfs.ParseStat(strings.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stat.PID != 1234 {
		t.Errorf("PID = %d, want 1234", stat.PID)
	}
	if stat.Name != "bash" {
		t.Errorf("Name = %q, want %q", stat.Name, "bash")
	}
	if stat.State != "S" {
		t.Errorf("State = %q, want %q", stat.State, "S")
	}
	if stat.UTime != 100 {
		t.Errorf("UTime = %d, want 100", stat.UTime)
	}
	if stat.STime != 20 {
		t.Errorf("STime = %d, want 20", stat.STime)
	}
	if stat.CPUID != 3 {
		t.Errorf("CPUID = %d, want 3", stat.CPUID)
	}
}

func TestParseStatCommWithSpacesAndParens(t *testing.T) {
	data := buildStat("42", "my worker (main)", validStatFields) + "\n"

	stat, err := procfs.ParseStat(strings.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stat.PID != 42 {
		t.Errorf("PID = %d, want 42", stat.PID)
	}
	if stat.Name != "my worker (main)" {
		t.Errorf("Name = %q, want %q", stat.Name, "my worker (main)")
	}
	if stat.State != "S" {
		t.Errorf("State = %q, want %q", stat.State, "S")
	}
}

func TestParseStatCorrupt(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{"empty", ""},
		{"missing-close-paren", "100 (proc S 1 1 1 0 -1 1 2 0 0 1 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0"},
		{"too-few-fields", buildStat("100", "proc", validStatFields[:5])},
		{"non-numeric-utime", buildStat("100", "proc", mutateField(validStatFields, 11, "x"))},
		{"non-numeric-cpu", buildStat("100", "proc", mutateField(validStatFields, 35, "y"))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := procfs.ParseStat(strings.NewReader(tc.data)); err == nil {
				t.Fatalf("expected error, got nil for %q", tc.data)
			}
		})
	}
}

func TestParseStatusValid(t *testing.T) {
	data := "Name:   bash\nState:  S (sleeping)\nThreads:        4\nVmSize:  102400 kB\nVmRSS:   51200 kB\n"

	status, err := procfs.ParseStatus(strings.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.VmSize != 102400*1024 {
		t.Errorf("VmSize = %d, want %d", status.VmSize, 102400*1024)
	}
	if status.VmRSS != 51200*1024 {
		t.Errorf("VmRSS = %d, want %d", status.VmRSS, 51200*1024)
	}
	if status.Threads != 4 {
		t.Errorf("Threads = %d, want 4", status.Threads)
	}
}

func TestParseStatusCorrupt(t *testing.T) {
	if _, err := procfs.ParseStatus(strings.NewReader("")); err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

func TestParseIOValid(t *testing.T) {
	data := "rchar: 100\nwchar: 200\nsyscr: 3\nsyscw: 4\nread_bytes: 1000\nwrite_bytes: 2000\ncancelled_write_bytes: 300\n"

	i, err := procfs.ParseIO(strings.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if i.ReadBytes != 1000 {
		t.Errorf("ReadBytes = %d, want 1000", i.ReadBytes)
	}
	if i.WriteBytes != 2000 {
		t.Errorf("WriteBytes = %d, want 2000", i.WriteBytes)
	}
}

func TestParseIORespectOnlyKnownFields(t *testing.T) {
	data := "read_bytes: 1000\nwrite_bytes: 2000\nunknown_field: 9999\n"

	i, err := procfs.ParseIO(strings.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if i.ReadBytes != 1000 || i.WriteBytes != 2000 {
		t.Errorf("got %+v, want ReadBytes=1000 WriteBytes=2000", i)
	}
}

func TestReadStatFromMapFS(t *testing.T) {
	files := map[string][]byte{
		"proc/1234/stat": []byte(buildStat("1234", "bash", validStatFields) + "\n"),
	}
	r := procfs.NewMapFSReader(files)

	stat, err := procfs.ReadStat(r, 1234)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stat.PID != 1234 || stat.UTime != 100 || stat.CPUID != 3 {
		t.Errorf("unexpected stat: %+v", stat)
	}
}

func TestReadStatMissingFile(t *testing.T) {
	r := procfs.NewMapFSReader(map[string][]byte{})
	if _, err := procfs.ReadStat(r, 9999); err == nil {
		t.Fatal("expected error for missing proc file, got nil")
	}
}

func TestReadStatusAndIOFromMapFS(t *testing.T) {
	files := map[string][]byte{
		"proc/1234/status": []byte("Threads:        7\nVmRSS:   51200 kB\nVmSize:  102400 kB\n"),
		"proc/1234/io":     []byte("read_bytes: 1000\nwrite_bytes: 2000\n"),
	}
	r := procfs.NewMapFSReader(files)

	status, err := procfs.ReadStatus(r, 1234)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Threads != 7 {
		t.Errorf("Threads = %d, want 7", status.Threads)
	}

	i, err := procfs.ReadIO(r, 1234)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if i.ReadBytes != 1000 || i.WriteBytes != 2000 {
		t.Errorf("got %+v, want ReadBytes=1000 WriteBytes=2000", i)
	}
}
