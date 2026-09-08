package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/dby/type-analyzer/internal/config"
	csvexport "github.com/dby/type-analyzer/internal/export/csv"
	"github.com/dby/type-analyzer/internal/extract"
	"github.com/dby/type-analyzer/internal/load"
	"github.com/dby/type-analyzer/internal/model"
	neo4jstore "github.com/dby/type-analyzer/internal/store/neo4j"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "analyze":
		analyze(os.Args[2:])
	case "schema":
		schema(os.Args[2:])
	case "query":
		query(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func analyze(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "YAML config path")
	noImport := fs.Bool("no-import", false, "analyze only; do not write Neo4j")
	csvDir := fs.String("csv-dir", "", "write graph CSV files to this directory; skips Neo4j unless --also-import")
	alsoImport := fs.Bool("also-import", false, "with --csv-dir, also import into Neo4j")
	fs.Parse(args)
	c, err := config.Load(*configPath)
	die(err)
	if *csvDir != "" {
		c.Output.CSVDir = *csvDir
	}
	if *noImport && *alsoImport {
		die(fmt.Errorf("--no-import and --also-import cannot be used together"))
	}
	loaded, diags, err := load.Packages(c)
	die(err)
	defer loaded.Cleanup()
	v, _ := exec.Command("go", "version").Output()
	g := extract.Graph(c, modelRun(c, loaded, string(v)), loaded.Packages, diags)
	fmt.Printf("graph=%s mode=%s packages=%d types=%d methods=%d implements=%d embeds=%d references=%d diagnostics=%d\n", g.Run.ID, loaded.Mode, len(loaded.Packages), len(g.Types), len(g.Methods), len(g.Implementers), len(g.Embeddings), len(g.References), len(g.Diagnostics))
	for _, d := range g.Diagnostics {
		fmt.Fprintf(os.Stderr, "%s: %s: %s\n", d.Package, d.Position, d.Message)
	}
	if c.Output.CSVDir != "" {
		die(csvexport.Write(c.Output.CSVDir, g))
		fmt.Printf("csv=%s\n", c.Output.CSVDir)
	}
	if *noImport || (c.Output.CSVDir != "" && !*alsoImport) {
		return
	}
	s, err := neo4jstore.Open(context.Background(), c)
	die(err)
	defer s.Close(context.Background())
	die(s.Import(context.Background(), g))
}

func schema(args []string) {
	fs := flag.NewFlagSet("schema", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "YAML config path")
	file := fs.String("file", "schema.cypher", "schema file")
	fs.Parse(args)
	c, err := config.Load(*configPath)
	die(err)
	s, err := neo4jstore.Open(context.Background(), c)
	die(err)
	defer s.Close(context.Background())
	die(s.ExecuteSchema(context.Background(), *file))
}

func query(args []string) {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	file := fs.String("file", "queries.cypher", "query file")
	fs.Parse(args)
	b, err := os.ReadFile(*file)
	die(err)
	fmt.Print(string(b))
}

func modelRun(c config.Config, l load.Result, gov string) model.Run {
	return model.Run{ID: c.Neo4j.GraphID, ProjectRoot: c.Target.Root, ProjectImportPath: l.ImportPath, LoadMode: l.Mode, GoVersion: gov}
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "type-analyzer:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: type-analyzer analyze|schema|query [flags]")
}
