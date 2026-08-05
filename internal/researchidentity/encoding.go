package researchidentity

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
)

const hashPrefix = "sha256:"

func hashFileRole(path, role string) (string, int64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", 0, err
	}
	if !info.Mode().IsRegular() {
		return "", 0, fmt.Errorf("%s is not a regular file", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	hash, err := canonicalcontract.HashRawReader(rawObjectContractName, canonicalContractVersion, role, info.Size(), f)
	if err != nil {
		return "", 0, err
	}
	return hash, info.Size(), nil
}

func validHash(value string) bool {
	if len(value) != len(hashPrefix)+64 || value[:len(hashPrefix)] != hashPrefix {
		return false
	}
	_, err := hex.DecodeString(value[len(hashPrefix):])
	return err == nil && value != hashPrefix+string(bytes.Repeat([]byte{'0'}, 64))
}
