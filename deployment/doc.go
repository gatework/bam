// Package deployment selects Address Manager RESTful v2 deployment collections.
//
// ForServer, ForZone, and ForNetwork scope collection requests to a server, zone,
// or network. Tasks selects the global deployment collection. Individual tasks
// are retrieved by ID from /api/v2/deployments/{id}, including through a scoped
// selector.
//
// Use Create with a JSON request defined by the target Address Manager's
// /api/openapi.json schema. This package retains complete bam.Document values
// and does not guess deployment service enums, terminal states, or retry policy.
// An accepted submission does not establish that deployment has completed. Read
// the returned task using Get and interpret its state according to that schema.
// HTTP 202 is reported by the shared client as bam.AcceptedError.
//
// The shared Collection type exposes general resource methods. This does not
// imply that every deployment endpoint supports every method; the server's
// OpenAPI schema defines its capabilities. In particular, this package does not
// promise that tasks can be replaced or deleted.
//
// The BlueCat Integrity 9.6 migration guide maps server deployment, quick
// deployment, selective deployment, and task retrieval to these collections:
//
// https://docs.bluecatnetworks.com/r/Address-Manager-RESTful-v2-API-Guide/v1-REST-API-to-RESTful-v2-API-migration-guide/9.6.0
package deployment
