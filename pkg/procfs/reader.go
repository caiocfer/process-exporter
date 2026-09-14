package procfs

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing/fstest"
)

// ProcReader abstracts read access to the /proc filesystem so parsers can be
// exercised against in-memory mock filesystems in tests.
type ProcReader interface {
	ReadFile(name string) ([]byte, error)
	ReadDir(name string) ([]fs.DirEntry, error)
}

// OSProcReader reads directly from the host filesystem.
type OSProcReader struct{}

// ReadFile implements ProcReader using the real filesystem.
func (OSProcReader) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// ReadDir implements ProcReader using the real filesystem.
func (OSProcReader) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

type mapFSReader struct {
	fs fs.FS
}

// ReadFile implements ProcReader against an fs.FS implementation. /proc paths
// are absolute, but fs.ValidPath requires relative paths, so the leading slash
// is stripped before delegating to the underlying filesystem.
func (m mapFSReader) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(m.fs, strings.TrimPrefix(name, "/"))
}

// ReadDir implements ProcReader against an fs.FS implementation.
func (m mapFSReader) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(m.fs, strings.TrimPrefix(name, "/"))
}

// NewMapFSReader builds a ProcReader from a map of fake /proc files.
func NewMapFSReader(files map[string][]byte) ProcReader {
	m := make(fstest.MapFS, len(files))
	for k, v := range files {
		m[k] = &fstest.MapFile{Data: v}
	}
	return mapFSReader{fs: m}
}

// ReadStat reads and parses /proc/<pid>/stat.
func ReadStat(r ProcReader, pid int) (Stat, error) {
	data, err := r.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return Stat{}, err
	}
	return ParseStat(bytes.NewReader(data))
}

// ReadStatus reads and parses /proc/<pid>/status.
func ReadStatus(r ProcReader, pid int) (Status, error) {
	data, err := r.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return Status{}, err
	}
	return ParseStatus(bytes.NewReader(data))
}

// ReadIO reads and parses /proc/<pid>/io.
func ReadIO(r ProcReader, pid int) (IO, error) {
	data, err := r.ReadFile(fmt.Sprintf("/proc/%d/io", pid))
	if err != nil {
		return IO{}, err
	}
	return ParseIO(bytes.NewReader(data))
}
