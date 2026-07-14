package qualification

import (
	"strings"
	"testing"

	"github.com/david22573/ak-engine/internal/preconditions"
)

func TestSyntheticAcceptedV3GovernanceBindsEngineRegistration(t *testing.T) {
	descriptor := validFrozenDescriptor(t)
	descriptor.SchemaVersion = FrozenDescriptorSchemaVersionV3
	descriptor.CandidateID = preconditions.GovernedCandidateFamily
	descriptor.IndependencePolicyHash = acceptedPolicyHashV3(t)
	descriptor.UncertaintyMethodHash = preconditions.AcceptedUncertaintyMethodDigestV2
	descriptor.SourceSchemaHash = sha('c')
	descriptor.ManifestContractHash = sha('d')
	descriptor.GovernanceDecisionHash = preconditions.DefaultConcentrationGovernanceDecisionV3().CanonicalDecisionHash
	descriptor.DescriptorHash = ""
	descriptor.DescriptorHash = mustDescriptorHashV2(t, descriptor)
	if err := descriptor.Verify(); err != nil {
		t.Fatalf("V3 descriptor: %v", err)
	}

	request := validRegistrationRequest(t)
	request.SchemaVersion = RegistrationRequestSchemaVersionV3
	request.FrozenCandidate = descriptor
	request.CandidateImplementationIdentity.CandidateID = descriptor.CandidateID
	request.ResearchIdentity = ResearchIdentity{
		SchemaVersion: ResearchIdentitySchemaVersionV3, DatasetID: descriptor.DatasetID, DatasetVersion: descriptor.DatasetVersion,
		ResearchWindowStart: descriptor.ResearchWindowStart, ResearchWindowEnd: descriptor.ResearchWindowEnd, EvaluationCutoff: descriptor.EvaluationCutoff,
		ManifestID: descriptor.ManifestID, ManifestHash: descriptor.ManifestHash, AvailabilityPolicyVersion: descriptor.AvailabilityPolicyVersion, CoveragePolicyVersion: descriptor.CoveragePolicyVersion,
		IndependencePolicyHash: descriptor.IndependencePolicyHash, UncertaintyMethodHash: descriptor.UncertaintyMethodHash,
		SourceSchemaHash: descriptor.SourceSchemaHash, ManifestContractHash: descriptor.ManifestContractHash, GovernanceDecisionHash: descriptor.GovernanceDecisionHash,
	}
	request.ArtifactIntegrityHash = ""
	request.ArtifactIntegrityHash = mustRegistrationHashV2(t, request)
	if err := request.Verify(); err != nil {
		t.Fatalf("accepted V3 registration failed: %v", err)
	}

	wrongDecision := request
	wrongDecision.ResearchIdentity.GovernanceDecisionHash = sha('e')
	wrongDecision.ArtifactIntegrityHash = ""
	wrongDecision.ArtifactIntegrityHash = mustRegistrationHashV2(t, wrongDecision)
	if err := wrongDecision.Verify(); err == nil || !strings.Contains(err.Error(), "governance-decision") {
		t.Fatalf("wrong governance decision passed: %v", err)
	}

	wrongPolicy := request
	wrongPolicy.ResearchIdentity.IndependencePolicyHash = sha('f')
	wrongPolicy.ArtifactIntegrityHash = ""
	wrongPolicy.ArtifactIntegrityHash = mustRegistrationHashV2(t, wrongPolicy)
	if err := wrongPolicy.Verify(); err == nil || !strings.Contains(err.Error(), "independence-policy") {
		t.Fatalf("unknown policy passed: %v", err)
	}
}

func TestEngineV2CannotMasqueradeAsAcceptedV3(t *testing.T) {
	descriptor := validFrozenDescriptor(t)
	descriptor.SchemaVersion = FrozenDescriptorSchemaVersionV2
	descriptor.IndependencePolicyHash = acceptedPolicyHashV3(t)
	descriptor.UncertaintyMethodHash = preconditions.AcceptedUncertaintyMethodDigestV2
	descriptor.SourceSchemaHash = sha('c')
	descriptor.ManifestContractHash = sha('d')
	descriptor.DescriptorHash = ""
	descriptor.DescriptorHash = mustDescriptorHashV2(t, descriptor)
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
	if err := request.Verify(); err == nil || !strings.Contains(err.Error(), "V2 registration remains pending") {
		t.Fatalf("V2 masquerade passed: %v", err)
	}
}

func acceptedPolicyHashV3(t *testing.T) string {
	t.Helper()
	hash, err := preconditions.AcceptedIndependencePolicyHashV3(preconditions.AcceptedIndependencePolicyV3Default())
	if err != nil {
		t.Fatal(err)
	}
	return hash
}
