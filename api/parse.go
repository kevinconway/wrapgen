package wrapgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// PackageParser consumes an absolute path to a valid Google-golang package
// directory and generates a parsed Package object from it.
type PackageParser interface {
	ParsePackage(path string) (*Package, error)
}

type defaultParser struct {
	dirParser      func(*token.FileSet, string, func(os.FileInfo) bool, parser.Mode) (map[string]*ast.Package, error)
	counter        func(string) int
	importManifest map[string]string
	ifaceManifest  map[string]map[string]*ast.InterfaceType
	targetPackage  string
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

// NewParser generates a PackageParser using the default implementation.
func NewParser() PackageParser {
	return &defaultParser{
		dirParser:      parser.ParseDir,
		counter:        counter(),
		importManifest: make(map[string]string),
		ifaceManifest:  make(map[string]map[string]*ast.InterfaceType),
		targetPackage:  "",
	}
}

func defaultGOPATH() string {
	var env = "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	}
	if runtime.GOOS == "plan9" {
		env = "home"
	}
	if home := os.Getenv(env); home != "" {
		var def = filepath.Join(home, "go")
		if filepath.Clean(def) == filepath.Clean(runtime.GOROOT()) {
			return ""
		}
		return def
	}
	return ""
}

func gopath() string {
	var p = os.Getenv("GOPATH")
	if p == "" {
		return defaultGOPATH()
	}
	return p
}

// Each directory path containing valid Google-golang code can only contain one package
// definition. This helper will attempt to extract that package from the directory.
func (p *defaultParser) openPackage(path string) (*ast.Package, error) {
	var fs = token.NewFileSet()
	var pkgs, e = p.dirParser(fs, path, nil, 0)
	if e != nil {
		return nil, e
	}
	if len(pkgs) < 1 {
		return nil, fmt.Errorf("found no valid Google-golang packages in %s", path)
	}
	for _, pkg := range pkgs {
		if strings.HasSuffix(pkg.Name, "_test") {
			continue
		}
		return pkg, nil
	}
	return nil, fmt.Errorf("impossible exit condition in package open")
}

// Each package requested is searched in the order of 1) vendor, 2) ${GOPATH}/src, and 3) stdlib.
// Packages not found in this search result in an error.
func (p *defaultParser) findImport(path string, base string) (*ast.Package, error) {
	if len(base) > 0 {
		var vendorDir = filepath.Join(base, "vendor", path)
		var pkg, e = p.openPackage(vendorDir)
		if e == nil {
			return pkg, nil
		}
	}
	var srcPath = filepath.Join(gopath(), "src", path)
	var pkg, e = p.openPackage(srcPath)
	if e == nil {
		return pkg, e
	}
	var rootPath = filepath.Join(runtime.GOROOT(), "src", path)
	return p.openPackage(rootPath)
}

// Iterate all files and populate import map with package imports
// Iterate all interfaces and populate iface map with root level ifaces
// Iterate all root level interfaces
// 	For local embedded types populate with contents from root level iface map
//  	For remote embedded types:
//		Iterate all files of remote package
//		Populate import mape with remote imports
//		Iterate all interfaces and populate iface map at selector name level
func (p *defaultParser) populateImports(pkg *ast.Package) {
	for _, f := range pkg.Files {
		for _, decl := range f.Decls {
			var gd, ok = decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.IMPORT {
				continue
			}
			for _, spec := range gd.Specs {
				var ok bool
				var is *ast.ImportSpec
				is, ok = spec.(*ast.ImportSpec)
				if !ok {
					continue
				}
				var importPath = string(is.Path.Value)
				importPath = importPath[1 : len(importPath)-1] // remove quotes

				// Default to the last path segment name as a best guess when an explicit
				// name is not given.
				var _, pkg = path.Split(importPath)
				pkg = strings.SplitN(pkg, ".", 2)[0]
				if is.Name != nil {
					if is.Name.Name == "_" {
						// This package was imported for a side-effect. Skip it.
						continue
					}
					pkg = is.Name.Name
					if pkg[len(pkg)-1] == '.' {
						pkg = pkg[0 : len(pkg)-1]
					}
				}
				p.importManifest[pkg] = importPath
			}
		}
	}
}

func (p *defaultParser) populateInterfaces(pkg *ast.Package, pkgName string) {
	if _, ok := p.ifaceManifest[pkgName]; !ok {
		p.ifaceManifest[pkgName] = make(map[string]*ast.InterfaceType)
	}
	// ast.PackageExports(pkg)
	for _, f := range pkg.Files {
		var iterator = NewInterfaceIterator(f)
		for ifaceName, iface, e := iterator.Next(); e != ErrIteratorComplete; ifaceName, iface, e = iterator.Next() {
			p.ifaceManifest[pkgName][ifaceName] = iface
		}
	}
}

func (p *defaultParser) ParsePackage(path string) (*Package, error) {
	p.targetPackage = path
	var pkg, e = p.openPackage(path)
	if e != nil {
		return nil, e
	}
	p.populateImports(pkg)
	var result = &Package{Name: pkg.Name, Interfaces: make([]*Interface, 0)}
	// Reduce the contents of the AST to only exported elements.
	ast.PackageExports(pkg)
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

func (p *defaultParser) ParseInterface(pkg string, name string, i *ast.InterfaceType) (*Interface, error) {
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
				var imp, e = p.findImport(p.importManifest[pkgName], p.targetPackage)
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

func (p *defaultParser) ParseFunc(pkg string, name string, f *ast.FuncType) (*Method, error) {
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

func (p *defaultParser) ParseType(pkg string, arg ast.Expr) (Type, error) {
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
