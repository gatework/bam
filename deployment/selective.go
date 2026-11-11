package deployment

import (
	"encoding/json"
	"fmt"

	"github.com/gatework/bam"
)

// BatchMode controls batching of selective deployment tasks.
type BatchMode string

// Selective deployment batching modes.
const (
	BatchDisabled BatchMode = "DISABLED"
	BatchByServer BatchMode = "BATCH_BY_SERVER"
)

// RecordScope controls inclusion of resources related to the selected records.
type RecordScope string

// Selective deployment record scopes.
const (
	RecordScopeRelated  RecordScope = "RELATED"
	RecordScopeSpecific RecordScope = "SPECIFIC"
)

// DynamicStrategy controls how a selective deployment handles dynamic records.
type DynamicStrategy string

// Strategies for handling dynamic records in a selective deployment.
const (
	DynamicAbort      DynamicStrategy = "ABORT"
	DynamicSkip       DynamicStrategy = "SKIP"
	DynamicMakeStatic DynamicStrategy = "MAKE_STATIC"
)

// SelectiveSpec selects DNS resource records or IP addresses by their globally
// unique IDs. The server validates resource kinds and ownership. Empty enum
// fields retain BAM defaults; ContinueOnFailure is sent explicitly, including
// false, so a zero value requests stopping on a failed record.
type SelectiveSpec struct {
	Resources         []bam.Reference `json:"resources"`
	BatchMode         BatchMode       `json:"batchMode,omitempty"`
	RecordScope       RecordScope     `json:"recordScope,omitempty"`
	ContinueOnFailure bool            `json:"continueOnFailureEnabled"`
	DynamicStrategy   DynamicStrategy `json:"dynamicRecordDeploymentStrategy,omitempty"`
}

// MarshalJSON validates required references and the documented enum values,
// then supplies the SelectiveDeployment resource discriminator.
func (s SelectiveSpec) MarshalJSON() ([]byte, error) {
	if len(s.Resources) == 0 {
		return nil, fmt.Errorf("deployment: selective deployment requires at least one resource")
	}
	for _, reference := range s.Resources {
		if reference.ID <= 0 {
			return nil, fmt.Errorf("deployment: selective resource id must be positive")
		}
	}
	if s.BatchMode != "" && s.BatchMode != BatchDisabled && s.BatchMode != BatchByServer {
		return nil, fmt.Errorf("deployment: unsupported selective batch mode")
	}
	if s.RecordScope != "" && s.RecordScope != RecordScopeRelated && s.RecordScope != RecordScopeSpecific {
		return nil, fmt.Errorf("deployment: unsupported selective record scope")
	}
	switch s.DynamicStrategy {
	case "", DynamicAbort, DynamicSkip, DynamicMakeStatic:
	default:
		return nil, fmt.Errorf("deployment: unsupported dynamic record strategy")
	}
	type spec SelectiveSpec
	return json.Marshal(struct {
		spec
		Type Kind `json:"type"`
	}{Type: KindSelective, spec: spec(s)})
}
