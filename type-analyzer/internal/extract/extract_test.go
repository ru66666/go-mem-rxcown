package extract_test

import (
	"path/filepath"
	"testing"

	"github.com/dby/type-analyzer/internal/config"
	"github.com/dby/type-analyzer/internal/extract"
	"github.com/dby/type-analyzer/internal/load"
	"github.com/dby/type-analyzer/internal/model"
)

func TestGraphExtractsRequiredRelationships(t *testing.T) {
	root, err := filepath.Abs("testdata/module")
	if err != nil {
		t.Fatal(err)
	}
	c := config.Config{}
	c.Target.Root, c.Target.Mode, c.Target.Patterns = root, "module", []string{"./..."}
	c.Analysis.IncludeDependencies = false
	loaded, diags, err := load.Packages(c)
	if err != nil {
		t.Fatal(err)
	}
	defer loaded.Cleanup()
	g := extract.Graph(c, model.Run{ID: "test", ProjectRoot: root, LoadMode: loaded.Mode}, loaded.Packages, diags)
	if len(g.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", g.Diagnostics)
	}
	assertImplementation(t, g, "example.test/demo.Base", true, true)
	assertImplementation(t, g, "example.test/demo.PtrOnly", false, true)
	assertField(t, g.Embeddings, "Inner", 0, "")
	assertField(t, g.References, "Ptr", 1, "direct")
	assertField(t, g.References, "Value", 0, "direct")
	assertField(t, g.References, "Items", 0, "direct.slice")
	for _, m := range g.Methods {
		if m.ID == "example.test/demo.Outer#Run" && !m.ReceiverPointer {
			t.Fatal("pointer receiver was not recorded")
		}
	}
}

func assertImplementation(t *testing.T, g model.Graph, concrete string, value, pointer bool) {
	t.Helper()
	for _, r := range g.Implementers {
		if r.ConcreteID == concrete && r.InterfaceID == "example.test/demo.Reader" {
			if r.ValueImpl != value || r.PointerImpl != pointer {
				t.Fatalf("implementation flags for %s: %#v", concrete, r)
			}
			return
		}
	}
	t.Fatalf("implementation missing for %s", concrete)
}
func assertField(t *testing.T, xs []model.FieldRelation, name string, depth int, path string) {
	t.Helper()
	for _, r := range xs {
		if r.FieldName == name && r.PointerDepth == depth && r.ReferencePath == path {
			return
		}
	}
	t.Fatalf("field relationship missing: %s depth=%d path=%q; got %#v", name, depth, path, xs)
}
