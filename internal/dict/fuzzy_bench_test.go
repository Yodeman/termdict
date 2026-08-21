package dict_test

import (
	"testing"

	"github.com/yodeman/termdict/internal/data/embedded"
	"github.com/yodeman/termdict/internal/dict"
)

// BenchmarkFuzzyMissWorstCase measures a did-you-mean lookup against
// the full embedded core (~22k words). The plan's target: well under
// 100 ms so the miss path stays interactive.
func BenchmarkFuzzyMissWorstCase(b *testing.B) {
	svc, err := dict.NewMulti(embedded.New())
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = svc.Fuzzy("zzqqxjwv", 5) // long, matches nothing: full scan
	}
}

func BenchmarkFuzzyTypicalTypo(b *testing.B) {
	svc, err := dict.NewMulti(embedded.New())
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = svc.Fuzzy("recieve", 5) //nolint:misspell // deliberate misspelling
	}
}
