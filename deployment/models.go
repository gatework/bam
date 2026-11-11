package deployment

// Kind identifies a concrete RESTful v2 deployment resource.
type Kind string

// Supported deployment kinds.
const (
	KindFull         Kind = "FullDeployment"
	KindDifferential Kind = "DifferentialDeployment"
	KindQuick        Kind = "QuickDeployment"
	KindValidation   Kind = "ValidationDeployment"
	KindSelective    Kind = "SelectiveDeployment"
)

// Service selects one service per server deployment request.
type Service string

// Services selected for deployment.
const (
	DNS    Service = "DNS"
	DHCPv4 Service = "DHCPv4"
	DHCPv6 Service = "DHCPv6"
	TFTP   Service = "TFTP"
)

// State is the server's deployment state. Unknown values remain readable, but
// Wait cannot treat them as evidence of either progress or success.
type State string

// Deployment task lifecycle states returned by BAM.
const (
	StatePending               State = "PENDING"
	StateQueued                State = "QUEUED"
	StateRunning               State = "RUNNING"
	StateCancelled             State = "CANCELLED"
	StateCancelling            State = "CANCELLING"
	StateCompleted             State = "COMPLETED"
	StateCompletedWithWarnings State = "COMPLETED_WITH_WARNINGS"
	StateFailed                State = "FAILED"
	StateUnknown               State = "UNKNOWN"
)

// Task is the read representation shared by deployment types. Date-time fields
// retain the server's text because BAM releases use different timestamp formats.
// Use Tasks.Document when fields beyond this projection are needed.
type Task struct {
	ID                 int64   `json:"id"`
	Type               Kind    `json:"type"`
	Service            Service `json:"service"`
	State              State   `json:"state"`
	Status             string  `json:"status"`
	Message            string  `json:"message"`
	PercentComplete    int     `json:"percentComplete"`
	Method             string  `json:"method"`
	StartDateTime      string  `json:"startDateTime"`
	CompletionDateTime string  `json:"completionDateTime"`
}

func knownKind(kind Kind) bool {
	switch kind {
	case KindFull, KindDifferential, KindQuick, KindValidation, KindSelective:
		return true
	default:
		return false
	}
}
