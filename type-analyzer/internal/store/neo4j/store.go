package neo4jstore

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dby/type-analyzer/internal/config"
	"github.com/dby/type-analyzer/internal/model"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Store struct {
	driver   neo4j.DriverWithContext
	database string
	batch    int
}

func Open(ctx context.Context, c config.Config) (*Store, error) {
	if c.Neo4j.URI == "" {
		return nil, fmt.Errorf("neo4j.uri is required for import")
	}
	password := os.Getenv(c.Neo4j.PasswordEnv)
	if c.Neo4j.PasswordEnv == "" || password == "" {
		return nil, fmt.Errorf("Neo4j password is missing: set neo4j.password_env and its environment variable")
	}
	d, err := neo4j.NewDriverWithContext(c.Neo4j.URI, neo4j.BasicAuth(c.Neo4j.User, password, ""))
	if err != nil {
		return nil, err
	}
	if err = d.VerifyConnectivity(ctx); err != nil {
		d.Close(ctx)
		return nil, err
	}
	return &Store{driver: d, database: c.Neo4j.Database, batch: c.Neo4j.BatchSize}, nil
}
func (s *Store) Close(ctx context.Context) error { return s.driver.Close(ctx) }

func (s *Store) Import(ctx context.Context, g model.Graph) error {
	rows := []map[string]any{{"id": g.Run.ID, "project_root": g.Run.ProjectRoot, "project_import_path": g.Run.ProjectImportPath, "load_mode": g.Run.LoadMode, "go_version": g.Run.GoVersion, "package_count": len(g.Types)}}
	if err := s.write(ctx, runQuery, rows); err != nil {
		return err
	}
	if err := s.clearRelations(ctx, g.Run.ID); err != nil {
		return err
	}
	var types, methods, impl, embeds, refs []map[string]any
	for _, x := range g.Types {
		types = append(types, map[string]any{"graph_id": x.GraphID, "id": x.ID, "name": x.Name, "qualified_name": x.QualifiedName, "kind": x.Kind, "package_name": x.PackageName, "package_path": x.PackagePath, "file_path": x.FilePath, "line": x.Line, "column": x.Column, "is_project_package": x.IsProjectPackage, "is_stdlib": x.IsStdlib, "type_params": x.TypeParams})
	}
	for _, x := range g.Methods {
		methods = append(methods, map[string]any{"graph_id": x.GraphID, "id": x.ID, "name": x.Name, "signature": x.Signature, "receiver_type_id": x.ReceiverTypeID, "receiver_pointer": x.ReceiverPointer, "file_path": x.FilePath, "line": x.Line, "column": x.Column})
	}
	for _, x := range g.Implementers {
		impl = append(impl, map[string]any{"graph_id": x.GraphID, "concrete_id": x.ConcreteID, "interface_id": x.InterfaceID, "value_impl": x.ValueImpl, "pointer_impl": x.PointerImpl})
	}
	for _, x := range g.Embeddings {
		embeds = append(embeds, fieldMap(x))
	}
	for _, x := range g.References {
		refs = append(refs, fieldMap(x))
	}
	for _, pair := range []struct {
		q string
		r []map[string]any
	}{{typeQuery, types}, {methodQuery, methods}, {typeRunQuery, types}, {methodRunQuery, methods}, {methodEdgeQuery, methods}, {implementsQuery, impl}, {embedQuery, embeds}, {referenceQuery, refs}} {
		if err := s.write(ctx, pair.q, pair.r); err != nil {
			return err
		}
	}
	if err := s.removeStale(ctx, g.Run.ID, typeIDs(g.Types), methodIDs(g.Methods)); err != nil {
		return err
	}
	return nil
}

func typeIDs(xs []model.Type) []string {
	ids := make([]string, len(xs))
	for i, x := range xs {
		ids[i] = x.ID
	}
	return ids
}
func methodIDs(xs []model.Method) []string {
	ids := make([]string, len(xs))
	for i, x := range xs {
		ids[i] = x.ID
	}
	return ids
}
func fieldMap(x model.FieldRelation) map[string]any {
	return map[string]any{"graph_id": x.GraphID, "id": x.ID, "source_id": x.SourceID, "target_id": x.TargetID, "field_name": x.FieldName, "field_index": x.FieldIndex, "pointer_depth": x.PointerDepth, "reference_path": x.ReferencePath, "file_path": x.FilePath, "line": x.Line, "column": x.Column}
}
func (s *Store) write(ctx context.Context, q string, rows []map[string]any) error {
	for start := 0; start < len(rows); start += s.batch {
		end := start + s.batch
		if end > len(rows) {
			end = len(rows)
		}
		session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, e := tx.Run(ctx, q, map[string]any{"rows": rows[start:end]})
			return nil, e
		})
		closeErr := session.Close(ctx)
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
func (s *Store) clearRelations(ctx context.Context, id string) error {
	return s.write(ctx, clearQuery, []map[string]any{{"graph_id": id}})
}
func (s *Store) removeStale(ctx context.Context, graphID string, typeIDs, methodIDs []string) error {
	if err := s.executeParams(ctx, staleTypesQuery, map[string]any{"graph_id": graphID, "ids": typeIDs}); err != nil {
		return err
	}
	return s.executeParams(ctx, staleMethodsQuery, map[string]any{"graph_id": graphID, "ids": methodIDs})
}

// ExecuteSchema runs the semicolon-terminated DDL statements distributed with this project.
func (s *Store) ExecuteSchema(ctx context.Context, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, q := range strings.Split(string(b), ";") {
		if strings.TrimSpace(q) == "" {
			continue
		}
		if err := s.execute(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) execute(ctx context.Context, query string) error {
	return s.executeParams(ctx, query, nil)
}
func (s *Store) executeParams(ctx context.Context, query string, params map[string]any) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})
	return err
}

const runQuery = `UNWIND $rows AS r MERGE (n:AnalysisRun {id:r.id}) SET n += r`
const typeQuery = `UNWIND $rows AS r MERGE (n:Type {graph_id:r.graph_id,id:r.id}) SET n += r`
const methodQuery = `UNWIND $rows AS r MERGE (n:Method {graph_id:r.graph_id,id:r.id}) SET n += r`
const typeRunQuery = `UNWIND $rows AS r MATCH (n:Type {graph_id:r.graph_id,id:r.id}),(a:AnalysisRun {id:r.graph_id}) MERGE (n)-[:IN_ANALYSIS {graph_id:r.graph_id}]->(a)`
const methodRunQuery = `UNWIND $rows AS r MATCH (n:Method {graph_id:r.graph_id,id:r.id}),(a:AnalysisRun {id:r.graph_id}) MERGE (n)-[:IN_ANALYSIS {graph_id:r.graph_id}]->(a)`
const methodEdgeQuery = `UNWIND $rows AS r MATCH (t:Type {graph_id:r.graph_id,id:r.receiver_type_id}),(m:Method {graph_id:r.graph_id,id:r.id}) MERGE (t)-[:DECLARES_METHOD {graph_id:r.graph_id}]->(m)`
const implementsQuery = `UNWIND $rows AS r MATCH (c:Type {graph_id:r.graph_id,id:r.concrete_id}),(i:Type {graph_id:r.graph_id,id:r.interface_id}) MERGE (c)-[e:IMPLEMENTS]->(i) SET e.graph_id=r.graph_id,e.value_impl=r.value_impl,e.pointer_impl=r.pointer_impl`
const embedQuery = `UNWIND $rows AS r MATCH (a:Type {graph_id:r.graph_id,id:r.source_id}),(b:Type {graph_id:r.graph_id,id:r.target_id}) MERGE (a)-[e:EMBEDS {id:r.id}]->(b) SET e += r`
const referenceQuery = `UNWIND $rows AS r MATCH (a:Type {graph_id:r.graph_id,id:r.source_id}),(b:Type {graph_id:r.graph_id,id:r.target_id}) MERGE (a)-[e:REFERENCES {id:r.id}]->(b) SET e += r`
const clearQuery = `UNWIND $rows AS r MATCH ()-[e]->() WHERE e.graph_id=r.graph_id DELETE e`
const staleTypesQuery = `MATCH (n:Type {graph_id:$graph_id}) WHERE NOT (n.id IN $ids) DETACH DELETE n`
const staleMethodsQuery = `MATCH (n:Method {graph_id:$graph_id}) WHERE NOT (n.id IN $ids) DETACH DELETE n`
