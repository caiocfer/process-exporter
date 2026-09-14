package procfs

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// IO holds disk I/O fields parsed from /proc/<pid>/io.
type IO struct {
	ReadBytes  uint64
	WriteBytes uint64
}

// ParseIO parses the contents of a /proc/<pid>/io file. Only read_bytes and
// write_bytes are extracted; other fields are ignored.
func ParseIO(r io.Reader) (IO, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return IO{}, err
	}

	var (
		ioStats    IO
		foundRead  bool
		foundWrite bool
	)

	for _, line := range strings.Split(string(data), "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)

		switch key {
		case "read_bytes":
			ioStats.ReadBytes, err = strconv.ParseUint(strings.TrimSpace(val), 10, 64)
			if err != nil {
				return IO{}, fmt.Errorf("procfs: invalid read_bytes %q: %w", strings.TrimSpace(val), err)
			}
			foundRead = true
		case "write_bytes":
			ioStats.WriteBytes, err = strconv.ParseUint(strings.TrimSpace(val), 10, 64)
			if err != nil {
				return IO{}, fmt.Errorf("procfs: invalid write_bytes %q: %w", strings.TrimSpace(val), err)
			}
			foundWrite = true
		}
	}

	if !foundRead || !foundWrite {
		return IO{}, fmt.Errorf("procfs: missing read_bytes or write_bytes")
	}

	return ioStats, nil
}
