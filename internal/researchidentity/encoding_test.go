package researchidentity

import (
	"testing"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
)

func TestRawHashIsStableAndRoleBound(t *testing.T) {
	payload := []byte("canonical raw hash vector")
	first, err := canonicalcontract.HashRaw(rawObjectContractName, canonicalContractVersion, "test_vector", payload)
	if err != nil {
		t.Fatal(err)
	}
	second, err := canonicalcontract.HashRaw(rawObjectContractName, canonicalContractVersion, "test_vector", payload)
	if err != nil {
		t.Fatal(err)
	}
	otherRole, err := canonicalcontract.HashRaw(rawObjectContractName, canonicalContractVersion, "other_vector", payload)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first == otherRole || !validHash(first) {
		t.Fatalf("raw role binding failed: first=%s second=%s other=%s", first, second, otherRole)
	}
}
