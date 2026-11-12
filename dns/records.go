package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"reflect"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

// RecordItems manages existing resource records without exposing creation at
// an unsupported global scope.
type RecordItems struct {
	resources resource.Reader[Record]
	client    *bam.Client
}

func newRecordItems(client *bam.Client, collection bam.Collection[Record]) RecordItems {
	return RecordItems{resources: resource.NewReader(collection), client: client}
}

// Records adds type-safe creation within a selected DNS zone.
type Records struct{ items RecordItems }

// Create creates a supported record subtype. HTTP 202 remains an AcceptedError;
// the caller must not treat a pending workflow as an existing record.
func (r Records) Create(ctx context.Context, spec RecordSpec, options ...bam.WriteOption) (Record, error) {
	created, err := resource.CollectionOf(r.items.resources.List).Create(ctx, spec, options...)
	if err != nil {
		return Record{}, err
	}
	if created.Type != string(spec.recordKind()) {
		return Record{}, fmt.Errorf("dns: create returned unexpected record type %q", created.Type)
	}
	return created, nil
}

// Delete removes a resource record by its global ID. Use AddressRecords.Unlink
// when only the association with an IP address should be removed.
func (r RecordItems) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return resource.CollectionOf(r.resources.List).Delete(ctx, id, options...)
}

// Update reads the complete representation and preserves host-address links
// before a full PUT. The read/edit/write cycle is not atomic with other writers.
func (r RecordItems) Update(ctx context.Context, id int64, changes RecordChanges, options ...bam.WriteOption) (Record, error) {
	if changes == nil || reflect.ValueOf(changes).Kind() == reflect.Pointer && reflect.ValueOf(changes).IsNil() {
		return Record{}, fmt.Errorf("dns: record changes cannot be nil")
	}
	return resource.CollectionOf(r.resources.List).Update(ctx, id, func(document bam.Document) error {
		var kind RecordKind
		if err := json.Unmarshal(document["type"], &kind); err != nil {
			return err
		}
		switch kind {
		case Host, Alias, MX, TXT, SRV, HINFO, NAPTR, Generic, External:
		default:
			return fmt.Errorf("dns: typed update schema is not available for %s", kind)
		}
		if err := changes.Apply(document); err != nil {
			return err
		}
		if kind != Host {
			return nil
		}
		if _, ok := document["reverseRecord"]; !ok {
			return fmt.Errorf("dns: complete host representation is missing reverse record setting")
		}
		if change, ok := changes.(interface{ changesAddresses() bool }); ok && change.changesAddresses() {
			return nil
		}
		return r.preserveAddresses(ctx, id, document)
	}, options...)
}

// Page fetches one page within the selected collection scope.
func (r RecordItems) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Record], error) {
	return r.resources.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (r RecordItems) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Record, error] {
	return r.resources.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (r RecordItems) Get(ctx context.Context, id int64) (Record, error) {
	return r.resources.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (r RecordItems) Document(ctx context.Context, id int64) (bam.Document, error) {
	return r.resources.Document(ctx, id)
}

// Page fetches one page within the selected collection scope.
func (r Records) Page(ctx context.Context, options bam.ListOptions) (bam.Page[Record], error) {
	return r.items.Page(ctx, options)
}

// All follows bounded pagination while the caller continues iteration.
func (r Records) All(ctx context.Context, options bam.ListOptions) iter.Seq2[Record, error] {
	return r.items.All(ctx, options)
}

// Get reads and validates an existing resource by ID.
func (r Records) Get(ctx context.Context, id int64) (Record, error) {
	return r.items.Get(ctx, id)
}

// Document reads the complete representation without field projection.
func (r Records) Document(ctx context.Context, id int64) (bam.Document, error) {
	return r.items.Document(ctx, id)
}

// Update applies typed changes to the complete document without retrying writes.
func (r Records) Update(ctx context.Context, id int64, changes RecordChanges, options ...bam.WriteOption) (Record, error) {
	return r.items.Update(ctx, id, changes, options...)
}

// Delete removes an existing resource without retrying the write.
func (r Records) Delete(ctx context.Context, id int64, options ...bam.WriteOption) error {
	return r.items.Delete(ctx, id, options...)
}

// Search retrieves matching records lazily.
func (r Records) Search(ctx context.Context, query RecordQuery) iter.Seq2[Record, error] {
	return r.items.Search(ctx, query)
}

// Find requires exactly one matching record.
func (r Records) Find(ctx context.Context, query RecordQuery) (Record, error) {
	return r.items.Find(ctx, query)
}

func (r RecordItems) preserveAddresses(ctx context.Context, recordID int64, document bam.Document) error {
	var addresses []linkedAddress
	for address, err := range New(r.client).Addresses(recordID).All(ctx, bam.ListOptions{}) {
		if err != nil {
			return err
		}
		if address.ID <= 0 || address.Type != "IPv4Address" && address.Type != "IPv6Address" {
			return fmt.Errorf("dns: invalid address association in host representation")
		}
		addresses = append(addresses, linkedAddress{ID: address.ID, Type: address.Type})
	}
	if len(addresses) == 0 {
		return fmt.Errorf("dns: host representation has no address associations")
	}
	return document.Set("addresses", addresses)
}
