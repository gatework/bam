package bam_test

import (
	"encoding/json"
	"testing"

	"github.com/gatework/bam"
)

func TestValueRetainsPrecisionAndOwnsStorage(t *testing.T) {
	t.Parallel()
	input := []byte(`{"integer":9007199254740993,"enabled":false,"empty":null}`)
	var value bam.Value
	if err := value.UnmarshalJSON(input); err != nil {
		t.Fatal(err)
	}
	input[0] = '['
	copyOfValue := value
	encoded, err := value.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	encoded[0] = '['
	if err := value.UnmarshalJSON([]byte(`true`)); err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(copyOfValue)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"integer":9007199254740993,"enabled":false,"empty":null}` {
		t.Fatalf("option storage changed: %s", got)
	}
	var decoded struct {
		Integer int64 `json:"integer"`
	}
	if err := copyOfValue.Decode(&decoded); err != nil || decoded.Integer != 9007199254740993 {
		t.Fatalf("decode lost precision: %+v, %v", decoded, err)
	}
}
