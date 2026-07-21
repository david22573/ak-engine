package persistence

import (
	"errors"
	"os"
	"syscall"

	"github.com/david22573/ak-rif/core"
)

// OpenRegularFile opens an authoritative input without following a final
// symbolic link and rejects non-regular files.
func OpenRegularFile(path, operation string) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return nil, core.NewAuthorityError(operation, core.CodeUnsafePath, "path", "symbolic-link authoritative inputs are not allowed", err)
		}
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, core.NewAuthorityError(operation, core.CodeArtifactInvalid, "path", "authoritative file cannot be inspected", err)
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, core.NewAuthorityError(operation, core.CodeUnsafePath, "path", "authoritative input must be a regular file", nil)
	}
	return file, nil
}
