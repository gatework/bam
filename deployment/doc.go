// Package deployment submits and observes Address Manager RESTful v2 deployments.
//
// Configuration mutations do not deploy automatically. Use ForServer for full,
// differential or validation deployments; ForZone and ForNetwork for quick
// deployments; and Selective for selected resource records or IP addresses.
// Each submission returns a task, not proof that configuration is active.
// Tasks.Wait observes completion with a context and bounded polling interval.
// Stopping a wait does not cancel the task on the server. HTTP 202 remains a
// bam.AcceptedError containing the pending workflow request.
//
// Wire models and state values follow the vendor's 9.5 OpenAPI export preserved
// at github.com/umich-vci/gobam2, revision e894d31d00a85fb41f1ab648882600e5476ca091,
// with the official 9.6 RESTful v2 changes applied. In particular, 9.6 supports
// the force-zone-transfer header and filtering deployments by method, state,
// status and message. The export is not a 9.6 appliance acceptance test.
// Scheduled deployment administration is outside this module's current scope.
package deployment
