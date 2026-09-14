package procfs

import (
	"bytes"
	"fmt"
)

// ReadCmdline reads and parses /proc/<pid>/cmdline, whose arguments are
// NUL-separated.
func ReadCmdline(r ProcReader, pid int) ([]string, error) {
	data, err := r.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return nil, err
	}
	return ParseCmdline(data)
}

// ParseCmdline splits NUL-separated /proc/<pid>/cmdline data into arguments,
// dropping the trailing empty element produced by the terminating NUL byte.
func ParseCmdline(data []byte) ([]string, error) {
	parts := bytes.Split(data, []byte{0})
	args := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		args = append(args, string(p))
	}
	return args, nil
}
