package model

// Graph is the complete, database-independent result of one analysis.
type Graph struct {
	Run          Run
	Types        []Type
	Methods      []Method
	Implementers []Implements
	Embeddings   []FieldRelation
	References   []FieldRelation
	Diagnostics  []Diagnostic
}

type Run struct {
	ID                string
	ProjectRoot       string
	ProjectImportPath string
	LoadMode          string
	GoVersion         string
}

type Type struct {
	GraphID, ID, Name, QualifiedName, Kind string
	PackageName, PackagePath, FilePath     string
	Line, Column                           int
	IsProjectPackage, IsStdlib             bool
	TypeParams                             []string
}

type Method struct {
	GraphID, ID, Name, Signature, ReceiverTypeID, FilePath string
	ReceiverPointer                                        bool
	Line, Column                                           int
}

type Implements struct {
	GraphID, ConcreteID, InterfaceID string
	ValueImpl, PointerImpl           bool
}

// FieldRelation is used for both EMBEDS and REFERENCES.
type FieldRelation struct {
	GraphID, ID, SourceID, TargetID, FieldName, ReferencePath, FilePath string
	FieldIndex, PointerDepth, Line, Column                              int
}

type Diagnostic struct{ Package, Position, Message string }
