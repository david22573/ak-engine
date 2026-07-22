package epochorchestrator

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/david22573/ak-engine/internal/partitionpipeline"
	"github.com/david22573/ak-engine/internal/qualificationrunner"
	"github.com/david22573/ak-rif/research"
)

const (
	BoundaryRepairResultVersion  = "ak.engine.pr4b0-r1p9.boundary-repair-result.v1"
	BoundaryConfigAuditVersion   = "ak.engine.pr4b0-r1p9.all-membership-boundary-audit.v1"
	PreparedDatasetSourceVersion = "ak.engine.prepared_dataset_source.v1"
)

var orderedBoundaryPartitions = []string{"DEVELOPMENT", "VALIDATION", "FINAL_HOLDOUT"}

type BoundaryRepairBaseline struct {
	ResearchID string          `json:"research_id"`
	ProtocolID string          `json:"protocol_id"`
	Engine     RepositoryCheck `json:"engine"`
	RIF        RepositoryCheck `json:"rif"`
	Historian  RepositoryCheck `json:"historian"`
}

type BoundarySemanticSnapshot struct {
	ProtocolSemanticSHA256 string                            `json:"protocol_semantic_sha256"`
	CandidateScope         research.CandidateScope           `json:"candidate_scope"`
	VariantLedger          qualificationrunner.VariantLedger `json:"variant_ledger"`
	RequiredSymbols        []string                          `json:"dataset_required_symbols"`
	CandidateTargets       []string                          `json:"candidate_target_symbols"`
	ContextOnlySymbols     []string                          `json:"context_only_symbols"`
	UniverseContractSHA256 string                            `json:"universe_contract_sha256"`
	EligibleInterval       research.Interval                 `json:"eligible_interval"`
	Partitions             []research.Partition              `json:"logical_partitions"`
	Authorities            research.AuthorityIdentity        `json:"authority_identity"`
}

type BoundarySemanticDelta struct {
	Before                    BoundarySemanticSnapshot `json:"before"`
	After                     BoundarySemanticSnapshot `json:"after"`
	SemanticDifferences       []string                 `json:"semantic_differences"`
	OnlyInfrastructureChanged bool                     `json:"only_infrastructure_changed"`
}

type BoundaryPlanRepair struct {
	Partition                       string                          `json:"partition"`
	ParentPlanSHA256                string                          `json:"parent_plan_sha256"`
	PreparedPlanSHA256              string                          `json:"prepared_plan_sha256"`
	ParentSourceIdentitySHA256      string                          `json:"parent_source_identity_sha256"`
	PreparedPartitionIdentitySHA256 string                          `json:"prepared_partition_source_identity_sha256"`
	PreparationManifestSHA256       string                          `json:"preparation_manifest_sha256"`
	Audit                           partitionpipeline.BoundaryAudit `json:"boundary_audit"`
}

type BoundaryRepairResult struct {
	SchemaVersion               string                 `json:"schema_version"`
	ParentConfigSHA256          string                 `json:"parent_config_sha256"`
	RepairedConfigSHA256        string                 `json:"repaired_config_sha256"`
	PreparedDatasetSourceSHA256 string                 `json:"prepared_dataset_source_sha256"`
	Baseline                    BoundaryRepairBaseline `json:"fresh_epoch_baseline"`
	Plans                       []BoundaryPlanRepair   `json:"ordered_plan_repairs"`
	Memberships                 int                    `json:"memberships"`
	ChildReferences             int                    `json:"child_references"`
	UniqueChildArtifacts        int                    `json:"unique_child_artifacts"`
	UnsafeMemberships           int                    `json:"unsafe_memberships"`
	OriginalADAUSDTDefectFound  bool                   `json:"original_adausdt_defect_detected"`
	OriginalADAUSDTChildSHA256  string                 `json:"original_adausdt_safe_child_sha256,omitempty"`
	OriginalADAUSDTChildRows    int                    `json:"original_adausdt_safe_child_rows,omitempty"`
	SemanticDelta               BoundarySemanticDelta  `json:"semantic_delta"`
	RealProtocolRegistrations   int                    `json:"real_protocol_registrations"`
	RealPartitionAccesses       int                    `json:"real_partition_accesses"`
	RealCandidateOutcomes       int                    `json:"real_candidate_outcomes"`
	ResultSHA256                string                 `json:"result_sha256"`
}

type BoundaryConfigAudit struct {
	SchemaVersion               string                                `json:"schema_version"`
	ConfigSHA256                string                                `json:"config_sha256"`
	PreparedDatasetSourceSHA256 string                                `json:"prepared_dataset_source_sha256"`
	Stages                      []partitionpipeline.BoundaryAudit     `json:"ordered_stage_audits"`
	Memberships                 int                                   `json:"memberships"`
	Artifacts                   int                                   `json:"artifacts"`
	Classes                     partitionpipeline.BoundaryClassCounts `json:"classes"`
	Rejected                    int                                   `json:"rejected_artifacts"`
	Missing                     int                                   `json:"missing_artifacts"`
	UnsafeMemberships           int                                   `json:"unsafe_memberships"`
	AuditSHA256                 string                                `json:"audit_sha256"`
}

func PrepareProductionBoundaryRepairConfig(parent Config, preparedRoot string, baseline BoundaryRepairBaseline) (Config, BoundaryRepairResult, error) {
	if parent.Synthetic {
		return Config{}, BoundaryRepairResult{}, errors.New("production boundary repair rejects synthetic parent configuration")
	}
	sealedParent, err := SealConfig(parent)
	if err != nil || !reflect.DeepEqual(parent, sealedParent) {
		return Config{}, BoundaryRepairResult{}, errors.New("parent production configuration is not canonically sealed")
	}
	if err := validateBoundaryRepairBaseline(baseline); err != nil {
		return Config{}, BoundaryRepairResult{}, err
	}
	if parent.RunnerBuild == nil {
		return Config{}, BoundaryRepairResult{}, errors.New("parent production runner build contract is missing")
	}
	before, err := boundarySemanticSnapshot(parent)
	if err != nil {
		return Config{}, BoundaryRepairResult{}, err
	}
	prepared := make(map[string]partitionpipeline.Plan, len(orderedBoundaryPartitions))
	manifestByPartition := make(map[string]partitionpipeline.PreparationManifest, len(orderedBoundaryPartitions))
	storeRoot, err := ensureBoundaryStoreBase(preparedRoot)
	if err != nil {
		return Config{}, BoundaryRepairResult{}, err
	}
	for _, partition := range orderedBoundaryPartitions {
		plan, ok := parent.Plans[partition]
		if !ok || plan.SchemaVersion != partitionpipeline.PlanSchemaVersion {
			return Config{}, BoundaryRepairResult{}, fmt.Errorf("%s parent plan is missing or not v1", partition)
		}
		stageRoot := filepath.Join(storeRoot, strings.ToLower(partition))
		preparedPlan, manifest, _, prepareErr := partitionpipeline.PreparePlan(plan, stageRoot)
		if prepareErr != nil {
			return Config{}, BoundaryRepairResult{}, fmt.Errorf("prepare %s: %w", partition, prepareErr)
		}
		prepared[partition] = preparedPlan
		manifestByPartition[partition] = manifest
	}
	datasetSource, err := preparedDatasetSourceIdentity(prepared)
	if err != nil {
		return Config{}, BoundaryRepairResult{}, err
	}
	for _, partition := range orderedBoundaryPartitions {
		prepared[partition], err = partitionpipeline.BindPreparedDatasetSource(prepared[partition], datasetSource)
		if err != nil {
			return Config{}, BoundaryRepairResult{}, fmt.Errorf("bind %s prepared dataset source: %w", partition, err)
		}
	}

	parentBytes, err := EncodeConfig(parent)
	if err != nil {
		return Config{}, BoundaryRepairResult{}, err
	}
	repaired, err := DecodeConfig(parentBytes)
	if err != nil {
		return Config{}, BoundaryRepairResult{}, err
	}
	repaired.ConfigSHA256 = ""
	repaired.Plans = prepared
	repaired.Repositories = map[string]RepositoryCheck{"engine": baseline.Engine, "rif": baseline.RIF, "historian": baseline.Historian}
	repaired.Identity.ResearchID = baseline.ResearchID
	repaired.Identity.Repositories.EngineStartingCommit = baseline.Engine.Commit
	repaired.Identity.Repositories.RIFStartingCommit = baseline.RIF.Commit
	repaired.Identity.Repositories.HistorianStartingCommit = baseline.Historian.Commit
	repaired.Identity.Repositories.ProtocolGitCommit = baseline.Engine.Commit
	repaired.Identity.Repositories.RunnerGitCommit = baseline.Engine.Commit
	repaired.Identity.Dataset.SourceIdentitySHA256 = datasetSource
	for index := range repaired.Identity.Partitions {
		partition := string(repaired.Identity.Partitions[index].Name)
		plan, ok := prepared[partition]
		if !ok {
			return Config{}, BoundaryRepairResult{}, errors.New("research identity contains an unexpected partition")
		}
		repaired.Identity.Partitions[index].RequiredSymbolCoverageSHA256 = plan.PlanSHA256
	}
	repaired.Protocol, repaired.Identity.Protocol, err = boundaryProtocolIdentity(parent.Protocol, parent.Identity.Protocol, baseline.ProtocolID)
	if err != nil {
		return Config{}, BoundaryRepairResult{}, err
	}
	repaired.RunnerBuild = &ProductionRunnerBuild{RepositoryPath: baseline.Engine.Path, Package: "./cmd/ak-engine", GOOS: parent.RunnerBuild.GOOS, GOARCH: parent.RunnerBuild.GOARCH}
	repaired.Runner, err = ComputeProductionRunnerIdentity(*repaired.RunnerBuild, baseline.Engine.Commit)
	if err != nil {
		return Config{}, BoundaryRepairResult{}, fmt.Errorf("compute repaired runner identity: %w", err)
	}
	repaired.Identity.Repositories.RunnerExecutableSHA256 = repaired.Runner.BinarySHA256
	repaired, err = SealConfig(repaired)
	if err != nil {
		return Config{}, BoundaryRepairResult{}, err
	}
	if err := ValidateConfigStructure(repaired); err != nil {
		return Config{}, BoundaryRepairResult{}, fmt.Errorf("validate repaired configuration without registration: %w", err)
	}
	after, err := boundarySemanticSnapshot(repaired)
	if err != nil {
		return Config{}, BoundaryRepairResult{}, err
	}
	delta := BoundarySemanticDelta{Before: before, After: after, SemanticDifferences: semanticDifferences(before, after)}
	delta.OnlyInfrastructureChanged = len(delta.SemanticDifferences) == 0
	if !delta.OnlyInfrastructureChanged {
		return Config{}, BoundaryRepairResult{}, fmt.Errorf("candidate or universe semantic drift: %s", strings.Join(delta.SemanticDifferences, ", "))
	}

	result := BoundaryRepairResult{SchemaVersion: BoundaryRepairResultVersion, ParentConfigSHA256: parent.ConfigSHA256, RepairedConfigSHA256: repaired.ConfigSHA256, PreparedDatasetSourceSHA256: datasetSource, Baseline: baseline, SemanticDelta: delta}
	children := map[string]struct{}{}
	for _, partition := range orderedBoundaryPartitions {
		plan := repaired.Plans[partition]
		audit, auditErr := partitionpipeline.AuditPreparedPlan(plan)
		if auditErr != nil {
			return Config{}, BoundaryRepairResult{}, fmt.Errorf("audit %s: %w", partition, auditErr)
		}
		manifest := manifestByPartition[partition]
		result.Plans = append(result.Plans, BoundaryPlanRepair{Partition: partition, ParentPlanSHA256: plan.ParentPlanSHA256, PreparedPlanSHA256: plan.PlanSHA256, ParentSourceIdentitySHA256: plan.ParentSourceIdentitySHA256, PreparedPartitionIdentitySHA256: plan.PreparedPartitionIdentity, PreparationManifestSHA256: manifest.ManifestSHA256, Audit: audit})
		result.Memberships += audit.Memberships
		result.ChildReferences += audit.Artifacts
		result.UnsafeMemberships += audit.UnsafeMemberships
		for _, entry := range manifest.Entries {
			children[entry.ChildSHA256] = struct{}{}
			if partition == "DEVELOPMENT" && entry.Symbol == "ADAUSDT" && entry.UTCDate == "2026-04-09" && entry.Parent.FragmentSHA256 == "sha256:cfad25fa8df42974951955c34965ab3bfa07615bfddfd0394d039a0b035d67a6" && entry.Parent.ReceiptSHA256 == "sha256:a51a972bc628d7073a80e5e9b09667b15d7bc355a75c996b3adbf913f5e1ade4" && entry.BoundaryClass == "RIGHT_CLIPPED" && entry.ChildRowCount == 560 && entry.ChildLastTimestampUTC.Before(entry.MembershipInterval.End) {
				result.OriginalADAUSDTDefectFound = true
				result.OriginalADAUSDTChildSHA256 = entry.ChildSHA256
				result.OriginalADAUSDTChildRows = entry.ChildRowCount
			}
		}
	}
	result.UniqueChildArtifacts = len(children)
	if result.Memberships != 1773 || result.UnsafeMemberships != 0 {
		return Config{}, BoundaryRepairResult{}, fmt.Errorf("real all-membership audit failed: memberships=%d unsafe=%d", result.Memberships, result.UnsafeMemberships)
	}
	if !result.OriginalADAUSDTDefectFound {
		return Config{}, BoundaryRepairResult{}, errors.New("original ADAUSDT DEVELOPMENT boundary defect was not detected and safely clipped")
	}
	result.ResultSHA256, err = boundaryRepairResultHash(result)
	return repaired, result, err
}

func ensureBoundaryStoreBase(root string) (string, error) {
	if root == "" || filepath.Clean(root) != root || !filepath.IsAbs(root) {
		return "", errors.New("prepared dataset root must be absolute and normalized")
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(root))
	if err != nil || parent != filepath.Dir(root) {
		return "", errors.New("prepared dataset root parent must be canonical and nonsymlinked")
	}
	if info, statErr := os.Lstat(root); errors.Is(statErr, os.ErrNotExist) {
		if err := os.Mkdir(root, 0o700); err != nil {
			return "", err
		}
	} else if statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return "", errors.New("prepared dataset root is unsafe")
	}
	real, err := filepath.EvalSymlinks(root)
	if err != nil || real != root {
		return "", errors.New("prepared dataset root is alternate or symlinked")
	}
	return root, nil
}

func AuditBoundaryConfig(cfg Config) (BoundaryConfigAudit, error) {
	if err := ValidateConfigStructure(cfg); err != nil {
		return BoundaryConfigAudit{}, err
	}
	audit := BoundaryConfigAudit{SchemaVersion: BoundaryConfigAuditVersion, ConfigSHA256: cfg.ConfigSHA256, PreparedDatasetSourceSHA256: cfg.Identity.Dataset.SourceIdentitySHA256}
	for _, partition := range orderedBoundaryPartitions {
		plan := cfg.Plans[partition]
		if plan.SchemaVersion != partitionpipeline.PreparedPlanSchemaVersion || plan.SourceIdentitySHA256 != audit.PreparedDatasetSourceSHA256 {
			return BoundaryConfigAudit{}, fmt.Errorf("%s is not bound to the prepared dataset source", partition)
		}
		stage, err := partitionpipeline.AuditPreparedPlan(plan)
		if err != nil {
			return BoundaryConfigAudit{}, err
		}
		audit.Stages = append(audit.Stages, stage)
		audit.Memberships += stage.Memberships
		audit.Artifacts += stage.Artifacts
		audit.Classes.Exact += stage.Classes.Exact
		audit.Classes.Left += stage.Classes.Left
		audit.Classes.Right += stage.Classes.Right
		audit.Classes.Both += stage.Classes.Both
		audit.Rejected += stage.Rejected
		audit.Missing += stage.Missing
		audit.UnsafeMemberships += stage.UnsafeMemberships
	}
	wantMemberships := 1773
	if cfg.Synthetic {
		wantMemberships = 27
	}
	if audit.Memberships != wantMemberships || audit.Rejected != 0 || audit.Missing != 0 || audit.UnsafeMemberships != 0 {
		return BoundaryConfigAudit{}, fmt.Errorf("boundary config is unsafe: memberships=%d rejected=%d missing=%d unsafe=%d", audit.Memberships, audit.Rejected, audit.Missing, audit.UnsafeMemberships)
	}
	hash, err := canonicalHash(struct {
		SchemaVersion               string                                `json:"schema_version"`
		ConfigSHA256                string                                `json:"config_sha256"`
		PreparedDatasetSourceSHA256 string                                `json:"prepared_dataset_source_sha256"`
		Stages                      []partitionpipeline.BoundaryAudit     `json:"ordered_stage_audits"`
		Memberships                 int                                   `json:"memberships"`
		Artifacts                   int                                   `json:"artifacts"`
		Classes                     partitionpipeline.BoundaryClassCounts `json:"classes"`
		Rejected                    int                                   `json:"rejected_artifacts"`
		Missing                     int                                   `json:"missing_artifacts"`
		UnsafeMemberships           int                                   `json:"unsafe_memberships"`
	}{audit.SchemaVersion, audit.ConfigSHA256, audit.PreparedDatasetSourceSHA256, audit.Stages, audit.Memberships, audit.Artifacts, audit.Classes, audit.Rejected, audit.Missing, audit.UnsafeMemberships})
	audit.AuditSHA256 = hash
	return audit, err
}

func validateBoundaryRepairBaseline(baseline BoundaryRepairBaseline) error {
	if baseline.ResearchID == "" || baseline.ProtocolID == "" || baseline.ResearchID == "pr4b0-r1-production-epoch" || baseline.ProtocolID == "pr4b0-r1-production-protocol" {
		return errors.New("fresh research and protocol identities are required")
	}
	for name, repository := range map[string]RepositoryCheck{"engine": baseline.Engine, "rif": baseline.RIF, "historian": baseline.Historian} {
		if err := verifyRepository(repository); err != nil {
			return fmt.Errorf("%s repair baseline: %w", name, err)
		}
		status, err := git(repository.Path, "status", "--porcelain")
		if err != nil || status != "" {
			return fmt.Errorf("%s repair baseline is not clean", name)
		}
	}
	return nil
}

func preparedDatasetSourceIdentity(plans map[string]partitionpipeline.Plan) (string, error) {
	identities := make([]partitionpipeline.HashIdentity, 0, len(orderedBoundaryPartitions))
	for _, partition := range orderedBoundaryPartitions {
		plan, ok := plans[partition]
		if !ok || plan.PreparedPartitionIdentity == "" {
			return "", fmt.Errorf("%s prepared partition identity is missing", partition)
		}
		identities = append(identities, partitionpipeline.HashIdentity{ID: partition, SHA256: plan.PreparedPartitionIdentity})
	}
	return canonicalHash(struct {
		SchemaVersion string                           `json:"schema_version"`
		Partitions    []partitionpipeline.HashIdentity `json:"ordered_partition_source_identities"`
	}{PreparedDatasetSourceVersion, identities})
}

func boundaryProtocolIdentity(raw json.RawMessage, identity research.ProtocolIdentity, protocolID string) (json.RawMessage, research.ProtocolIdentity, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, research.ProtocolIdentity{}, err
	}
	id, _ := json.Marshal(protocolID)
	object["protocol_id"] = id
	encoded, err := json.Marshal(object)
	if err != nil {
		return nil, research.ProtocolIdentity{}, err
	}
	hash := byteHash(encoded)
	identity.ID = protocolID
	identity.SHA256 = hash
	identity.ContentAddressedIdentity = hash
	return encoded, identity, nil
}

func boundarySemanticSnapshot(cfg Config) (BoundarySemanticSnapshot, error) {
	var protocol map[string]json.RawMessage
	if err := json.Unmarshal(cfg.Protocol, &protocol); err != nil {
		return BoundarySemanticSnapshot{}, err
	}
	delete(protocol, "protocol_id")
	protocolHash, err := canonicalHash(protocol)
	if err != nil {
		return BoundarySemanticSnapshot{}, err
	}
	partitions := append([]research.Partition(nil), cfg.Identity.Partitions...)
	for index := range partitions {
		partitions[index].RequiredSymbolCoverageSHA256 = ""
	}
	sort.Slice(partitions, func(i, j int) bool { return partitions[i].Name < partitions[j].Name })
	return BoundarySemanticSnapshot{ProtocolSemanticSHA256: protocolHash, CandidateScope: cfg.Identity.CandidateScope, VariantLedger: cfg.VariantLedger, RequiredSymbols: append([]string(nil), cfg.Identity.Dataset.RequiredSymbols...), CandidateTargets: append([]string(nil), cfg.Identity.Dataset.CandidateTargetSymbols...), ContextOnlySymbols: append([]string(nil), cfg.Identity.Dataset.ContextOnlySymbols...), UniverseContractSHA256: cfg.Identity.Dataset.UniverseContractSHA256, EligibleInterval: cfg.Identity.Dataset.EligibleInterval, Partitions: partitions, Authorities: cfg.Identity.Authorities}, nil
}

func semanticDifferences(before, after BoundarySemanticSnapshot) []string {
	if reflect.DeepEqual(before, after) {
		return []string{}
	}
	differences := []string{}
	checks := []struct {
		name  string
		equal bool
	}{{"protocol_semantics", before.ProtocolSemanticSHA256 == after.ProtocolSemanticSHA256}, {"candidate_scope", reflect.DeepEqual(before.CandidateScope, after.CandidateScope)}, {"variant_ledger", reflect.DeepEqual(before.VariantLedger, after.VariantLedger)}, {"symbols_and_roles", reflect.DeepEqual(before.RequiredSymbols, after.RequiredSymbols) && reflect.DeepEqual(before.CandidateTargets, after.CandidateTargets) && reflect.DeepEqual(before.ContextOnlySymbols, after.ContextOnlySymbols)}, {"universe_contract", before.UniverseContractSHA256 == after.UniverseContractSHA256}, {"eligible_interval", reflect.DeepEqual(before.EligibleInterval, after.EligibleInterval)}, {"partition_windows", reflect.DeepEqual(before.Partitions, after.Partitions)}, {"authorities", reflect.DeepEqual(before.Authorities, after.Authorities)}}
	for _, check := range checks {
		if !check.equal {
			differences = append(differences, check.name)
		}
	}
	return differences
}

func boundaryRepairResultHash(result BoundaryRepairResult) (string, error) {
	result.ResultSHA256 = ""
	return canonicalHash(result)
}
