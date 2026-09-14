package procfs

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Stat holds the parsed fields from /proc/<pid>/stat.
type Stat struct {
	PID   int
	Name  string
	State string
	UTime uint64
	STime uint64
	CPUID int
}

// ParseStat parses the contents of a /proc/<pid>/stat file.
//
// The comm field (field 2) may contain spaces and parentheses, so the last
// closing parenthesis delimits its end. After comm, utime is field 14 and
// stime is field 15 (1-indexed); the final field is the processor (cpu_id).
func ParseStat(r io.Reader) (Stat, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return Stat{}, err
	}

	open := bytes.IndexByte(data, '(')
	closeParen := bytes.LastIndexByte(data, ')')
	if open == -1 || closeParen == -1 || closeParen < open {
		return Stat{}, fmt.Errorf("procfs: invalid stat format: missing parentheses")
	}

	pidStr := strings.TrimSpace(string(data[:open]))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return Stat{}, fmt.Errorf("procfs: invalid pid %q: %w", pidStr, err)
	}

	name := string(data[open+1 : closeParen])
	fields := strings.Fields(string(data[closeParen+1:]))
	if len(fields) < 13 {
		return Stat{}, fmt.Errorf("procfs: stat has %d fields after comm, need at least 13", len(fields))
	}

	state := fields[0]

	utime, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return Stat{}, fmt.Errorf("procfs: invalid utime %q: %w", fields[11], err)
	}

	stime, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return Stat{}, fmt.Errorf("procfs: invalid stime %q: %w", fields[12], err)
	}

	cpuStr := fields[len(fields)-1]
	cpuID, err := strconv.Atoi(cpuStr)
	if err != nil {
		return Stat{}, fmt.Errorf("procfs: invalid cpu id %q: %w", cpuStr, err)
	}

	return Stat{
		PID:   pid,
		Name:  name,
		State: state,
		UTime: utime,
		STime: stime,
		CPUID: cpuID,
	}, nil
}
