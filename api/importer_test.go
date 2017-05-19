package wrapgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestImportInternalCacheHit(t *testing.T) {
	var path = "test"
	var pkg = &ast.Package{}
	var dirParser = func(*token.FileSet, string, func(os.FileInfo) bool, parser.Mode) (map[string]*ast.Package, error) {
		t.Fatal("tried to parse directory on a cache hit")
		return nil, fmt.Errorf("")
	}
	var importer = &defaultImporter{make(map[string]*ast.Package), dirParser}
	importer.cache[path] = pkg
	var result, e = importer._import(path)
	if e != nil {
		t.Fatalf("unexpected error finding import: %s", e.Error())
	}
	if result != pkg {
		t.Fatal("did not find expected cache result")
	}
}

func TestImportInternalDirParserFailure(t *testing.T) {
	var path = "test"
	var dirParser = func(*token.FileSet, string, func(os.FileInfo) bool, parser.Mode) (map[string]*ast.Package, error) {
		return nil, fmt.Errorf("")
	}
	var importer = &defaultImporter{make(map[string]*ast.Package), dirParser}
	var _, e = importer._import(path)
	if e == nil {
		t.Fatal("importer did not propagate error")
	}
}

func TestImportInternalMissingPackages(t *testing.T) {
	var path = "test"
	var dirParser = func(*token.FileSet, string, func(os.FileInfo) bool, parser.Mode) (map[string]*ast.Package, error) {
		return nil, nil
	}
	var importer = &defaultImporter{make(map[string]*ast.Package), dirParser}
	var _, e = importer._import(path)
	if e == nil {
		t.Fatal("importer did not error on missing package definitions")
	}
}

func TestImportInternalOnlyTestPackages(t *testing.T) {
	var path = "test"
	var dirParser = func(*token.FileSet, string, func(os.FileInfo) bool, parser.Mode) (map[string]*ast.Package, error) {
		return map[string]*ast.Package{"foo_test": &ast.Package{Name: "foo_test"}, "bar_test": &ast.Package{Name: "bar_test"}}, nil
	}
	var importer = &defaultImporter{make(map[string]*ast.Package), dirParser}
	var _, e = importer._import(path)
	if e == nil {
		t.Fatal("importer did not error when only test packages are found")
	}
}

func TestImportInternalCachesFoundPackages(t *testing.T) {
	var path = "test"
	var pkg = &ast.Package{Name: "pkg"}
	var dirParser = func(*token.FileSet, string, func(os.FileInfo) bool, parser.Mode) (map[string]*ast.Package, error) {
		return map[string]*ast.Package{"foo_test": &ast.Package{Name: "foo_test"}, "bar": pkg}, nil
	}
	var importer = &defaultImporter{make(map[string]*ast.Package), dirParser}
	var result, e = importer._import(path)
	if e != nil {
		t.Fatalf("unexpected error importing package")
	}
	if result != pkg {
		t.Fatalf("expected %v but got %v", pkg, result)
	}
	if importer.cache[path] != pkg {
		t.Fatalf("importer did not cache package")
	}
}

func TestImportSearchPathVendor(t *testing.T) {
	var from = "test"
	var pkg = "test2"
	var expected = filepath.Join(gopath(), "src", from, "vendor", pkg)
	var expectedPkg = &ast.Package{}
	var srcPath = filepath.Join(gopath(), "src", pkg)
	var rootPath = filepath.Join(runtime.GOROOT(), "src", pkg)
	var dirParser = func(fs *token.FileSet, path string, filter func(os.FileInfo) bool, mode parser.Mode) (map[string]*ast.Package, error) {
		if path == srcPath || path == rootPath {
			return nil, fmt.Errorf("")
		}
		return map[string]*ast.Package{pkg: expectedPkg}, nil
	}
	var importer = &defaultImporter{
		map[string]*ast.Package{expected: expectedPkg, srcPath: nil, rootPath: nil},
		dirParser,
	}
	var result, e = importer.Import(pkg, from)
	if e != nil {
		t.Fatalf("unexpected error importing package: %s", e.Error())
	}
	if result != expectedPkg {
		t.Fatalf("expected %v but got %v", expectedPkg, result)
	}
}

func TestImportSearchPathSrc(t *testing.T) {
	var from = "test"
	var pkg = "test2"
	var expectedPkg = &ast.Package{}
	var vendorPath = filepath.Join(gopath(), "src", from, "vendor", pkg)
	var srcPath = filepath.Join(gopath(), "src", pkg)
	var rootPath = filepath.Join(runtime.GOROOT(), "src", pkg)
	var dirParser = func(fs *token.FileSet, path string, filter func(os.FileInfo) bool, mode parser.Mode) (map[string]*ast.Package, error) {
		if path == vendorPath || path == rootPath {
			return nil, fmt.Errorf("")
		}
		return map[string]*ast.Package{pkg: expectedPkg}, nil
	}
	var importer = &defaultImporter{
		map[string]*ast.Package{srcPath: expectedPkg, vendorPath: nil, rootPath: nil},
		dirParser,
	}
	var result, e = importer.Import(pkg, "")
	if e != nil {
		t.Fatalf("unexpected error importing package: %s", e.Error())
	}
	if result != expectedPkg {
		t.Fatalf("expected %v but got %v", expectedPkg, result)
	}
}

func TestImportSearchPathRoot(t *testing.T) {
	var from = "test"
	var pkg = "test2"
	var expectedPkg = &ast.Package{}
	var vendorPath = filepath.Join(gopath(), "src", from, "vendor", pkg)
	var srcPath = filepath.Join(gopath(), "src", pkg)
	var rootPath = filepath.Join(runtime.GOROOT(), "src", pkg)
	var dirParser = func(fs *token.FileSet, path string, filter func(os.FileInfo) bool, mode parser.Mode) (map[string]*ast.Package, error) {
		if path == vendorPath || path == srcPath {
			return nil, fmt.Errorf("")
		}
		return map[string]*ast.Package{pkg: expectedPkg}, nil
	}
	var importer = &defaultImporter{
		map[string]*ast.Package{rootPath: expectedPkg},
		dirParser,
	}
	var result, e = importer.Import(pkg, "")
	if e != nil {
		t.Fatalf("unexpected error importing package: %s", e.Error())
	}
	if result != expectedPkg {
		t.Fatalf("expected %v but got %v", expectedPkg, result)
	}
}

func TestImportsNoFiles(t *testing.T) {
	var importer = &defaultImporter{
		make(map[string]*ast.Package),
		func(*token.FileSet, string, func(os.FileInfo) bool, parser.Mode) (map[string]*ast.Package, error) {
			t.Fatal("unexpected call to dir parser")
			return nil, nil
		},
	}
	var pkg = &ast.Package{}
	var imports = importer.Imports(pkg)
	if len(imports) != 0 {
		t.Fatalf("unexpected imports %v", imports)
	}
}

func TestImportsNoImports(t *testing.T) {
	var importer = &defaultImporter{
		make(map[string]*ast.Package),
		func(*token.FileSet, string, func(os.FileInfo) bool, parser.Mode) (map[string]*ast.Package, error) {
			t.Fatal("unexpected call to dir parser")
			return nil, nil
		},
	}
	var pkg = &ast.Package{
		Files: map[string]*ast.File{
			"foo": &ast.File{
				Decls: []ast.Decl{
					&ast.FuncDecl{},
					&ast.GenDecl{Tok: token.TYPE},
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.TypeSpec{},
						},
					},
				},
			},
		},
	}
	var imports = importer.Imports(pkg)
	if len(imports) != 0 {
		t.Fatalf("unexpected imports %v", imports)
	}
}

func TestImportsNameGiven(t *testing.T) {
	var importer = &defaultImporter{
		make(map[string]*ast.Package),
		func(*token.FileSet, string, func(os.FileInfo) bool, parser.Mode) (map[string]*ast.Package, error) {
			t.Fatal("unexpected call to dir parser")
			return nil, nil
		},
	}
	var pkg = &ast.Package{
		Files: map[string]*ast.File{
			"foo": &ast.File{
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{Value: `"foo.com/bar/baz"`},
								Name: &ast.Ident{
									Name: "barbaz",
								},
							},
						},
					},
				},
			},
		},
	}
	var imports = importer.Imports(pkg)
	if len(imports) != 1 {
		t.Fatalf("unexpected imports %v", imports)
	}
	if _, ok := imports["barbaz"]; !ok {
		t.Fatalf("could not find import %v", imports)
	}
	if imports["barbaz"] != "foo.com/bar/baz" {
		t.Fatalf("did not find expected import path. got %s", imports["barbaz"])
	}
}

func TestImportsNoNameGiven(t *testing.T) {
	var importer = &defaultImporter{
		make(map[string]*ast.Package),
		func(*token.FileSet, string, func(os.FileInfo) bool, parser.Mode) (map[string]*ast.Package, error) {
			t.Fatal("unexpected call to dir parser")
			return nil, nil
		},
	}
	var pkg = &ast.Package{
		Files: map[string]*ast.File{
			"foo": &ast.File{
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{Value: `"foo.com/bar/baz"`},
							},
						},
					},
				},
			},
		},
	}
	var imports = importer.Imports(pkg)
	if len(imports) != 1 {
		t.Fatalf("unexpected imports %v", imports)
	}
	if _, ok := imports["baz"]; !ok {
		t.Fatalf("could not find import %v", imports)
	}
	if imports["baz"] != "foo.com/bar/baz" {
		t.Fatalf("did not find expected import path. got %s", imports["barbaz"])
	}
}

func TestImportsSkipSideEffects(t *testing.T) {
	var importer = &defaultImporter{
		make(map[string]*ast.Package),
		func(*token.FileSet, string, func(os.FileInfo) bool, parser.Mode) (map[string]*ast.Package, error) {
			t.Fatal("unexpected call to dir parser")
			return nil, nil
		},
	}
	var pkg = &ast.Package{
		Files: map[string]*ast.File{
			"foo": &ast.File{
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{Value: `"foo.com/bar/baz"`},
								Name: &ast.Ident{
									Name: "_",
								},
							},
						},
					},
				},
			},
		},
	}
	var imports = importer.Imports(pkg)
	if len(imports) != 0 {
		t.Fatalf("unexpected imports %v", imports)
	}
}
