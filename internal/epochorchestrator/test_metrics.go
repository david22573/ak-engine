package epochorchestrator

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/david22573/ak-engine/internal/rifbridge"
	"github.com/david22573/ak-engine/internal/qualificationrunner"
)

func ExtractMetrics(o *Orchestrator, variantID string) {
	stage := "DEVELOPMENT"
	snapshot, _ := o.authority.Snapshot()
	set := findStageSet(snapshot, stage)

	var auth rifbridge.StageVariantAuthorization
	for _, a := range set.Authorizations {
		if a.Configuration.VariantID == variantID {
			auth = a
			break
		}
	}

	envelope, _ := o.authority.ExportStageExecutionEnvelope(set.ExecutionSetID, auth.AuthorizationID)
	bridgeEnvelope := rifbridge.StageExecutionEnvelope{}
	data, _ := json.Marshal(envelope)
	json.Unmarshal(data, &bridgeEnvelope)

	artifactJSON, _ := os.ReadFile(fmt.Sprintf("%s/partition-registry/artifacts/%s/%s/%s.json", o.epochRoot, o.config.DatasetIdentity.Checkpoint.SHA256[7:], stage, o.config.Plans[stage].PlanSHA256[7:]))
	
	result, _, _ := qualificationrunner.ExecuteAuthorizedVariant(bridgeEnvelope, artifactJSON)
	
	metricsBytes, _ := json.MarshalIndent(result.Metrics, "", "  ")
	fmt.Printf("VARIANT %s METRICS:\n%s\n\n", variantID, string(metricsBytes))
	
	fmt.Printf("VARIANT %s DECISION:\n", variantID)
	decisionBytes, _ := json.MarshalIndent(result.GateDecision, "", "  ")
	fmt.Printf("%s\n\n", string(decisionBytes))
}
