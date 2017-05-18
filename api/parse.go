package wrapgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"strconv"
)

// PackageParser consumes an absolute path to a valid Google-golang package
// directory and generates a parsed Package object from it.
type PackageParser interface {
	ParsePackage(path string) (*Package, error)
}

type defaultParser struct{}

// NewParser generates a PackageParser using the default implementation.
func NewParser() PackageParser {
	return &defaultParser{}
}

func (p *defaultParser) ParsePackage(path string) (*Package, error) {
	return (&internalParser{
		importer:       &defaultImporter{make(map[string]*ast.Package), parser.ParseDir},
		counter:        counter(),
		importManifest: make(map[string]string),
		ifaceManifest:  make(map[string]map[string]*ast.InterfaceType),
		target:         path,
	}).ParsePackage(path)
}

func counter() func(string) int {
	var counts = make(map[string]int)
	return func(name string) int {
		var value, ok = counts[name]
		if !ok {
			value = 0
			counts[name] = 0
		}
		counts[name] = counts[name] + 1
		return value
	}
}

// internalParser is used to isolate the state management associated with parsing a package. This allows
// the defaultParser to be re-used or used concurrently without needed to reset or lock the internal state.
type internalParser struct {
	importer       importer
	counter        func(string) int
	importManifest map[string]string
	ifaceManifest  map[string]map[string]*ast.InterfaceType
	target         string
}

func (p *internalParser) populateInterfaces(pkg *ast.Package, pkgName string) {
	if _, ok := p.ifaceManifest[pkgName]; !ok {
		p.ifaceManifest[pkgName] = make(map[string]*ast.InterfaceType)
	}
	for _, f := range pkg.Files {
		mergeMapIface(p.ifaceManifest[pkgName], NewInterfaceMapper().Map(NewInterfaceIterator(f)))
	}
}

func (p *internalParser) populateImports(pkg *ast.Package) {
	mergeMapString(p.importManifest, p.importer.Imports(pkg))
}

func (p *internalParser) ParsePackage(path string) (*Package, error) {
	var pkg, e = p.importer.Import(path, "")
	if e != nil {
		return nil, e
	}
	p.populateImports(pkg)
	ast.PackageExports(pkg)
	var result = &Package{Name: pkg.Name, Interfaces: make([]*Interface, 0)}
	p.populateInterfaces(pkg, "")
	for _, f := range pkg.Files {
		var iterator = NewInterfaceIterator(f)
		for ifaceName, iface, e := iterator.Next(); e != ErrIteratorComplete; ifaceName, iface, e = iterator.Next() {
			var i, er = p.ParseInterface(pkg.Name, ifaceName, iface)
			if er != nil {
				return nil, er
			}
			result.Interfaces = append(result.Interfaces, i)
		}
	}
	for name, path := range p.importManifest {
		result.Imports = append(result.Imports, &Import{Package: name, Path: path})
	}
	return result, nil
}

func (p *internalParser) ParseInterface(pkg string, name string, i *ast.InterfaceType) (*Interface, error) {
	var iface = &Interface{Name: name, Methods: make([]*Method, 0)}
	for _, attribute := range i.Methods.List {
		switch n := attribute.Type.(type) {
		case *ast.FuncType:
			var m, e = p.ParseFunc(pkg, attribute.Names[0].String(), n)
			if e != nil {
				return nil, e
			}
			iface.Methods = append(iface.Methods, m)
		case *ast.Ident:
			var src, ok = p.ifaceManifest[""][n.String()]
			if !ok {
				return nil, fmt.Errorf("missing local embedded interface %s", n.String())
			}
			var i, e = p.ParseInterface(pkg, n.String(), src)
			if e != nil {
				return nil, e
			}
			iface.Methods = append(iface.Methods, i.Methods...)
		case *ast.SelectorExpr:
			var pkgName = n.X.(*ast.Ident).String()
			var _, ok = p.ifaceManifest[pkgName]
			if !ok {
				var imp, e = p.importer.Import(p.importManifest[pkgName], p.target)
				if e != nil {
					return nil, e
				}
				p.populateImports(imp)
				p.populateInterfaces(imp, pkgName)
			}
			var src *ast.InterfaceType
			src, ok = p.ifaceManifest[pkgName][n.Sel.String()]
			if !ok {
				return nil, fmt.Errorf("missing remote embedded interface %s.%s %v", pkgName, n.Sel.String(), p.ifaceManifest[pkgName])
			}
			var i, e = p.ParseInterface(pkgName, n.Sel.String(), src)
			if e != nil {
				return nil, e
			}
			iface.Methods = append(iface.Methods, i.Methods...)
		default:
			continue
		}
	}
	return iface, nil
}

func (p *internalParser) ParseFunc(pkg string, name string, f *ast.FuncType) (*Method, error) {
	var method = &Method{Name: name, In: make([]*Parameter, 0), Out: make([]*Parameter, 0)}
	if f.Params != nil {
		for _, arg := range f.Params.List {
			var param = &Parameter{Name: fmt.Sprintf("param%d", p.counter(pkg+name+"param"))}
			if len(arg.Names) > 0 {
				param.Name = arg.Names[0].String()
			}
			var t, e = p.ParseType(pkg, arg.Type)
			if e != nil {
				return nil, e
			}
			param.Type = t
			method.In = append(method.In, param)
		}
	}
	if f.Results != nil {
		for _, arg := range f.Results.List {
			var param = &Parameter{Name: fmt.Sprintf("result%d", p.counter(pkg+name+"result"))}
			if len(arg.Names) > 0 {
				param.Name = arg.Names[0].String()
			}
			var t, e = p.ParseType(pkg, arg.Type)
			if e != nil {
				return nil, e
			}
			param.Type = t
			method.Out = append(method.Out, param)
		}
	}
	return method, nil
}

func (p *internalParser) ParseType(pkg string, arg ast.Expr) (Type, error) {
	switch n := arg.(type) {
	case *ast.ArrayType:
		var len = -1
		if n.Len != nil {
			len, _ = strconv.Atoi(n.Len.(*ast.BasicLit).Value)
		}
		var typ, e = p.ParseType(pkg, n.Elt)
		if e != nil {
			return nil, e
		}
		return &TypeArray{Len: len, Type: typ}, nil
	case *ast.ChanType:
		var t, e = p.ParseType(pkg, n.Value)
		if e != nil {
			return nil, e
		}
		var chanType = &TypeChan{Type: t}
		if n.Dir == ast.SEND {
			chanType.WriteOnly = true
		}
		if n.Dir == ast.RECV {
			chanType.ReadOnly = true
		}
		return chanType, nil
	case *ast.Ellipsis:
		var t, e = p.ParseType(pkg, n.Elt)
		if e != nil {
			return nil, e
		}
		return &TypeVariadic{Type: t}, nil
	case *ast.FuncType:
		var method, e = p.ParseFunc(pkg, "", n)
		if e != nil {
			return nil, e
		}
		var result = &TypeFunc{In: make([]Type, 0), Out: make([]Type, 0)}
		for _, param := range method.In {
			result.In = append(result.In, param.Type)
		}
		for _, param := range method.Out {
			result.Out = append(result.Out, param.Type)
		}
		return result, nil
	case *ast.Ident:
		if n.IsExported() {
			// assume type in this package
			return &TypeExported{Package: pkg, Type: TypeBuiltin(n.Name)}, nil
		}
		return TypeBuiltin(n.Name), nil
	case *ast.InterfaceType:
		if n.Methods != nil && len(n.Methods.List) > 0 {
			return nil, fmt.Errorf("can't handle non-empty unnamed interface types at %v", n.Pos())
		}
		return TypeBuiltin("interface{}"), nil
	case *ast.MapType:
		var key Type
		var value Type
		var e error
		key, e = p.ParseType(pkg, n.Key)
		if e != nil {
			return nil, e
		}
		value, e = p.ParseType(pkg, n.Value)
		if e != nil {
			return nil, e
		}
		return &TypeMap{Key: key, Value: value}, nil
	case *ast.SelectorExpr:
		var pkgName = n.X.(*ast.Ident).String()
		return &TypeExported{Package: pkgName, Type: TypeBuiltin(n.Sel.String())}, nil
	case *ast.StarExpr:
		var t, e = p.ParseType(pkg, n.X)
		if e != nil {
			return nil, e
		}
		return &TypePointer{Type: t}, nil
	case *ast.StructType:
		if n.Fields != nil && len(n.Fields.List) > 0 {
			return nil, fmt.Errorf("can't handle non-empty unnamed struct types at %v", n.Pos())
		}
		return TypeBuiltin("struct{}"), nil
	}
	return nil, fmt.Errorf("unknown type: %T", arg)
}
