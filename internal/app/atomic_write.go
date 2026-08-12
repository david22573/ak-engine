package app

import (
	"encoding/json"
	"os"

	"github.com/david22573/ak-engine/internal/atomicfile"
)

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	return atomicfile.WriteFile(path, data, perm)
}

func atomicWriteJSONFile(path string, value interface{}, prefix, indent string, perm os.FileMode) error {
	data, err := json.MarshalIndent(value, prefix, indent)
	if err != nil {
		return err
	}
	return atomicWriteFile(path, data, perm)
}
