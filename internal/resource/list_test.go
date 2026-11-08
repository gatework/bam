package resource_test

import (
	"errors"
	"iter"
	"testing"

	"github.com/gatework/bam"
	"github.com/gatework/bam/internal/resource"
)

func TestOneDoesNotHideAmbiguityOrIteratorFailure(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("later page failed")
	for _, tc := range []struct {
		name    string
		values  []int
		failure error
		want    int
		wantErr error
		reads   int
	}{
		{"none", nil, nil, 0, bam.ErrNotFound, 0},
		{"one", []int{7}, nil, 7, nil, 1},
		{"ambiguous", []int{7, 8, 9}, nil, 0, bam.ErrAmbiguous, 2},
		{"later failure", []int{7}, wantErr, 0, wantErr, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			var seq iter.Seq2[int, error] = func(yield func(int, error) bool) {
				for _, v := range tc.values {
					reads++
					if !yield(v, nil) {
						return
					}
				}
				if tc.failure != nil {
					yield(0, tc.failure)
				}
			}
			got, err := resource.One(seq)
			if got != tc.want || !errors.Is(err, tc.wantErr) || reads != tc.reads {
				t.Fatalf("One=%d,%v reads=%d", got, err, reads)
			}
		})
	}
}
