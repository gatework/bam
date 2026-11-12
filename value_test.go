package bam_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/gatework/bam"
)

func FuzzValue_UnmarshalJSON(f *testing.F) {
	for _, raw := range []string{"null", " \n null \t", "false", "0", "[]", `{"large":9007199254740993,"empty":null}`, `"null"`, "invalid"} {
		f.Add([]byte(raw))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		original := bytes.Clone(raw)
		var value bam.Value
		err := value.UnmarshalJSON(raw)
		if !json.Valid(original) {
			if err == nil || !value.IsZero() {
				t.Fatal("invalid JSON became an option value")
			}
			return
		}
		if err != nil || value.IsZero() {
			t.Fatalf("valid JSON was rejected: %v", err)
		}
		if value.IsNull() != bytes.Equal(bytes.TrimSpace(original), []byte("null")) {
			t.Fatal("null, unset, and explicit values were conflated")
		}
		raw[0] ^= 1
		encoded, err := value.MarshalJSON()
		if err != nil || !bytes.Equal(encoded, original) {
			t.Fatalf("input ownership or exact JSON bytes were lost: %v", err)
		}
		encoded[0] ^= 1
		again, err := value.MarshalJSON()
		if err != nil || !bytes.Equal(again, original) {
			t.Fatalf("output aliases the stored value: %v", err)
		}
	})
}

func TestValue_IsNull(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		input  any
		isNull bool
	}{
		{"nil", nil, true},
		{"nil pointer", (*string)(nil), true},
		{"nil slice", []string(nil), true},
		{"nil map", map[string]string(nil), true},
		{"false", false, false},
		{"zero", 0, false},
		{"empty string", "", false},
		{"empty array", []string{}, false},
		{"nested null", map[string]any{"child": nil}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			value, err := bam.ValueOf(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if value.IsZero() || value.IsNull() != tt.isNull {
				t.Fatalf("IsZero=%v IsNull=%v", value.IsZero(), value.IsNull())
			}
		})
	}
	var unset bam.Value
	if !unset.IsZero() || unset.IsNull() {
		t.Fatal("unset value must remain distinct from explicit null")
	}
}

func TestValue_UnmarshalJSON(t *testing.T) {
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
