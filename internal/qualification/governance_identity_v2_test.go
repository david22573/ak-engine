package qualification

import "testing"

func TestSyntheticGovernanceHashesBindFrozenCandidateLifecycleV2(t *testing.T) {
	descriptor := validFrozenDescriptor(t)
	descriptor.SchemaVersion = FrozenDescriptorSchemaVersionV2
	descriptor.IndependencePolicyHash = sha('a')
	descriptor.UncertaintyMethodHash = sha('b')
	descriptor.SourceSchemaHash = sha('c')
	descriptor.ManifestContractHash = sha('d')
	descriptor.DescriptorHash = ""
	descriptor.DescriptorHash = mustDescriptorHashV2(t, descriptor)
	if err := descriptor.Verify(); err != nil {
		t.Fatalf("V2 frozen descriptor failed: %v", err)
	}

	request := validRegistrationRequest(t)
	request.SchemaVersion = RegistrationRequestSchemaVersionV2
	request.FrozenCandidate = descriptor
	request.ResearchIdentity.SchemaVersion = ResearchIdentitySchemaVersionV2
	request.ResearchIdentity.IndependencePolicyHash = descriptor.IndependencePolicyHash
	request.ResearchIdentity.UncertaintyMethodHash = descriptor.UncertaintyMethodHash
	request.ResearchIdentity.SourceSchemaHash = descriptor.SourceSchemaHash
	request.ResearchIdentity.ManifestContractHash = descriptor.ManifestContractHash
	request.ArtifactIntegrityHash = ""
	request.ArtifactIntegrityHash = mustRegistrationHashV2(t, request)
	if err := request.Verify(); err != nil {
		t.Fatalf("V2 synthetic lifecycle request failed: %v", err)
	}

	mutated := request
	mutated.ResearchIdentity.ManifestContractHash = sha('e')
	if err := mutated.Verify(); err == nil {
		t.Fatal("governance identity mutation did not invalidate frozen lifecycle binding")
	}
	missing := descriptor
	missing.SourceSchemaHash = ""
	missing.DescriptorHash = ""
	missing.DescriptorHash = mustDescriptorHashV2(t, missing)
	if err := missing.Verify(); err == nil {
		t.Fatal("missing governance hash passed V2 frozen descriptor validation")
	}
}

func TestV1FrozenIdentityRejectsV2GovernanceFields(t *testing.T) {
	descriptor := validFrozenDescriptor(t)
	descriptor.IndependencePolicyHash = sha('a')
	descriptor.DescriptorHash = ""
	descriptor.DescriptorHash = mustDescriptorHashV2(t, descriptor)
	if err := descriptor.Verify(); err == nil {
		t.Fatal("V1 frozen identity silently accepted a V2 governance hash")
	}
}

func mustDescriptorHashV2(t *testing.T, descriptor FrozenCandidateDescriptor) string {
	t.Helper()
	hash, err := descriptor.CanonicalHash()
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func mustRegistrationHashV2(t *testing.T, request CandidateRegistrationRequest) string {
	t.Helper()
	hash, err := request.CanonicalHash()
	if err != nil {
		t.Fatal(err)
	}
	return hash
}
