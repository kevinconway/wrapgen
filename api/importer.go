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
	"strings"
)

type importer interface {
	// Import loads a package using the same search path as the import statement. If a vendor directory
	// should be searched then a non-zero "from" string must be given as the import path of the
	// package containing a vendor directory.
	Import(pkg string, from string) (*ast.Package, error)
	// Imports generates a map of selector name -> import path of every package imported within
	// the given ast.Package instance.
	Imports(pkg *ast.Package) map[string]string
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

type dirParser func(*token.FileSet, string, func(os.FileInfo) bool, parser.Mode) (map[string]*ast.Package, error)

type defaultImporter struct {
	cache     map[string]*ast.Package
	dirParser dirParser
}

// _import performs a lookup of the given file path and returns the first valid, non-test
// package found.
func (i *defaultImporter) _import(pkg string) (*ast.Package, error) {
	if cached, ok := i.cache[pkg]; ok {
		return cached, nil
	}
	var fs = token.NewFileSet()
	var pkgs, e = i.dirParser(fs, pkg, nil, 0)
	if e != nil {
		return nil, e
	}
	if len(pkgs) < 1 {
		return nil, fmt.Errorf("no valid Google-golang packages found in %s", pkg)
	}
	var result *ast.Package
	for _, p := range pkgs {
		if strings.HasSuffix(p.Name, "_test") {
			continue
		}
		result = p
		break
	}
	if result == nil {
		return nil, fmt.Errorf("only test packages found in %s", pkg)
	}
	i.cache[pkg] = result
	return result, nil
}

// Import searches the same search path as the import statement to find
// a valid Google-golang package definition and returns it. Only the first
// valid, non-test package definition is returned.
func (i *defaultImporter) Import(pkg string, from string) (*ast.Package, error) {
	if len(from) > 0 {
		var vendorPath = filepath.Join(gopath(), "src", from, "vendor", pkg)
		var p, e = i._import(vendorPath)
		if e == nil {
			return p, nil
		}
	}
	var srcPath = filepath.Join(gopath(), "src", pkg)
	var p, e = i._import(srcPath)
	if e == nil {
		return p, nil
	}
	var rootPath = filepath.Join(runtime.GOROOT(), "src", pkg)
	return i._import(rootPath)
}

// Imports generates a map of name -> path of all the import statements
// contained within the given ast.Package instance.
func (i *defaultImporter) Imports(pkg *ast.Package) map[string]string {
	var results = make(map[string]string)
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
				results[pkg] = importPath
			}
		}
	}
	return results
}
