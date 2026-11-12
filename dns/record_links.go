package dns

import (
	"context"
	"fmt"
	"iter"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// RecordAddresses queries the address associations of one record.
type RecordAddresses struct{ resources resource.List[Address] }

// RecordDependencies queries records dependent on another record.
type RecordDependencies struct{ resources resource.Reader[Record] }

// AddressRecords manages record membership for one IP address. Unlink removes
// the association; RecordItems.Delete removes the DNS resource itself.
type AddressRecords struct {
	resources resource.List[Record]
	client    *bam.Client
	addressID int64
}

// Link associates an existing HostRecord or ExternalHostRecord with an address.
// Target must have a positive ID and explicit Host or External type.
func (r AddressRecords) Link(ctx context.Context, target RecordTarget, options ...bam.WriteOption) (Record, error) {
	if target.ID <= 0 || target.Type != Host && target.Type != External {
		return Record{}, fmt.Errorf("dns: address link requires a host or external host id and type")
	}
	record, err := resource.CollectionOf(r.resources).Create(ctx, RecordTarget{ID: target.ID, Type: target.Type}, options...)
	if err != nil {
		return Record{}, err
	}
	if record.ID != target.ID || record.Type != string(target.Type) {
		return Record{}, fmt.Errorf("dns: linked record response does not match the requested host")
	}
	return record, nil
}

// Get reads one linked record and verifies membership through the nested URI.
func (r AddressRecords) Get(ctx context.Context, id int64) (Record, error) {
	return bam.LinkedCollection[Record](r.client, "addresses", r.addressID, "resourceRecords").Get(ctx, id)
}

// Unlink removes only the address-to-record association.
func (r AddressRecords) Unlink(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return bam.LinkedCollection[Record](r.client, "addresses", r.addressID, "resourceRecords").Delete(ctx, id, options...)
}

// Page fetches one page within the selected collection scope.
func (r RecordAddresses) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Address], error) {
	return r.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (r RecordAddresses) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Address, error] {
	return r.resources.All(ctx, options)
}

// Page fetches one page within the selected collection scope.
func (r RecordDependencies) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Record], error) {
	return r.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (r RecordDependencies) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Record, error] {
	return r.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (r RecordDependencies) Get(ctx context.Context, id int64) (Record, error) {
	return r.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (r RecordDependencies) Document(ctx context.Context, id int64) (bam.Document, error) {
	return r.resources.Document(ctx, id)
}

// Page fetches one page within the selected collection scope.
func (r AddressRecords) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Record], error) {
	return r.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (r AddressRecords) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Record, error] {
	return r.resources.All(ctx, options)
}
