package csvexport

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/dby/type-analyzer/internal/model"
)

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	g := model.Graph{Run: model.Run{ID: "g", LoadMode: "module"}, Types: []model.Type{{GraphID: "g", ID: "p.T", Name: "T", Kind: "non_interface", Line: 2}}, Methods: []model.Method{{GraphID: "g", ID: "p.T#M", Name: "M", ReceiverTypeID: "p.T"}}, Implementers: []model.Implements{{GraphID: "g", ConcreteID: "p.T", InterfaceID: "p.I", ValueImpl: true}}, Embeddings: []model.FieldRelation{{GraphID: "g", ID: "e", SourceID: "p.T", TargetID: "p.U", FieldName: "U"}}, References: []model.FieldRelation{{GraphID: "g", ID: "r", SourceID: "p.T", TargetID: "p.U", FieldName: "Ptr", PointerDepth: 1}}, Diagnostics: []model.Diagnostic{{Package: "p", Message: "bad"}}}
	if err := Write(dir, g); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"analysis_runs.csv", "types.csv", "methods.csv", "implements.csv", "embeds.csv", "references.csv", "diagnostics.csv"} {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		rows, err := csv.NewReader(f).ReadAll()
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 {
			t.Fatalf("%s: got %d rows, want header and record", name, len(rows))
		}
	}
}
