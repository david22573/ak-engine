package governance

import (
	"testing"
)

//go:generate go run github.com/david22573/ak-ops/tools/contractgen --write

func TestClaimsSynchronization(t *testing.T) {
	claims := AllClaims()
	if len(claims) == 0 {
		t.Fatal("AllClaims() returned empty list")
	}

	clm1, ok := LookupClaim("CLM-001")
	if !ok {
		t.Fatal("LookupClaim(CLM-001) not found")
	}
	if clm1.Classification != "CODE_BACKED_AND_TESTED" {
		t.Errorf("CLM-001 classification = %q, want CODE_BACKED_AND_TESTED", clm1.Classification)
	}

	fields := AllFieldSpecs()
	if len(fields) == 0 {
		t.Fatal("AllFieldSpecs() returned empty list")
	}
	fSpec, ok := LookupFieldSpec("schema_name")
	if !ok {
		t.Fatal("LookupFieldSpec(schema_name) not found")
	}
	if !fSpec.Required {
		t.Errorf("schema_name required = %t, want true", fSpec.Required)
	}

	vectors := AllGoldenVectors()
	if len(vectors) == 0 {
		t.Fatal("AllGoldenVectors() returned empty list")
	}
	v1, ok := LookupGoldenVector("V001")
	if !ok {
		t.Fatal("LookupGoldenVector(V001) not found")
	}
	if v1.Class != "POSITIVE" {
		t.Errorf("V001 class = %q, want POSITIVE", v1.Class)
	}
}
