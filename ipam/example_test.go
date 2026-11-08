package ipam_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"

	"github.com/gatework/bam"
	"github.com/gatework/bam/dhcp"
	"github.com/gatework/bam/ipam"
)

func ExampleAddresses() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body bam.Document
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body["id"] = json.RawMessage("42")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()
	client, err := bam.New(server.URL)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	addresses := ipam.New(client).Addresses(7)

	fixed, err := addresses.AssignStatic(ctx, ipam.AddressSpec{
		Name: "application", Address: netip.MustParseAddr("192.0.2.10"),
	})
	if err != nil {
		panic(err)
	}
	held, err := addresses.Reserve(ctx, ipam.AddressSpec{
		Name: "future use", Address: netip.MustParseAddr("192.0.2.11"),
	})
	if err != nil {
		panic(err)
	}
	mac, err := ipam.ParseMAC("aa:bb:cc:00:11:22")
	if err != nil {
		panic(err)
	}
	binding, err := dhcp.New(client).Reservations(7).Create(ctx, dhcp.ReservationSpec{
		Name: "workstation", Address: netip.MustParseAddr("192.0.2.12"),
		MAC: ipam.MACReference{Address: mac},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(fixed.Address, fixed.State)
	fmt.Println(held.Address, held.State)
	fmt.Println(binding.Address, binding.State, binding.MACAddress.Address)
	// Output:
	// 192.0.2.10 STATIC
	// 192.0.2.11 RESERVED
	// 192.0.2.12 DHCP_RESERVED AA-BB-CC-00-11-22
}
