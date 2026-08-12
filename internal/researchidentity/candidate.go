package researchidentity

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/david22573/ak-engine/internal/canonicalcontract"
)

const (
	candidateRegistryID            = "ak.engine.deep-candidate-registry"
	candidateRegistryVersion       = "1"
	candidateRegistrySchemaVersion = 1
	candidateInventoryEncoding     = "ak.canonical.json.v1"
)

type ImplementationFile struct {
	Path            string
	InclusionReason string
}

type Registration struct {
	CandidateID           string
	CandidateVersion      string
	CandidateType         string
	Family                string
	Side                  string
	Aliases               []string
	ImplementationLocator string
	ImplementationFiles   []ImplementationFile
	UsesRegimes           bool
}

type Registry struct {
	registryID      string
	registryVersion string
	entriesByKey    map[string]Registration
}

func NewRegistry(registryID, registryVersion string, entries []Registration) (*Registry, error) {
	if strings.TrimSpace(registryID) == "" || strings.TrimSpace(registryVersion) == "" {
		return nil, fmt.Errorf("registry id and version are required")
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("candidate registry is empty")
	}
	registry := &Registry{registryID: registryID, registryVersion: registryVersion, entriesByKey: map[string]Registration{}}
	ids := map[string]struct{}{}
	for _, entry := range entries {
		if strings.TrimSpace(entry.CandidateID) == "" || strings.TrimSpace(entry.CandidateVersion) == "" || strings.TrimSpace(entry.CandidateType) == "" || strings.TrimSpace(entry.Family) == "" || strings.TrimSpace(entry.Side) == "" || strings.TrimSpace(entry.ImplementationLocator) == "" {
			return nil, fmt.Errorf("candidate registration is incomplete")
		}
		if len(entry.ImplementationFiles) == 0 {
			return nil, fmt.Errorf("candidate %s has no implementation files", entry.CandidateID)
		}
		if _, exists := ids[entry.CandidateID]; exists {
			return nil, fmt.Errorf("duplicate candidate id %q", entry.CandidateID)
		}
		ids[entry.CandidateID] = struct{}{}
		keys := append([]string{candidateLookupKey(entry.Family, entry.Side)}, normalizedAliasKeys(entry.Aliases, entry.Side)...)
		for _, key := range keys {
			if key == "" {
				return nil, fmt.Errorf("candidate %s has an empty lookup key", entry.CandidateID)
			}
			if prior, exists := registry.entriesByKey[key]; exists {
				return nil, fmt.Errorf("duplicate candidate registration key %q for %s and %s", key, prior.CandidateID, entry.CandidateID)
			}
			registry.entriesByKey[key] = entry
		}
	}
	return registry, nil
}

func DefaultRegistry() (*Registry, error) {
	families := []struct {
		name  string
		alias string
	}{
		{name: "CompressionBreakout", alias: "compression_breakout"},
		{name: "ShockFade", alias: "shock_fade"},
		{name: "TrendContinuation", alias: "trend_continuation"},
		{name: "VolumeMomentum", alias: "volume_momentum"},
		{name: "BetaAgrees", alias: "beta_agrees"},
		{name: "BetaDiverges", alias: "beta_diverges"},
	}
	entries := make([]Registration, 0, len(families)*2)
	for _, family := range families {
		for _, side := range []string{"LONG", "SHORT"} {
			entries = append(entries, Registration{
				CandidateID:           fmt.Sprintf("ak.engine.deep.%s.%s", strings.ReplaceAll(family.alias, "_", "-"), strings.ToLower(side)),
				CandidateVersion:      "1.0.0",
				CandidateType:         "deep_research_strategy",
				Family:                family.name,
				Side:                  side,
				Aliases:               []string{family.alias},
				ImplementationLocator: "path:internal/app/evaluate_candidate_deep.go#deepCandidateRule",
				ImplementationFiles: []ImplementationFile{
					{
						Path:            "internal/app/evaluate_candidate_deep.go",
						InclusionReason: "candidate rules, evaluation series, metrics, gates, and classification",
					},
					{
						Path:            "internal/temporal/contract.go",
						InclusionReason: "canonical source availability and decision-time ordering",
					},
					{
						Path:            "internal/executionseries/spec.go",
						InclusionReason: "canonical fill, range, and horizon semantics",
					},
				},
				UsesRegimes: true,
			})
		}
	}
	return NewRegistry(candidateRegistryID, candidateRegistryVersion, entries)
}

func (r *Registry) Resolve(repositoryRoot, family, side string) (CandidateIdentity, error) {
	if r == nil {
		return CandidateIdentity{}, fmt.Errorf("candidate registry is nil")
	}
	entry, ok := r.entriesByKey[candidateLookupKey(family, side)]
	if !ok {
		return CandidateIdentity{}, fmt.Errorf("candidate %q/%q is not registered", family, side)
	}
	implementation, err := buildImplementationIdentity(repositoryRoot, entry.ImplementationFiles)
	if err != nil {
		return CandidateIdentity{}, err
	}
	aliases := append([]string{}, entry.Aliases...)
	for index := range aliases {
		aliases[index] = strings.ToLower(strings.TrimSpace(aliases[index]))
	}
	sort.Strings(aliases)
	identity := CandidateIdentity{
		Contract:              canonicalcontract.NewHeader(candidateRegistrationSchemaName, canonicalContractVersion, candidateRegistrationRole),
		CandidateID:           entry.CandidateID,
		RegistryID:            r.registryID,
		RegistryVersion:       r.registryVersion,
		RegistrySchemaVersion: candidateRegistrySchemaVersion,
		CandidateVersion:      entry.CandidateVersion,
		CandidateType:         entry.CandidateType,
		Family:                entry.Family,
		Side:                  strings.ToUpper(entry.Side),
		Aliases:               aliases,
		ImplementationLocator: entry.ImplementationLocator,
		UsesRegimes:           entry.UsesRegimes,
		Implementation:        implementation,
	}
	identity.ArtifactHash, err = artifactHash(candidateRegistrationSchemaName, candidateRegistrationRole, identity)
	if err != nil {
		return CandidateIdentity{}, err
	}
	identity.RegistrationRecordHash = identity.ArtifactHash
	return identity, nil
}

// Lookup returns the immutable registration selected by a canonical family or
// approved alias. Callers use its canonical family/side to ensure the evaluated
// rule is exactly the one whose registration is hashed.
func (r *Registry) Lookup(family, side string) (Registration, error) {
	if r == nil {
		return Registration{}, fmt.Errorf("candidate registry is nil")
	}
	entry, ok := r.entriesByKey[candidateLookupKey(family, side)]
	if !ok {
		return Registration{}, fmt.Errorf("candidate %q/%q is not registered", family, side)
	}
	entry.Aliases = append([]string(nil), entry.Aliases...)
	entry.ImplementationFiles = append([]ImplementationFile(nil), entry.ImplementationFiles...)
	return entry, nil
}

func buildImplementationIdentity(repositoryRoot string, files []ImplementationFile) (CandidateImplementationIdentity, error) {
	return buildImplementationIdentityForRole(repositoryRoot, files, "candidate_source")
}

func buildImplementationIdentityForRole(repositoryRoot string, files []ImplementationFile, rawRole string) (CandidateImplementationIdentity, error) {
	rootAbs, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return CandidateImplementationIdentity{}, err
	}
	seen := map[string]struct{}{}
	entries := make([]FileInventoryEntry, 0, len(files))
	for _, requested := range files {
		path, err := normalizeRepositoryPath(requested.Path)
		if err != nil {
			return CandidateImplementationIdentity{}, err
		}
		if _, exists := seen[path]; exists {
			return CandidateImplementationIdentity{}, fmt.Errorf("duplicate implementation path %q", path)
		}
		seen[path] = struct{}{}
		absolute := filepath.Join(rootAbs, filepath.FromSlash(path))
		resolved, err := filepath.EvalSymlinks(absolute)
		if err != nil {
			return CandidateImplementationIdentity{}, fmt.Errorf("implementation file %s: %w", path, err)
		}
		resolvedRelative, err := filepath.Rel(rootAbs, resolved)
		if err != nil || filepath.IsAbs(resolvedRelative) || resolvedRelative == ".." || strings.HasPrefix(resolvedRelative, ".."+string(filepath.Separator)) {
			return CandidateImplementationIdentity{}, fmt.Errorf("implementation symlink escapes repository root: %s", path)
		}
		if resolved != absolute {
			return CandidateImplementationIdentity{}, fmt.Errorf("implementation path must be a regular file, not a symlink: %s", path)
		}
		hash, size, err := hashFileRole(absolute, rawRole)
		if err != nil {
			return CandidateImplementationIdentity{}, fmt.Errorf("implementation file %s: %w", path, err)
		}
		entries = append(entries, FileInventoryEntry{Path: path, SizeBytes: size, SHA256: hash, ModeClass: "regular", InclusionReason: requested.InclusionReason})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	identity := CandidateImplementationIdentity{
		Contract:          canonicalcontract.NewHeader(candidateImplementationSchemaName, canonicalContractVersion, candidateImplementationRole),
		Files:             entries,
		InventoryEncoding: candidateInventoryEncoding,
	}
	hash, err := artifactHash(candidateImplementationSchemaName, candidateImplementationRole, identity)
	if err != nil {
		return CandidateImplementationIdentity{}, err
	}
	identity.ArtifactHash = hash
	identity.ImplementationHash = hash
	return identity, nil
}

func normalizeRepositoryPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" || filepath.IsAbs(path) || strings.Contains(path, "\\") {
		return "", fmt.Errorf("invalid repository-relative path %q", path)
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("path traversal is not permitted: %q", path)
	}
	return clean, nil
}

func candidateLookupKey(family, side string) string {
	return strings.ToLower(strings.TrimSpace(family)) + "\x00" + strings.ToUpper(strings.TrimSpace(side))
}

func normalizedAliasKeys(aliases []string, side string) []string {
	keys := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		keys = append(keys, candidateLookupKey(alias, side))
	}
	return keys
}
