package procfs

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Status holds memory/thread fields parsed from /proc/<pid>/status.
// Memory values are converted from kB to bytes.
type Status struct {
	VmSize  uint64
	VmRSS   uint64
	Threads int
}

// ParseStatus parses the contents of a /proc/<pid>/status file. VmRSS and
// VmSize are reported in kB in /proc and are converted to bytes.
func ParseStatus(r io.Reader) (Status, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return Status{}, err
	}

	var (
		status   Status
		haveRSS  bool
		haveSize bool
	)

	for _, line := range strings.Split(string(data), "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)

		switch key {
		case "VmRSS":
			status.VmRSS, err = parseKb(val)
			if err != nil {
				return Status{}, err
			}
			haveRSS = true
		case "VmSize":
			status.VmSize, err = parseKb(val)
			if err != nil {
				return Status{}, err
			}
			haveSize = true
		case "Threads":
			status.Threads, err = strconv.Atoi(strings.TrimSpace(val))
			if err != nil {
				return Status{}, fmt.Errorf("procfs: invalid threads %q: %w", strings.TrimSpace(val), err)
			}
		}
	}

	if !haveRSS || !haveSize {
		return Status{}, fmt.Errorf("procfs: missing VmRSS or VmSize")
	}

	return status, nil
}

func parseKb(val string) (uint64, error) {
	parts := strings.Fields(val)
	if len(parts) == 0 {
		return 0, fmt.Errorf("procfs: empty value")
	}
	kb, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("procfs: invalid value %q: %w", parts[0], err)
	}
	return kb * 1024, nil
}
