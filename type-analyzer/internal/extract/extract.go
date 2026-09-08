package extract

import (
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/dby/type-analyzer/internal/config"
	"github.com/dby/type-analyzer/internal/model"
	"golang.org/x/tools/go/packages"
)

type declaration struct {
	named *types.Named
	typ   model.Type
	pkg   *packages.Package
}

func Graph(c config.Config, run model.Run, pkgs []*packages.Package, diagnostics []model.Diagnostic) model.Graph {
	g := model.Graph{Run: run, Diagnostics: diagnostics}
	decls := map[string]declaration{}
	for _, p := range pkgs {
		if p.Types == nil {
			continue
		}
		if !c.Analysis.IncludeStdlib && isStdlibPackage(canonicalPath(p.PkgPath)) {
			continue
		}
		for _, name := range p.Types.Scope().Names() {
			obj, ok := p.Types.Scope().Lookup(name).(*types.TypeName)
			if !ok {
				continue
			}
			named, ok := unalias(obj.Type()).(*types.Named)
			if !ok {
				continue
			}
			id := typeID(named)
			if id == "" {
				continue
			}
			if _, exists := decls[id]; exists {
				continue
			}
			pos := position(p, obj.Pos())
			kind := "non_interface"
			if _, ok := named.Underlying().(*types.Interface); ok {
				kind = "interface"
			}
			project := under(c.Target.Root, pos.Filename)
			if !c.Analysis.IncludeDependencies && !project {
				continue
			}
			decls[id] = declaration{named: named, pkg: p, typ: model.Type{GraphID: run.ID, ID: id, Name: obj.Name(), QualifiedName: id, Kind: kind, PackageName: p.Name, PackagePath: canonicalPath(p.PkgPath), FilePath: pos.Filename, Line: pos.Line, Column: pos.Column, IsProjectPackage: project, IsStdlib: isStdlibPackage(canonicalPath(p.PkgPath)), TypeParams: typeParams(named)}}
		}
	}
	for _, d := range decls {
		g.Types = append(g.Types, d.typ)
	}
	sort.Slice(g.Types, func(i, j int) bool { return g.Types[i].ID < g.Types[j].ID })
	interfaces := []declaration{}
	concretes := []declaration{}
	for _, d := range decls {
		if d.typ.Kind == "interface" {
			iface := d.named.Underlying().(*types.Interface)
			iface.Complete()
			interfaces = append(interfaces, d)
		} else {
			concretes = append(concretes, d)
		}
		g.Methods = append(g.Methods, methods(run.ID, d)...)
		g.Embeddings = append(g.Embeddings, embeds(run.ID, d, decls)...)
		g.References = append(g.References, references(run.ID, d, decls)...)
	}
	for _, concrete := range concretes {
		for _, iface := range interfaces {
			v := types.Implements(concrete.named, iface.named.Underlying().(*types.Interface))
			ptr := types.Implements(types.NewPointer(concrete.named), iface.named.Underlying().(*types.Interface))
			if v || ptr {
				g.Implementers = append(g.Implementers, model.Implements{GraphID: run.ID, ConcreteID: concrete.typ.ID, InterfaceID: iface.typ.ID, ValueImpl: v, PointerImpl: ptr})
			}
		}
	}
	sort.Slice(g.Methods, func(i, j int) bool { return g.Methods[i].ID < g.Methods[j].ID })
	sort.Slice(g.Implementers, func(i, j int) bool {
		return g.Implementers[i].ConcreteID+g.Implementers[i].InterfaceID < g.Implementers[j].ConcreteID+g.Implementers[j].InterfaceID
	})
	return g
}

func methods(graphID string, d declaration) []model.Method {
	var funcs []*types.Func
	if iface, ok := d.named.Underlying().(*types.Interface); ok {
		iface.Complete()
		for i := 0; i < iface.NumExplicitMethods(); i++ {
			funcs = append(funcs, iface.ExplicitMethod(i))
		}
	} else {
		for i := 0; i < d.named.NumMethods(); i++ {
			funcs = append(funcs, d.named.Method(i))
		}
	}
	result := make([]model.Method, 0, len(funcs))
	for _, fn := range funcs {
		sig, _ := fn.Type().(*types.Signature)
		p := position(d.pkg, fn.Pos())
		pointer := sig != nil && sig.Recv() != nil && isPointer(sig.Recv().Type())
		result = append(result, model.Method{GraphID: graphID, ID: d.typ.ID + "#" + fn.Name(), Name: fn.Name(), Signature: types.TypeString(fn.Type(), qualifier), ReceiverTypeID: d.typ.ID, ReceiverPointer: pointer, FilePath: p.Filename, Line: p.Line, Column: p.Column})
	}
	return result
}

func embeds(graphID string, d declaration, decls map[string]declaration) []model.FieldRelation {
	st, ok := d.named.Underlying().(*types.Struct)
	if !ok {
		return nil
	}
	var out []model.FieldRelation
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if !f.Embedded() {
			continue
		}
		target, depth := directNamed(f.Type())
		id := typeID(target)
		if target == nil {
			continue
		}
		if _, ok := decls[id]; !ok {
			continue
		}
		p := position(d.pkg, f.Pos())
		out = append(out, model.FieldRelation{GraphID: graphID, ID: d.typ.ID + "#embed#" + itoa(i) + "#" + id, SourceID: d.typ.ID, TargetID: id, FieldName: f.Name(), FieldIndex: i, PointerDepth: depth, FilePath: p.Filename, Line: p.Line, Column: p.Column})
	}
	return out
}

func references(graphID string, d declaration, decls map[string]declaration) []model.FieldRelation {
	st, ok := d.named.Underlying().(*types.Struct)
	if !ok {
		return nil
	}
	var out []model.FieldRelation
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if f.Embedded() {
			continue
		}
		seen := map[string]bool{}
		for _, hit := range namedIn(f.Type(), "direct", 0) {
			id := typeID(hit.named)
			key := id + "|" + hit.path
			if id == "" || seen[key] {
				continue
			}
			seen[key] = true
			if _, ok := decls[id]; !ok {
				continue
			}
			p := position(d.pkg, f.Pos())
			out = append(out, model.FieldRelation{GraphID: graphID, ID: d.typ.ID + "#field#" + itoa(i) + "#" + hit.path + "#" + id, SourceID: d.typ.ID, TargetID: id, FieldName: f.Name(), FieldIndex: i, PointerDepth: hit.depth, ReferencePath: hit.path, FilePath: p.Filename, Line: p.Line, Column: p.Column})
		}
	}
	return out
}

type hit struct {
	named *types.Named
	path  string
	depth int
}

func namedIn(t types.Type, path string, depth int) []hit {
	t = unalias(t)
	switch x := t.(type) {
	case *types.Pointer:
		return namedIn(x.Elem(), path, depth+1)
	case *types.Named:
		out := []hit{{x, path, depth}}
		if a := x.TypeArgs(); a != nil {
			for i := 0; i < a.Len(); i++ {
				out = append(out, namedIn(a.At(i), path+".type_arg["+itoa(i)+"]", depth)...)
			}
		}
		return out
	case *types.Slice:
		return namedIn(x.Elem(), path+".slice", depth)
	case *types.Array:
		return namedIn(x.Elem(), path+".array", depth)
	case *types.Map:
		return append(namedIn(x.Key(), path+".map_key", depth), namedIn(x.Elem(), path+".map_value", depth)...)
	case *types.Chan:
		return namedIn(x.Elem(), path+".chan", depth)
	}
	return nil
}
func directNamed(t types.Type) (*types.Named, int) {
	d := 0
	for {
		t = unalias(t)
		p, ok := t.(*types.Pointer)
		if !ok {
			break
		}
		d++
		t = p.Elem()
	}
	n, _ := unalias(t).(*types.Named)
	return n, d
}
func unalias(t types.Type) types.Type { return types.Unalias(t) }
func typeID(n *types.Named) string {
	if n == nil || n.Obj() == nil || n.Obj().Pkg() == nil {
		return ""
	}
	return canonicalPath(n.Obj().Pkg().Path()) + "." + n.Obj().Name()
}
func typeParams(n *types.Named) []string {
	p := n.TypeParams()
	r := make([]string, p.Len())
	for i := 0; i < p.Len(); i++ {
		r[i] = p.At(i).String()
	}
	return r
}
func qualifier(p *types.Package) string {
	if p == nil {
		return ""
	}
	return canonicalPath(p.Path())
}
func canonicalPath(path string) string {
	if i := strings.LastIndex(path, "/vendor/"); i >= 0 {
		return path[i+len("/vendor/"):]
	}
	return path
}
func isStdlibPackage(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return first != "" && !strings.Contains(first, ".")
}
func isPointer(t types.Type) bool { _, ok := unalias(t).(*types.Pointer); return ok }
func position(p *packages.Package, pos token.Pos) token.Position {
	if p == nil || p.Fset == nil {
		return token.Position{}
	}
	return p.Fset.PositionFor(pos, true)
}
func under(root, file string) bool {
	if file == "" {
		return false
	}
	rel, err := filepath.Rel(root, file)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func itoa(i int) string { return strconv.Itoa(i) }
