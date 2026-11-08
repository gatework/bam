package bam

import "errors"

// ErrNotFound means an exact lookup matched no resources. It is distinct from
// an HTTP 404 for a resource path, which remains an APIError.
var ErrNotFound = errors.New("bam: no matching resource")

// ErrAmbiguous means an exact lookup matched multiple resources. The caller
// must narrow its configuration, view, zone or network scope.
var ErrAmbiguous = errors.New("bam: multiple matching resources")
