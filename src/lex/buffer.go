package lex

import (
	"os"
	"path/filepath"
	"unsafe"

	"github.com/julelang/jule"
)

// File instance of fs.
type File struct {
	_path string
}

// NewFile returns new File points to Jule file.
func NewFile(path string) *File { return &File{path} }

// IsJule reports file path is Jule source code.
// Returns false if error occur.
func (f *File) IsJule() bool {
	abs, err := filepath.Abs(f._path)
	return err == nil && filepath.Ext(abs) == jule.SRC_EXT
}

// IsOk reports file path is exist and accessible or not.
func (f *File) IsOk() bool {
	_, err := os.Stat(f._path)
	return err == nil
}

// Path returns full path.
func (f *File) Path() string { return f._path }

// Dir returns directory.
func (f *File) Dir() string { return filepath.Dir(f._path) }

// Name returns filename.
func (f *File) Name() string { return filepath.Base(f._path) }

// Addr returns uintptr(unsafe.Pointer(f)).
func (f *File) Addr() uintptr { return uintptr(unsafe.Pointer(f)) }