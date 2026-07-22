package partitionpipeline

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/david22573/ak-engine/internal/qualificationrunner"
)

// PreparePlan is the trusted, pre-registration data-plane boundary. It reads
// authenticated parent fragments and emits a runtime plan that contains only
// stage-contained child artifacts. It never accepts candidate or result data.
func PreparePlan(parent Plan, preparedRoot string) (Plan, PreparationManifest, BoundaryAudit, error) {
	if parent.SchemaVersion != PlanSchemaVersion {
		return Plan{}, PreparationManifest{}, BoundaryAudit{}, errors.New("preparation requires an authenticated v1 parent plan")
	}
	if err := VerifyPlan(parent); err != nil {
		return Plan{}, PreparationManifest{}, BoundaryAudit{}, fmt.Errorf("verify parent plan: %w", err)
	}
	root, err := ensurePreparedRoot(preparedRoot)
	if err != nil {
		return Plan{}, PreparationManifest{}, BoundaryAudit{}, err
	}

	preparedMembers := make([]SourceManifest, 0, len(parent.SourceManifests))
	entries := []PreparationManifestEntry{}
	classCounts := BoundaryClassCounts{}
	symbols := map[string]*BoundarySymbolSummary{}
	for _, member := range parent.SourceManifests {
		membership, err := sourceMembershipInterval(parent, member)
		if err != nil {
			return Plan{}, PreparationManifest{}, BoundaryAudit{}, err
		}
		summary := symbols[member.Symbol]
		if summary == nil {
			summary = &BoundarySymbolSummary{Symbol: member.Symbol}
			symbols[member.Symbol] = summary
		}
		summary.Memberships++
		preparedMember := SourceManifest{
			Symbol:             member.Symbol,
			UTCDate:            member.UTCDate,
			FileSHA256:         member.FileSHA256,
			PartitionSHA256:    member.PartitionSHA256,
			ExpectedRows:       member.ExpectedRows,
			MembershipInterval: &membership,
		}
		allRows := []normalizedRecord{}
		for artifactIndex, parentRef := range member.FragmentArtifacts {
			parentFragment, err := readFragment(parent, parentRef)
			if err != nil {
				return Plan{}, PreparationManifest{}, BoundaryAudit{}, fmt.Errorf("read parent fragment %s/%s/%d: %w", member.UTCDate, member.Symbol, artifactIndex, err)
			}
			parentIdentity, err := authenticatedParentIdentity(parent, member, artifactIndex, parentFragment)
			if err != nil {
				return Plan{}, PreparationManifest{}, BoundaryAudit{}, fmt.Errorf("authenticate parent fragment %s/%s/%d: %w", member.UTCDate, member.Symbol, artifactIndex, err)
			}
			if err := validateParentFragment(parentFragment, parentIdentity, member.Symbol, !parent.SyntheticFixture); err != nil {
				return Plan{}, PreparationManifest{}, BoundaryAudit{}, fmt.Errorf("validate parent fragment %s/%s/%d: %w", member.UTCDate, member.Symbol, artifactIndex, err)
			}
			child, class, err := sliceChild(parentFragment, parentIdentity, member.Symbol, membership, !parent.SyntheticFixture)
			if err != nil {
				return Plan{}, PreparationManifest{}, BoundaryAudit{}, fmt.Errorf("slice parent fragment %s/%s/%d: %w", member.UTCDate, member.Symbol, artifactIndex, err)
			}
			encoded, err := encodeChildArtifact(child)
			if err != nil {
				return Plan{}, PreparationManifest{}, BoundaryAudit{}, err
			}
			storedHash := byteHash(encoded)
			relative := filepath.ToSlash(filepath.Join("children", hashLeaf(child.ChildSHA256)+".json.gz"))
			path, err := preparedWritePath(root, relative)
			if err != nil {
				return Plan{}, PreparationManifest{}, BoundaryAudit{}, err
			}
			if err := writeContentAddressed(path, encoded, 0o400); err != nil {
				return Plan{}, PreparationManifest{}, BoundaryAudit{}, err
			}
			first := time.UnixMilli(child.Records[0].OpenTimeMS).UTC()
			last := time.UnixMilli(child.Records[len(child.Records)-1].OpenTimeMS).UTC()
			start, end := child.AuthorizedInterval.Start.UTC(), child.AuthorizedInterval.End.UTC()
			ref := SourceArtifact{
				SourceRootID:           PreparedSourceRootID,
				RelativePath:           relative,
				CanonicalSHA256:        child.ChildSHA256,
				Encoding:               PreparedFragmentEncoding,
				ReceiptSHA256:          child.Parent.ReceiptSHA256,
				ObservedAvailableAtUTC: child.Parent.ObservedAvailableAtUTC,
				StoredFileSHA256:       storedHash,
				ParentSourceRootID:     child.Parent.SourceRootID,
				ParentReceiptSHA256:    child.Parent.ReceiptSHA256,
				ParentFragmentSHA256:   child.Parent.FragmentSHA256,
				ParentProvenanceSHA256: child.Parent.ProvenanceSHA256,
				AuthorizedStartUTC:     &start,
				AuthorizedEndUTC:       &end,
				ChildRowCount:          len(child.Records),
				ChildFirstTimestampUTC: &first,
				ChildLastTimestampUTC:  &last,
				TransformationVersion:  BoundaryTransformationVersion,
				BoundaryClass:          class,
			}
			preparedMember.FragmentArtifacts = append(preparedMember.FragmentArtifacts, ref)
			allRows = append(allRows, child.Records...)
			entry := PreparationManifestEntry{
				Partition:              parent.PartitionName,
				Symbol:                 member.Symbol,
				UTCDate:                member.UTCDate,
				MembershipInterval:     membership,
				ParentManifestSHA256:   member.FileSHA256,
				ParentPartitionSHA256:  member.PartitionSHA256,
				ParentExpectedRows:     member.ExpectedRows,
				ChildRelativePath:      relative,
				ChildSHA256:            child.ChildSHA256,
				StoredFileSHA256:       storedHash,
				ChildRowCount:          len(child.Records),
				ChildFirstTimestampUTC: first,
				ChildLastTimestampUTC:  last,
				BoundaryClass:          class,
				Parent:                 child.Parent,
			}
			entries = append(entries, entry)
			incrementClass(&classCounts, class)
			incrementClass(&summary.Classes, class)
			summary.Artifacts++
		}
		sort.Slice(preparedMember.FragmentArtifacts, func(i, j int) bool {
			left, right := preparedMember.FragmentArtifacts[i], preparedMember.FragmentArtifacts[j]
			if !left.AuthorizedStartUTC.Equal(*right.AuthorizedStartUTC) {
				return left.AuthorizedStartUTC.Before(*right.AuthorizedStartUTC)
			}
			if !left.AuthorizedEndUTC.Equal(*right.AuthorizedEndUTC) {
				return left.AuthorizedEndUTC.Before(*right.AuthorizedEndUTC)
			}
			return left.CanonicalSHA256 < right.CanonicalSHA256
		})
		if err := validateMembershipRows(parent, member, membership, allRows); err != nil {
			return Plan{}, PreparationManifest{}, BoundaryAudit{}, fmt.Errorf("membership %s/%s: %w", member.UTCDate, member.Symbol, err)
		}
		preparedMembers = append(preparedMembers, preparedMember)
	}

	manifest := PreparationManifest{
		SchemaVersion:              PreparationManifestVersion,
		ParentPlanSHA256:           parent.PlanSHA256,
		Partition:                  parent.PartitionName,
		CheckpointSHA256:           parent.Checkpoint.SHA256,
		ParentSourceIdentitySHA256: parent.SourceIdentitySHA256,
		TransformationVersion:      BoundaryTransformationVersion,
		Entries:                    entries,
	}
	manifest, err = sealPreparationManifest(manifest)
	if err != nil {
		return Plan{}, PreparationManifest{}, BoundaryAudit{}, err
	}
	manifestBytes, err := encodePreparationManifest(manifest)
	if err != nil {
		return Plan{}, PreparationManifest{}, BoundaryAudit{}, err
	}
	manifestRelative := filepath.ToSlash(filepath.Join("manifests", hashLeaf(manifest.ManifestSHA256)+".json"))
	manifestPath, err := preparedWritePath(root, manifestRelative)
	if err != nil {
		return Plan{}, PreparationManifest{}, BoundaryAudit{}, err
	}
	if err := writeContentAddressed(manifestPath, manifestBytes, 0o400); err != nil {
		return Plan{}, PreparationManifest{}, BoundaryAudit{}, err
	}
	if err := syncPreparedRoot(root); err != nil {
		return Plan{}, PreparationManifest{}, BoundaryAudit{}, err
	}

	prepared := parent
	prepared.SchemaVersion = PreparedPlanSchemaVersion
	prepared.ParentPlanSHA256 = parent.PlanSHA256
	prepared.ParentSourceIdentitySHA256 = parent.SourceIdentitySHA256
	prepared.PreparedPartitionIdentity = manifest.PreparedSourceIdentitySHA256
	prepared.SourceIdentitySHA256 = manifest.PreparedSourceIdentitySHA256
	prepared.PreparationManifest = &HashIdentity{ID: manifestRelative, SHA256: manifest.ManifestSHA256}
	prepared.PreparedSourceRoot = root
	prepared.SourceRoot = ""
	prepared.ProspectiveSourceRoot = ""
	prepared.SourceManifests = preparedMembers
	prepared.PlanSHA256 = ""
	prepared.PlanSHA256, err = planHash(prepared)
	if err != nil {
		return Plan{}, PreparationManifest{}, BoundaryAudit{}, err
	}
	if err := VerifyPlan(prepared); err != nil {
		return Plan{}, PreparationManifest{}, BoundaryAudit{}, fmt.Errorf("verify prepared plan: %w", err)
	}

	symbolList := make([]BoundarySymbolSummary, 0, len(symbols))
	for _, symbol := range parent.DatasetRequiredSymbols {
		if summary := symbols[symbol]; summary != nil {
			symbolList = append(symbolList, *summary)
		}
	}
	audit := BoundaryAudit{
		SchemaVersion:             BoundaryAuditSchemaVersion,
		Partition:                 parent.PartitionName,
		ParentPlanSHA256:          parent.PlanSHA256,
		PreparedPlanSHA256:        prepared.PlanSHA256,
		PreparationManifestSHA256: manifest.ManifestSHA256,
		Memberships:               len(preparedMembers),
		Artifacts:                 len(entries),
		Classes:                   classCounts,
		Symbols:                   symbolList,
	}
	audit.AuditSHA256, err = boundaryAuditHash(audit)
	if err != nil {
		return Plan{}, PreparationManifest{}, BoundaryAudit{}, err
	}
	return prepared, manifest, audit, nil
}

func ensurePreparedRoot(root string) (string, error) {
	if root == "" || filepath.Clean(root) != root || !filepath.IsAbs(root) {
		return "", errUnsafePath
	}
	parent, err := canonicalRealDirectory(filepath.Dir(root))
	if err != nil {
		return "", err
	}
	root = filepath.Join(parent, filepath.Base(root))
	if err := ensureNonsymlinkDirectory(root); err != nil {
		return "", err
	}
	for _, name := range []string{"children", "manifests"} {
		path := filepath.Join(root, name)
		if err := ensureNonsymlinkDirectory(path); err != nil {
			return "", err
		}
	}
	return canonicalRealDirectory(root)
}

func ensureNonsymlinkDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if mkdirErr := os.Mkdir(path, 0o700); mkdirErr != nil && !errors.Is(mkdirErr, os.ErrExist) {
			return mkdirErr
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return errUnsafePath
	}
	return nil
}

func sourceMembershipInterval(plan Plan, member SourceManifest) (Interval, error) {
	if plan.SyntheticFixture {
		return plan.PartitionInterval, nil
	}
	date, err := time.Parse("2006-01-02", member.UTCDate)
	if err != nil {
		return Interval{}, errors.New("membership UTC date is invalid")
	}
	interval := Interval{Start: date.UTC(), End: date.UTC().Add(24 * time.Hour)}
	if interval.Start.Before(plan.PartitionInterval.Start) || interval.End.After(plan.PartitionInterval.End) {
		return Interval{}, errors.New("membership is outside the authorized stage")
	}
	return interval, nil
}

func authenticatedParentIdentity(plan Plan, member SourceManifest, index int, fragment sourceFragment) (ParentProvenance, error) {
	if index >= len(member.ReceiptArtifacts) {
		return ParentProvenance{}, errors.New("parent receipt is missing")
	}
	receiptRef, fragmentRef := member.ReceiptArtifacts[index], member.FragmentArtifacts[index]
	if receiptRef.CanonicalSHA256 != fragmentRef.ReceiptSHA256 {
		return ParentProvenance{}, errors.New("receipt and fragment binding differ")
	}
	if plan.SyntheticFixture {
		if len(fragment.Records) == 0 {
			return ParentProvenance{}, errors.New("synthetic parent is empty")
		}
		start := time.UnixMilli(fragment.Records[0].OpenTimeMS).UTC()
		end := time.UnixMilli(fragment.Records[len(fragment.Records)-1].OpenTimeMS).UTC().Add(time.Minute)
		parent := ParentProvenance{SourceRootID: fragmentRef.SourceRootID, ReceiptSHA256: receiptRef.CanonicalSHA256, FragmentSHA256: fragmentRef.CanonicalSHA256, ParentObjectID: fragmentRef.SourceRootID + ":" + fragmentRef.CanonicalSHA256, ParentStartUTC: start, ParentEndUTC: end, ParentRowCount: len(fragment.Records), ObservedAvailableAtUTC: fragmentRef.ObservedAvailableAtUTC.UTC()}
		return sealParentProvenance(parent)
	}
	root, err := sourceRootByID(plan.SourceRoot, plan.ProspectiveSourceRoot, receiptRef.SourceRootID)
	if err != nil {
		return ParentProvenance{}, err
	}
	path, err := secureJoin(root, receiptRef.RelativePath, true)
	if err != nil {
		return ParentProvenance{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ParentProvenance{}, err
	}
	indexed, err := decodeIndexedReceipt(data, receiptRef.SourceRootID, receiptRef.RelativePath)
	if err != nil || indexed.canonicalSHA256 != receiptRef.CanonicalSHA256 || indexed.fragmentPath != fragmentRef.RelativePath || indexed.fragmentHash != fragmentRef.CanonicalSHA256 || indexed.symbol != member.Symbol {
		return ParentProvenance{}, errors.New("authenticated receipt metadata differs from plan membership")
	}
	parent := ParentProvenance{SourceRootID: indexed.sourceRootID, ReceiptSHA256: indexed.canonicalSHA256, FragmentSHA256: indexed.fragmentHash, ParentObjectID: indexed.sourceRootID + ":" + indexed.fragmentHash, ParentStartUTC: indexed.parentStartUTC, ParentEndUTC: indexed.parentEndUTC, ParentRowCount: indexed.parentRowCount, ObservedAvailableAtUTC: indexed.observedAvailableAtUTC}
	return sealParentProvenance(parent)
}

func validateParentFragment(fragment sourceFragment, parent ParentProvenance, symbol string, requireMinuteCoverage bool) error {
	if fragment.Symbol != symbol || fragment.FragmentHash != parent.FragmentSHA256 || len(fragment.Records) != parent.ParentRowCount || len(fragment.Records) == 0 {
		return errors.New("parent fragment identity, symbol, or row count mismatch")
	}
	for index, row := range fragment.Records {
		opened := time.UnixMilli(row.OpenTimeMS).UTC()
		if row.Symbol != symbol || row.CloseTimeMS != row.OpenTimeMS+59999 || !row.MarketEventTimeUTC.Equal(opened) || !row.ProviderCandleCloseTimeUTC.Equal(time.UnixMilli(row.CloseTimeMS).UTC()) {
			return errors.New("parent rows are wrong-symbol, out of order, duplicate, gapped, or malformed")
		}
		if index > 0 && fragment.Records[index-1].OpenTimeMS >= row.OpenTimeMS {
			return errors.New("parent rows are wrong-symbol, out of order, duplicate, gapped, or malformed")
		}
		if requireMinuteCoverage {
			expected := parent.ParentStartUTC.Add(time.Duration(index) * time.Minute)
			if !opened.Equal(expected) {
				return errors.New("parent rows are wrong-symbol, out of order, duplicate, gapped, or malformed")
			}
		}
	}
	lastEnd := time.UnixMilli(fragment.Records[len(fragment.Records)-1].OpenTimeMS).UTC().Add(time.Minute)
	if !lastEnd.Equal(parent.ParentEndUTC) {
		return errors.New("parent metadata bounds differ from source rows")
	}
	return nil
}

func sliceChild(fragment sourceFragment, parent ParentProvenance, symbol string, membership Interval, requireMinuteCoverage bool) (ChildArtifact, string, error) {
	start := parent.ParentStartUTC
	if start.Before(membership.Start) {
		start = membership.Start
	}
	end := parent.ParentEndUTC
	if end.After(membership.End) {
		end = membership.End
	}
	if !start.Before(end) {
		return ChildArtifact{}, "", errors.New("authorized child slice is empty")
	}
	rows := make([]normalizedRecord, 0, int(end.Sub(start)/time.Minute))
	for _, row := range fragment.Records {
		opened := time.UnixMilli(row.OpenTimeMS).UTC()
		if !opened.Before(start) && opened.Before(end) {
			if row.Symbol != symbol {
				return ChildArtifact{}, "", errors.New("wrong-symbol row in authorized slice")
			}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 || (requireMinuteCoverage && len(rows) != int(end.Sub(start)/time.Minute)) {
		return ChildArtifact{}, "", errors.New("authorized child slice is empty or incomplete")
	}
	class := boundaryClass(parent.ParentStartUTC.Before(membership.Start), parent.ParentEndUTC.After(membership.End))
	child := ChildArtifact{SchemaVersion: ChildArtifactSchemaVersion, Symbol: symbol, AuthorizedInterval: Interval{Start: start, End: end}, Parent: parent, TransformationVersion: BoundaryTransformationVersion, Records: rows}
	var err error
	child.ChildSHA256, err = childArtifactHash(child)
	return child, class, err
}

func validateMembershipRows(plan Plan, member SourceManifest, membership Interval, rows []normalizedRecord) error {
	if len(rows) == 0 || len(rows) != member.ExpectedRows {
		return fmt.Errorf("coverage row count mismatch: expected=%d actual=%d", member.ExpectedRows, len(rows))
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].OpenTimeMS < rows[j].OpenTimeMS })
	for index, row := range rows {
		opened := time.UnixMilli(row.OpenTimeMS).UTC()
		if row.Symbol != member.Symbol || opened.Before(membership.Start) || !opened.Before(membership.End) {
			return errors.New("membership contains an unauthorized row")
		}
		if index > 0 && rows[index-1].OpenTimeMS >= row.OpenTimeMS {
			return errors.New("membership contains a duplicate or out-of-order timestamp")
		}
		if !plan.SyntheticFixture {
			expected := membership.Start.Add(time.Duration(index) * time.Minute)
			if !opened.Equal(expected) {
				return errors.New("membership has a missing expected candle")
			}
		}
	}
	if !plan.SyntheticFixture && !time.UnixMilli(rows[len(rows)-1].OpenTimeMS).UTC().Add(time.Minute).Equal(membership.End) {
		return errors.New("membership coverage does not reach the exclusive end")
	}
	return nil
}

func sealParentProvenance(parent ParentProvenance) (ParentProvenance, error) {
	parent.ProvenanceSHA256 = ""
	hash, err := canonicalHash(parent)
	if err != nil {
		return ParentProvenance{}, err
	}
	parent.ProvenanceSHA256 = hash
	return parent, nil
}

func verifyParentProvenance(parent ParentProvenance) error {
	want := parent
	want.ProvenanceSHA256 = ""
	hash, err := canonicalHash(want)
	if err != nil || !validSHA(parent.ReceiptSHA256) || !validSHA(parent.FragmentSHA256) || parent.ParentObjectID != parent.SourceRootID+":"+parent.FragmentSHA256 || !parent.ParentStartUTC.Before(parent.ParentEndUTC) || parent.ParentRowCount <= 0 || parent.ObservedAvailableAtUTC.IsZero() || parent.ProvenanceSHA256 != hash {
		return errors.New("parent provenance is invalid")
	}
	return nil
}

func childArtifactHash(child ChildArtifact) (string, error) {
	child.ChildSHA256 = ""
	return canonicalHash(child)
}

func encodeChildArtifact(child ChildArtifact) ([]byte, error) {
	if err := verifyChildArtifact(child); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(child)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	writer, err := gzip.NewWriterLevel(&out, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	writer.Header.ModTime = time.Unix(0, 0).UTC()
	writer.Header.OS = 255
	if _, err := writer.Write(raw); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func verifyChildArtifact(child ChildArtifact) error {
	if child.SchemaVersion != ChildArtifactSchemaVersion || child.Symbol == "" || !intervalValid(child.AuthorizedInterval) || child.TransformationVersion != BoundaryTransformationVersion || len(child.Records) == 0 || !validSHA(child.ChildSHA256) {
		return errors.New("boundary child artifact is incomplete")
	}
	if err := verifyParentProvenance(child.Parent); err != nil {
		return err
	}
	for index, row := range child.Records {
		opened := time.UnixMilli(row.OpenTimeMS).UTC()
		if row.Symbol != child.Symbol || opened.Before(child.AuthorizedInterval.Start) || !opened.Before(child.AuthorizedInterval.End) || row.CloseTimeMS != row.OpenTimeMS+59999 || !row.MarketEventTimeUTC.Equal(opened) || !row.ProviderCandleCloseTimeUTC.Equal(time.UnixMilli(row.CloseTimeMS).UTC()) {
			return errors.New("boundary child contains an unauthorized or malformed row")
		}
		if index > 0 && child.Records[index-1].OpenTimeMS >= row.OpenTimeMS {
			return errors.New("boundary child timestamps are not strictly monotonic")
		}
	}
	want, err := childArtifactHash(child)
	if err != nil || want != child.ChildSHA256 {
		return errors.New("boundary child canonical hash mismatch")
	}
	return nil
}

func sealPreparationManifest(manifest PreparationManifest) (PreparationManifest, error) {
	manifest.SchemaVersion = PreparationManifestVersion
	manifest.TransformationVersion = BoundaryTransformationVersion
	manifest.PreparedSourceIdentitySHA256 = ""
	manifest.ManifestSHA256 = ""
	baseHash, err := canonicalHash(manifest)
	if err != nil {
		return PreparationManifest{}, err
	}
	sourceHash, err := canonicalHash(struct {
		ParentSourceIdentitySHA256 string `json:"parent_source_identity_sha256"`
		CheckpointSHA256           string `json:"checkpoint_sha256"`
		PreparationManifestSHA256  string `json:"preparation_manifest_sha256"`
		TransformationVersion      string `json:"transformation_schema_version"`
	}{manifest.ParentSourceIdentitySHA256, manifest.CheckpointSHA256, baseHash, BoundaryTransformationVersion})
	if err != nil {
		return PreparationManifest{}, err
	}
	manifest.PreparedSourceIdentitySHA256 = sourceHash
	manifest.ManifestSHA256 = baseHash
	return manifest, nil
}

func verifyPreparationManifest(manifest PreparationManifest) error {
	sealed, err := sealPreparationManifest(manifest)
	if err != nil || !validSHA(manifest.ParentPlanSHA256) || !validSHA(manifest.CheckpointSHA256) || !validSHA(manifest.ParentSourceIdentitySHA256) || len(manifest.Entries) == 0 || sealed.PreparedSourceIdentitySHA256 != manifest.PreparedSourceIdentitySHA256 || sealed.ManifestSHA256 != manifest.ManifestSHA256 {
		return errors.New("boundary preparation manifest identity is invalid")
	}
	return nil
}

func encodePreparationManifest(manifest PreparationManifest) ([]byte, error) {
	if err := verifyPreparationManifest(manifest); err != nil {
		return nil, err
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func boundaryAuditHash(audit BoundaryAudit) (string, error) {
	audit.AuditSHA256 = ""
	return canonicalHash(audit)
}

func boundaryClass(left, right bool) string {
	switch {
	case left && right:
		return "BOTH_BOUNDARIES"
	case left:
		return "LEFT_CLIPPED"
	case right:
		return "RIGHT_CLIPPED"
	default:
		return "EXACT_BOUNDARY"
	}
}

func incrementClass(counts *BoundaryClassCounts, class string) {
	switch class {
	case "EXACT_BOUNDARY":
		counts.Exact++
	case "LEFT_CLIPPED":
		counts.Left++
	case "RIGHT_CLIPPED":
		counts.Right++
	case "BOTH_BOUNDARIES":
		counts.Both++
	}
}

func preparedWritePath(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || strings.Contains(relative, "..") || strings.Contains(relative, `\`) {
		return "", errUnsafePath
	}
	path := filepath.Join(root, filepath.FromSlash(relative))
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errUnsafePath
	}
	if err := rejectSymlinkComponents(filepath.Dir(path)); err != nil {
		return "", err
	}
	return path, nil
}

// writeContentAddressed uses an exclusive hard-link publish. The temporary
// file's contents are durable before publication. PreparePlan batches the
// directory fsync after all links are present; an interruption before that
// point is safely resumed from the same content identities.
func writeContentAddressed(path string, data []byte, mode os.FileMode) error {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm() != mode.Perm() {
			return errUnsafePath
		}
		existing, readErr := os.ReadFile(path)
		if readErr != nil || !bytes.Equal(existing, data) {
			return errors.New("content-addressed artifact conflicts with existing bytes")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".boundary-tmp-")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Link(tempPath, path); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		existing, readErr := os.ReadFile(path)
		if readErr != nil || !bytes.Equal(existing, data) {
			return errors.New("concurrent content-addressed artifact conflicts")
		}
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != mode.Perm() {
		return errors.New("published content-addressed artifact mode is unsafe")
	}
	return nil
}

func syncPreparedRoot(root string) error {
	for _, path := range []string{filepath.Join(root, "children"), filepath.Join(root, "manifests"), root} {
		directory, err := os.Open(path)
		if err != nil {
			return err
		}
		syncErr := directory.Sync()
		closeErr := directory.Close()
		if syncErr != nil {
			return syncErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func verifyPreparedPlan(plan Plan) error {
	if plan.SchemaVersion != PreparedPlanSchemaVersion || !validSHA(plan.ParentPlanSHA256) || !validSHA(plan.ParentSourceIdentitySHA256) || !validSHA(plan.PreparedPartitionIdentity) || !validSHA(plan.SourceIdentitySHA256) || plan.PreparationManifest == nil || !validSHA(plan.PreparationManifest.SHA256) || plan.PreparationManifest.ID == "" || plan.SourceRoot != "" || plan.ProspectiveSourceRoot != "" || plan.PreparedSourceRoot == "" {
		return errors.New("prepared plan boundary identity is incomplete")
	}
	root, err := canonicalRealDirectory(plan.PreparedSourceRoot)
	if err != nil || root != plan.PreparedSourceRoot {
		return errors.New("prepared source root is alternate or unsafe")
	}
	wantInterval, ok := acceptedIntervals[plan.PartitionName]
	if !ok {
		return errors.New("prepared plan name substitution")
	}
	if plan.SyntheticFixture {
		wantSynthetic, exists := syntheticAcceptedIntervals[plan.PartitionName]
		if !exists || (!reflectDeepEqualInterval(plan.PartitionInterval, wantSynthetic) && !reflectDeepEqualInterval(plan.PartitionInterval, wantInterval)) {
			return errors.New("prepared synthetic plan boundary substitution")
		}
	} else if !reflectDeepEqualInterval(plan.PartitionInterval, wantInterval) {
		return errors.New("prepared plan boundary substitution")
	}
	universe, err := qualificationUniverse()
	if err != nil {
		return err
	}
	if !stringSlicesEqual(plan.DatasetRequiredSymbols, universe.dataset) || !stringSlicesEqual(plan.CandidateTargetSymbols, universe.targets) || !stringSlicesEqual(plan.ContextOnlySymbols, universe.context) || plan.UniverseContractSHA256 != universe.hash {
		return errors.New("prepared plan universe substitution")
	}
	wantMemberships := plan.ExpectedStructuralDays * len(plan.DatasetRequiredSymbols)
	if plan.SyntheticFixture {
		wantMemberships = len(plan.DatasetRequiredSymbols)
	}
	if len(plan.SourceManifests) != wantMemberships {
		return errors.New("prepared plan membership cardinality mismatch")
	}
	manifest, err := loadPreparationManifest(plan)
	if err != nil {
		return err
	}
	if manifest.ManifestSHA256 != plan.PreparationManifest.SHA256 || manifest.ParentPlanSHA256 != plan.ParentPlanSHA256 || manifest.ParentSourceIdentitySHA256 != plan.ParentSourceIdentitySHA256 || manifest.PreparedSourceIdentitySHA256 != plan.PreparedPartitionIdentity || manifest.CheckpointSHA256 != plan.Checkpoint.SHA256 || manifest.Partition != plan.PartitionName {
		return errors.New("prepared plan and preparation manifest identities differ")
	}
	entries := map[string]PreparationManifestEntry{}
	for _, entry := range manifest.Entries {
		key := preparationEntryKey(entry.Partition, entry.Symbol, entry.UTCDate, entry.ChildSHA256)
		if _, duplicate := entries[key]; duplicate {
			return errors.New("preparation manifest contains duplicate child provenance")
		}
		entries[key] = entry
	}
	seenEntries := 0
	for index, member := range plan.SourceManifests {
		if !contains(plan.DatasetRequiredSymbols, member.Symbol) || member.MembershipInterval == nil || !intervalValid(*member.MembershipInterval) || member.RelativePath != "" || !validSHA(member.FileSHA256) || !validSHA(member.PartitionSHA256) || member.ExpectedRows <= 0 || len(member.ReceiptArtifacts) != 0 || len(member.FragmentArtifacts) == 0 {
			return errors.New("prepared membership is incomplete or exposes parent paths")
		}
		if index > 0 && (plan.SourceManifests[index-1].UTCDate > member.UTCDate || (plan.SourceManifests[index-1].UTCDate == member.UTCDate && plan.SourceManifests[index-1].Symbol >= member.Symbol)) {
			return errors.New("prepared membership order is noncanonical")
		}
		expectedMembership, err := sourceMembershipInterval(plan, member)
		if err != nil || !reflectDeepEqualInterval(*member.MembershipInterval, expectedMembership) {
			return errors.New("prepared membership interval differs from the authorized logical membership")
		}
		rows := []normalizedRecord{}
		for _, ref := range member.FragmentArtifacts {
			child, err := readChildArtifact(plan, ref)
			if err != nil {
				return err
			}
			if child.Symbol != member.Symbol || child.AuthorizedInterval.Start.Before(member.MembershipInterval.Start) || child.AuthorizedInterval.End.After(member.MembershipInterval.End) || child.AuthorizedInterval.Start.Before(plan.PartitionInterval.Start) || child.AuthorizedInterval.End.After(plan.PartitionInterval.End) {
				return errors.New("prepared child crosses a membership or stage boundary")
			}
			key := preparationEntryKey(plan.PartitionName, member.Symbol, member.UTCDate, child.ChildSHA256)
			entry, ok := entries[key]
			if !ok || entry.ParentManifestSHA256 != member.FileSHA256 || entry.ParentPartitionSHA256 != member.PartitionSHA256 || entry.ParentExpectedRows != member.ExpectedRows || entry.ChildRelativePath != ref.RelativePath || entry.StoredFileSHA256 != ref.StoredFileSHA256 || entry.ChildRowCount != ref.ChildRowCount || entry.BoundaryClass != ref.BoundaryClass || entry.Parent.ProvenanceSHA256 != ref.ParentProvenanceSHA256 || !reflectDeepEqualInterval(entry.MembershipInterval, *member.MembershipInterval) {
				return errors.New("prepared child lacks exact manifest provenance")
			}
			delete(entries, key)
			seenEntries++
			rows = append(rows, child.Records...)
		}
		if err := validateMembershipRows(plan, member, *member.MembershipInterval, rows); err != nil {
			return err
		}
	}
	if seenEntries != len(manifest.Entries) || len(entries) != 0 {
		return errors.New("preparation manifest contains missing or extra children")
	}
	want, err := planHash(plan)
	if err != nil || want != plan.PlanSHA256 {
		return errors.New("prepared plan canonical hash mismatch")
	}
	return nil
}

// BindPreparedDatasetSource assigns a content-addressed source identity shared
// by every stage plan in one prepared dataset. The partition-store identity and
// preparation manifest remain unchanged and independently verifiable.
func BindPreparedDatasetSource(plan Plan, datasetSourceIdentitySHA256 string) (Plan, error) {
	if err := verifyPreparedPlan(plan); err != nil {
		return Plan{}, err
	}
	if !validSHA(datasetSourceIdentitySHA256) {
		return Plan{}, errors.New("prepared dataset source identity is invalid")
	}
	plan.SourceIdentitySHA256 = datasetSourceIdentitySHA256
	plan.PlanSHA256 = ""
	var err error
	plan.PlanSHA256, err = planHash(plan)
	if err != nil {
		return Plan{}, err
	}
	if err := verifyPreparedPlan(plan); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

type universeView struct {
	dataset []string
	targets []string
	context []string
	hash    string
}

func qualificationUniverse() (universeView, error) {
	universe, err := qualificationrunner.V00UniverseContract()
	if err != nil {
		return universeView{}, err
	}
	return universeView{dataset: universe.DatasetRequiredSymbols, targets: universe.CandidateTargetSymbols, context: universe.ContextOnlySymbols, hash: universe.ContractSHA256}, nil
}

func reflectDeepEqualInterval(left, right Interval) bool {
	return left.Start.Equal(right.Start) && left.End.Equal(right.End)
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func loadPreparationManifest(plan Plan) (PreparationManifest, error) {
	if plan.PreparationManifest == nil {
		return PreparationManifest{}, errors.New("prepared plan lacks a preparation manifest")
	}
	path, err := secureJoin(plan.PreparedSourceRoot, plan.PreparationManifest.ID, true)
	if err != nil {
		return PreparationManifest{}, err
	}
	wantRelative := filepath.ToSlash(filepath.Join("manifests", hashLeaf(plan.PreparationManifest.SHA256)+".json"))
	info, err := os.Lstat(path)
	if plan.PreparationManifest.ID != wantRelative || err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o400 {
		return PreparationManifest{}, errors.New("preparation manifest path or mode is noncanonical")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return PreparationManifest{}, err
	}
	var manifest PreparationManifest
	if err := strictJSON(data, &manifest); err != nil {
		return PreparationManifest{}, err
	}
	canonical, err := encodePreparationManifest(manifest)
	if err != nil || !bytes.Equal(data, canonical) || manifest.ManifestSHA256 != plan.PreparationManifest.SHA256 {
		return PreparationManifest{}, errors.New("preparation manifest bytes or hash changed")
	}
	return manifest, nil
}

func readChildArtifact(plan Plan, ref SourceArtifact) (ChildArtifact, error) {
	if ref.SourceRootID != PreparedSourceRootID || ref.Encoding != PreparedFragmentEncoding || ref.RelativePath == "" || !validSHA(ref.CanonicalSHA256) || !validSHA(ref.StoredFileSHA256) || !validSHA(ref.ParentReceiptSHA256) || !validSHA(ref.ParentFragmentSHA256) || !validSHA(ref.ParentProvenanceSHA256) || ref.ParentSourceRootID == "" || ref.ReceiptSHA256 != ref.ParentReceiptSHA256 || ref.TransformationVersion != BoundaryTransformationVersion || ref.AuthorizedStartUTC == nil || ref.AuthorizedEndUTC == nil || ref.ChildFirstTimestampUTC == nil || ref.ChildLastTimestampUTC == nil || ref.ChildRowCount <= 0 {
		return ChildArtifact{}, errors.New("prepared child reference is incomplete")
	}
	path, err := secureJoin(plan.PreparedSourceRoot, ref.RelativePath, true)
	if err != nil {
		return ChildArtifact{}, err
	}
	wantRelative := filepath.ToSlash(filepath.Join("children", hashLeaf(ref.CanonicalSHA256)+".json.gz"))
	info, err := os.Lstat(path)
	if ref.RelativePath != wantRelative || err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o400 {
		return ChildArtifact{}, errors.New("prepared child path or mode is noncanonical")
	}
	encoded, err := os.ReadFile(path)
	if err != nil || byteHash(encoded) != ref.StoredFileSHA256 {
		return ChildArtifact{}, errors.New("prepared child stored bytes changed")
	}
	reader, err := gzip.NewReader(bytes.NewReader(encoded))
	if err != nil {
		return ChildArtifact{}, err
	}
	raw, readErr := io.ReadAll(io.LimitReader(reader, 64*1024*1024+1))
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || len(raw) > 64*1024*1024 {
		return ChildArtifact{}, errors.New("prepared child is truncated or exceeds the size limit")
	}
	var child ChildArtifact
	if err := strictJSON(raw, &child); err != nil {
		return ChildArtifact{}, err
	}
	if err := verifyChildArtifact(child); err != nil {
		return ChildArtifact{}, err
	}
	first := time.UnixMilli(child.Records[0].OpenTimeMS).UTC()
	last := time.UnixMilli(child.Records[len(child.Records)-1].OpenTimeMS).UTC()
	if child.ChildSHA256 != ref.CanonicalSHA256 || child.Symbol == "" || child.Parent.SourceRootID != ref.ParentSourceRootID || child.Parent.ReceiptSHA256 != ref.ParentReceiptSHA256 || child.Parent.FragmentSHA256 != ref.ParentFragmentSHA256 || child.Parent.ProvenanceSHA256 != ref.ParentProvenanceSHA256 || !child.AuthorizedInterval.Start.Equal(ref.AuthorizedStartUTC.UTC()) || !child.AuthorizedInterval.End.Equal(ref.AuthorizedEndUTC.UTC()) || len(child.Records) != ref.ChildRowCount || !first.Equal(ref.ChildFirstTimestampUTC.UTC()) || !last.Equal(ref.ChildLastTimestampUTC.UTC()) || boundaryClass(child.Parent.ParentStartUTC.Before(child.AuthorizedInterval.Start), child.Parent.ParentEndUTC.After(child.AuthorizedInterval.End)) != ref.BoundaryClass {
		return ChildArtifact{}, errors.New("prepared child metadata differs from its plan reference")
	}
	return child, nil
}

func preparationEntryKey(partition, symbol, date, child string) string {
	return strings.Join([]string{partition, symbol, date, child}, "\x00")
}

func AuditPreparedPlan(plan Plan) (BoundaryAudit, error) {
	if err := verifyPreparedPlan(plan); err != nil {
		return BoundaryAudit{SchemaVersion: BoundaryAuditSchemaVersion, Partition: plan.PartitionName, PreparedPlanSHA256: plan.PlanSHA256, Rejected: 1, UnsafeMemberships: 1}, err
	}
	manifest, err := loadPreparationManifest(plan)
	if err != nil {
		return BoundaryAudit{}, err
	}
	classes := BoundaryClassCounts{}
	symbols := map[string]*BoundarySymbolSummary{}
	for _, member := range plan.SourceManifests {
		summary := symbols[member.Symbol]
		if summary == nil {
			summary = &BoundarySymbolSummary{Symbol: member.Symbol}
			symbols[member.Symbol] = summary
		}
		summary.Memberships++
		for _, ref := range member.FragmentArtifacts {
			incrementClass(&classes, ref.BoundaryClass)
			incrementClass(&summary.Classes, ref.BoundaryClass)
			summary.Artifacts++
		}
	}
	symbolList := make([]BoundarySymbolSummary, 0, len(symbols))
	for _, symbol := range plan.DatasetRequiredSymbols {
		if summary := symbols[symbol]; summary != nil {
			symbolList = append(symbolList, *summary)
		}
	}
	audit := BoundaryAudit{SchemaVersion: BoundaryAuditSchemaVersion, Partition: plan.PartitionName, ParentPlanSHA256: plan.ParentPlanSHA256, PreparedPlanSHA256: plan.PlanSHA256, PreparationManifestSHA256: manifest.ManifestSHA256, Memberships: len(plan.SourceManifests), Artifacts: len(manifest.Entries), Classes: classes, Symbols: symbolList}
	audit.AuditSHA256, err = boundaryAuditHash(audit)
	return audit, err
}
