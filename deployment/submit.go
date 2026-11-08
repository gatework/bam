package deployment

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gatework/bam"
)

type submission struct {
	collection bam.Collection[Task]
	path       string
}

func (s submission) create(ctx context.Context, spec any, options ...bam.WriteOption) (Task, error) {
	task, err := s.collection.Create(ctx, spec, options...)
	if err != nil {
		return Task{}, err
	}
	if !knownKind(task.Type) {
		return Task{}, &bam.ProtocolError{
			Method: http.MethodPost, Path: s.path, Detail: "expected a deployment task",
		}
	}
	return task, nil
}

type serviceSpec struct {
	Type    Kind    `json:"type"`
	Service Service `json:"service"`
}

// Full submits the complete configuration of one service to the server.
func (t ServerTarget) Full(ctx context.Context, service Service, options ...bam.WriteOption) (Task, error) {
	return t.service(ctx, KindFull, service, options...)
}

// Differential requests deployment of changes since the previous deployment.
// The server validates whether a differential deployment is possible.
func (t ServerTarget) Differential(ctx context.Context, service Service, options ...bam.WriteOption) (Task, error) {
	return t.service(ctx, KindDifferential, service, options...)
}

// Validate starts pre-deployment validation for DNS, DHCPv4 or DHCPv6. It does
// not deploy the configuration, even when the validation task succeeds.
func (t ServerTarget) Validate(ctx context.Context, service Service, options ...bam.WriteOption) (Task, error) {
	return t.service(ctx, KindValidation, service, options...)
}

func (t ServerTarget) service(ctx context.Context, kind Kind, service Service, options ...bam.WriteOption) (Task, error) {
	switch service {
	case DNS, DHCPv4, DHCPv6:
	case TFTP:
		if kind == KindValidation {
			return Task{}, fmt.Errorf("deployment: tftp does not support pre-deployment validation")
		}
	default:
		return Task{}, fmt.Errorf("deployment: a supported service is required")
	}
	return t.submission.create(ctx, serviceSpec{Type: kind, Service: service}, options...)
}

// Quick deploys resource records updated by the current user in the selected
// zone or network. It does not deploy unrelated server configuration changes.
func (t QuickTarget) Quick(ctx context.Context, options ...bam.WriteOption) (Task, error) {
	return t.submission.create(ctx, struct {
		Type Kind `json:"type"`
	}{Type: KindQuick}, options...)
}

// WithForceZoneTransfer requests a re-transfer to a secondary DNS server.
// Apply it to a DNS deployment whose primary role is already configured.
func WithForceZoneTransfer(enabled bool) bam.WriteOption {
	return bam.WriteHeader("x-bcn-force-zone-transfer", strconv.FormatBool(enabled))
}
