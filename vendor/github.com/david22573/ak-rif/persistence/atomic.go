// Package persistence provides the only authoritative file-write primitive
// used by RIF control state and artifacts.
package persistence

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/david22573/ak-rif/core"
)

// AtomicWriter writes complete files by durable same-directory replacement.
// The function fields are intentionally injectable for failure-path tests.
type AtomicWriter struct {
	CreateTemp func(string, string) (*os.File, error)
	Rename     func(string, string) error
	SyncDir    func(*os.File) error
}

// DefaultAtomicWriter uses the host filesystem.
func DefaultAtomicWriter() AtomicWriter {
	return AtomicWriter{
		CreateTemp: os.CreateTemp,
		Rename:     os.Rename,
		SyncDir:    func(dir *os.File) error { return dir.Sync() },
	}
}

// WriteFileAtomic writes data without exposing a partial authoritative file.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	return DefaultAtomicWriter().WriteFile(path, data, perm)
}

// WriteFile performs a durable atomic replacement in the destination directory.
func (w AtomicWriter) WriteFile(path string, data []byte, perm os.FileMode) (retErr error) {
	if err := validateDestination(path); err != nil {
		return err
	}
	if w.CreateTemp == nil || w.Rename == nil || w.SyncDir == nil {
		return atomicError("atomic writer is not fully configured", nil)
	}

	dir := filepath.Dir(path)
	tmp, err := w.CreateTemp(dir, ".rif-tmp-*")
	if err != nil {
		return atomicError("create temporary file", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := writeAll(tmp, data); err != nil {
		return atomicError("write temporary file", err)
	}
	if err := tmp.Chmod(perm.Perm()); err != nil {
		return atomicError("set temporary file permissions", err)
	}
	if err := tmp.Sync(); err != nil {
		return atomicError("flush temporary file", err)
	}
	if err := tmp.Close(); err != nil {
		return atomicError("close temporary file", err)
	}
	if err := w.Rename(tmpName, path); err != nil {
		return atomicError("replace destination", err)
	}

	dirHandle, err := os.Open(dir)
	if err != nil {
		return atomicError("open destination directory", err)
	}
	defer dirHandle.Close()
	if err := w.SyncDir(dirHandle); err != nil && !errors.Is(err, os.ErrInvalid) {
		return atomicError("sync destination directory", err)
	}
	return nil
}

func writeAll(w io.Writer, data []byte) (int, error) {
	written := 0
	for written < len(data) {
		n, err := w.Write(data[written:])
		written += n
		if err != nil {
			return written, err
		}
		if n == 0 {
			return written, io.ErrShortWrite
		}
	}
	return written, nil
}

func validateDestination(path string) error {
	if path == "" || filepath.Base(path) == "." || filepath.Base(path) == string(filepath.Separator) {
		return core.NewAuthorityError("atomic_write", core.CodeUnsafePath, "path", "destination path is invalid", nil)
	}
	clean := filepath.Clean(path)
	if clean != path {
		return core.NewAuthorityError("atomic_write", core.CodeUnsafePath, "path", "destination path must be normalized", nil)
	}
	for _, component := range strings.Split(path, string(filepath.Separator)) {
		if component == ".." {
			return core.NewAuthorityError("atomic_write", core.CodeUnsafePath, "path", "parent-directory traversal is not allowed", nil)
		}
	}
	absDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return core.NewAuthorityError("atomic_write", core.CodeUnsafePath, "path", "destination directory cannot be resolved", err)
	}
	current := string(filepath.Separator)
	for _, component := range strings.Split(strings.TrimPrefix(absDir, string(filepath.Separator)), string(filepath.Separator)) {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		if info, err := os.Lstat(current); err != nil {
			return core.NewAuthorityError("atomic_write", core.CodeUnsafePath, "path", "destination path component cannot be inspected", err)
		} else if info.Mode()&os.ModeSymlink != 0 {
			return core.NewAuthorityError("atomic_write", core.CodeUnsafePath, "path", "symbolic-link path components are not allowed", nil)
		}
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return core.NewAuthorityError("atomic_write", core.CodeUnsafePath, "path", "symbolic-link destinations are not allowed", nil)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return core.NewAuthorityError("atomic_write", core.CodeUnsafePath, "path", "destination cannot be inspected", err)
	}
	if info, err := os.Stat(filepath.Dir(path)); err != nil {
		return core.NewAuthorityError("atomic_write", core.CodeUnsafePath, "path", "destination directory is unavailable", err)
	} else if !info.IsDir() {
		return core.NewAuthorityError("atomic_write", core.CodeUnsafePath, "path", "destination parent is not a directory", nil)
	}
	return nil
}

func atomicError(action string, cause error) error {
	return core.NewAuthorityError("atomic_write", core.CodeAtomicWriteFailed, "", fmt.Sprintf("%s failed", action), cause)
}
