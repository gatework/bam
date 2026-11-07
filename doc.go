// Package bam implements BlueCat Address Manager RESTful v2 transport,
// authentication, and resource collections. Domain packages provide DNS, IPAM,
// DHCP, configuration, and deployment operations on the same client.
//
// The protocol baseline is BAM 9.6. All network operations accept a context.
// Writes are never retried automatically. A WorkflowRequest returned with HTTP
// 202 produces an AcceptedError: acceptance does not mean the change is applied.
package bam
