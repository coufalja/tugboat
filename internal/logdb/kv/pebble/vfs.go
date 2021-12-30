package pebble

import (
	"io"
	"os"

	pvfs "github.com/cockroachdb/pebble/vfs"
	"github.com/coufalja/tugboat/internal/vfs"
)

// PebbleFS is a wrapper struct that implements the pebble/vfs.FS interface.
type PebbleFS struct {
	fs vfs.IFS
}

var _ pvfs.FS = (*PebbleFS)(nil)

// NewPebbleFS creates a new pebble/vfs.FS instance.
func NewPebbleFS(fs vfs.IFS) pvfs.FS {
	return &PebbleFS{fs}
}

// GetFreeSpace ...
func (p *PebbleFS) GetFreeSpace(path string) (uint64, error) {
	return p.fs.GetFreeSpace(path)
}

// Create ...
func (p *PebbleFS) Create(name string) (pvfs.File, error) {
	return p.fs.Create(name)
}

// Link ...
func (p *PebbleFS) Link(oldname, newname string) error {
	return p.fs.Link(oldname, newname)
}

// Open ...
func (p *PebbleFS) Open(name string, opts ...pvfs.OpenOption) (pvfs.File, error) {
	f, err := p.fs.Open(name)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt.Apply(f)
	}
	return f, nil
}

// OpenDir ...
func (p *PebbleFS) OpenDir(name string) (pvfs.File, error) {
	return p.fs.OpenDir(name)
}

// Remove ...
func (p *PebbleFS) Remove(name string) error {
	return p.fs.Remove(name)
}

// RemoveAll ...
func (p *PebbleFS) RemoveAll(name string) error {
	return p.fs.RemoveAll(name)
}

// Rename ...
func (p *PebbleFS) Rename(oldname, newname string) error {
	return p.fs.Rename(oldname, newname)
}

// ReuseForWrite ...
func (p *PebbleFS) ReuseForWrite(oldname, newname string) (pvfs.File, error) {
	return p.fs.ReuseForWrite(oldname, newname)
}

// MkdirAll ...
func (p *PebbleFS) MkdirAll(dir string, perm os.FileMode) error {
	return p.fs.MkdirAll(dir, perm)
}

// Lock ...
func (p *PebbleFS) Lock(name string) (io.Closer, error) {
	return p.fs.Lock(name)
}

// List ...
func (p *PebbleFS) List(dir string) ([]string, error) {
	return p.fs.List(dir)
}

// Stat ...
func (p *PebbleFS) Stat(name string) (os.FileInfo, error) {
	return p.fs.Stat(name)
}

// PathBase ...
func (p *PebbleFS) PathBase(path string) string {
	return p.fs.PathBase(path)
}

// PathJoin ...
func (p *PebbleFS) PathJoin(elem ...string) string {
	return p.fs.PathJoin(elem...)
}

// PathDir ...
func (p *PebbleFS) PathDir(path string) string {
	return p.fs.PathDir(path)
}
