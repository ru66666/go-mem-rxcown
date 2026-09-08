// Package csvexport writes the database-independent graph model as RFC 4180 CSV.
package csvexport

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dby/type-analyzer/internal/model"
)

// Write creates one file per node/relationship kind. Re-running with the same
// directory replaces only these well-known CSV files.
func Write(dir string, g model.Graph) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := writeFile(dir, "analysis_runs.csv", []string{"graph_id", "project_root", "project_import_path", "load_mode", "go_version"}, func(w *csv.Writer) {
		w.Write([]string{g.Run.ID, g.Run.ProjectRoot, g.Run.ProjectImportPath, g.Run.LoadMode, strings.TrimSpace(g.Run.GoVersion)})
	}); err != nil {
		return err
	}
	if err := writeFile(dir, "types.csv", []string{"graph_id", "id", "name", "qualified_name", "kind", "package_name", "package_path", "file_path", "line", "column", "is_project_package", "is_stdlib", "type_params"}, func(w *csv.Writer) {
		for _, x := range g.Types {
			w.Write([]string{x.GraphID, x.ID, x.Name, x.QualifiedName, x.Kind, x.PackageName, x.PackagePath, x.FilePath, num(x.Line), num(x.Column), truth(x.IsProjectPackage), truth(x.IsStdlib), strings.Join(x.TypeParams, "|")})
		}
	}); err != nil {
		return err
	}
	if err := writeFile(dir, "methods.csv", []string{"graph_id", "id", "name", "signature", "receiver_type_id", "receiver_pointer", "file_path", "line", "column"}, func(w *csv.Writer) {
		for _, x := range g.Methods {
			w.Write([]string{x.GraphID, x.ID, x.Name, x.Signature, x.ReceiverTypeID, truth(x.ReceiverPointer), x.FilePath, num(x.Line), num(x.Column)})
		}
	}); err != nil {
		return err
	}
	if err := writeFile(dir, "implements.csv", []string{"graph_id", "concrete_id", "interface_id", "value_impl", "pointer_impl"}, func(w *csv.Writer) {
		for _, x := range g.Implementers {
			w.Write([]string{x.GraphID, x.ConcreteID, x.InterfaceID, truth(x.ValueImpl), truth(x.PointerImpl)})
		}
	}); err != nil {
		return err
	}
	if err := writeFields(dir, "embeds.csv", g.Embeddings); err != nil {
		return err
	}
	if err := writeFields(dir, "references.csv", g.References); err != nil {
		return err
	}
	return writeFile(dir, "diagnostics.csv", []string{"package", "position", "message"}, func(w *csv.Writer) {
		for _, x := range g.Diagnostics {
			w.Write([]string{x.Package, x.Position, x.Message})
		}
	})
}

func writeFields(dir, name string, xs []model.FieldRelation) error {
	return writeFile(dir, name, []string{"graph_id", "id", "source_id", "target_id", "field_name", "field_index", "pointer_depth", "reference_path", "file_path", "line", "column"}, func(w *csv.Writer) {
		for _, x := range xs {
			w.Write([]string{x.GraphID, x.ID, x.SourceID, x.TargetID, x.FieldName, num(x.FieldIndex), num(x.PointerDepth), x.ReferencePath, x.FilePath, num(x.Line), num(x.Column)})
		}
	})
}
func writeFile(dir, name string, header []string, records func(*csv.Writer)) (err error) {
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	w := csv.NewWriter(f)
	if err = w.Write(header); err != nil {
		return err
	}
	records(w)
	w.Flush()
	if err = w.Error(); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}
func num(n int) string    { return strconv.Itoa(n) }
func truth(v bool) string { return strconv.FormatBool(v) }
