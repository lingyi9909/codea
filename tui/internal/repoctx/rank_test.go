package repoctx

import (
	"reflect"
	"testing"
)

func TestRankFullSignalPriority(t *testing.T) {
	idx := RepositoryIndex{
		Files: []SourceFile{
			{Path: "01-exact.go"},
			{Path: "02-changed.go"},
			{Path: "03-type.go"},
			{Path: "04-graph.go"},
			{Path: "05-lexical.go", Terms: []string{"needle"}},
		},
		Symbols: []Symbol{
			{ID: "seed", Name: "Needle", Path: "01-exact.go"},
			{ID: "changed", Name: "Other", Path: "02-changed.go"},
			{ID: "type", Name: "TypeHolder", Path: "03-type.go", TypeRefs: []string{"Needle"}},
			{ID: "neighbor", Name: "Neighbor", Path: "04-graph.go"},
		},
		Relations: []Relation{{From: "seed", To: "neighbor", Kind: RelationCalls}},
	}

	got := Rank(idx, Query{Text: "Needle", ChangedFiles: []string{"02-changed.go"}})
	reasons := make([]string, 0, len(got))
	for _, item := range got {
		reasons = append(reasons, item.Reason)
	}
	want := []string{"exact-symbol", "changed-file", "type-import", "graph-neighbor", "lexical"}
	if !reflect.DeepEqual(reasons, want) {
		t.Fatalf("reasons=%v want=%v", reasons, want)
	}

	again := Rank(idx, Query{Text: "Needle", ChangedFiles: []string{"02-changed.go"}})
	if !reflect.DeepEqual(got, again) {
		t.Fatalf("ranking is not deterministic: %+v vs %+v", got, again)
	}
}
