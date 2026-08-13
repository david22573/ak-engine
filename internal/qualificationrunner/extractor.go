package qualificationrunner

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/david22573/ak-engine/internal/rifbridge"
)

func ExtractMetrics(variantID string) {
	rifData, _ := os.ReadFile("/home/davidmiguel22573/Github/canonical-run/ak-epoch-canonical-15-npkfyqnz/rif-research-governance.json")
	var snapshot rifbridge.ResearchGovernanceSnapshot
	json.Unmarshal(rifData, &snapshot)

	progData, _ := os.ReadFile("/home/davidmiguel22573/Github/canonical-run/ak-epoch-canonical-15-npkfyqnz/engine-development-progress.json")
	var prog struct {
		ExecutionReceipts []StageBatchExecutionReceipt `json:"execution_receipts"`
	}
	json.Unmarshal(progData, &prog)

	envelope := rifbridge.StageExecutionEnvelope{
		SchemaVersion: "ak.rif.stage_execution_envelope.v1",
		Snapshot:      snapshot,
		ExecutionSet:  snapshot.StageExecutionSets[0],
	}

	var auth rifbridge.StageVariantAuthorization
	for _, a := range snapshot.StageExecutionSets[0].Authorizations {
		if a.Configuration.VariantID == variantID {
			auth = a
			break
		}
	}

	protoData, _ := os.ReadFile("/home/davidmiguel22573/Github/canonical-run/ak-epoch-canonical-15-npkfyqnz/protocol.json")
	var proto struct {
		EngineVariantLedger struct {
			Variants []struct {
				ID            string                 `json:"id"`
				Configuration CandidateConfiguration `json:"configuration"`
			} `json:"variants"`
		} `json:"engine_variant_ledger"`
	}
	json.Unmarshal(protoData, &proto)

	var config CandidateConfiguration
	for _, v := range proto.EngineVariantLedger.Variants {
		if v.ID == variantID {
			config = v.Configuration
			break
		}
	}

	artifactJSON, _ := os.ReadFile("/home/davidmiguel22573/Github/canonical-run/ak-epoch-canonical-15-npkfyqnz/partition-registry/artifacts/bef53d11aa9ce9a6dad61b89ef7ace063b6da812ff92208d463c6ecfbfe8f29c/DEVELOPMENT/e7f9b6192f9a453e5f5c32d129abe116901b1536ff5760ef709d833f314da911.json")

	result, err := executeStageArtifact(envelope, auth, config, artifactJSON)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	
	// result.ResultArtifact is json.RawMessage
	var res map[string]interface{}
	json.Unmarshal(result.ResultArtifact, &res)
	
	metricsBytes, _ := json.MarshalIndent(res["metrics"], "", "  ")
	fmt.Printf("VARIANT %s METRICS:\n%s\n\n", variantID, string(metricsBytes))
	
	decisionBytes, _ := json.MarshalIndent(res["gate_decision"], "", "  ")
	fmt.Printf("VARIANT %s DECISION:\n%s\n\n", variantID, string(decisionBytes))
}
