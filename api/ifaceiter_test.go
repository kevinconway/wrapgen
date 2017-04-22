package wrapgen

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestIterEmpty(t *testing.T) {
	var source = &ast.File{}
	var iter = NewInterfaceIterator(source)
	var _, _, e = iter.Next()
	if e != ErrIteratorComplete {
		t.Fatalf("expected empty iterator but got %s", e)
	}
}

func TestIterNoTypeDeclarations(t *testing.T) {
	var source = &ast.File{
		Decls: []ast.Decl{
			&ast.FuncDecl{},
			&ast.FuncDecl{},
			&ast.GenDecl{Tok: token.IMPORT},
		},
	}
	var iter = NewInterfaceIterator(source)
	var _, _, e = iter.Next()
	if e != ErrIteratorComplete {
		t.Fatalf("expected empty iterator but got %s", e)
	}
}

func TestIterNoSpecsInDecl(t *testing.T) {
	var source = &ast.File{
		Decls: []ast.Decl{
			&ast.GenDecl{Tok: token.TYPE},
		},
	}
	var iter = NewInterfaceIterator(source)
	var _, _, e = iter.Next()
	if e != ErrIteratorComplete {
		t.Fatalf("expected empty iterator but got %s", e)
	}
}

func TestIterMismatchSpecTypeAndDeclTok(t *testing.T) {
	var source = &ast.File{
		Decls: []ast.Decl{
			&ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{&ast.ImportSpec{}}},
		},
	}
	var iter = NewInterfaceIterator(source)
	var _, _, e = iter.Next()
	if e != ErrIteratorComplete {
		t.Fatalf("expected empty iterator but got %s", e)
	}
}

func TestIterTypeSpecsButNotInterfaces(t *testing.T) {
	var source = &ast.File{
		Decls: []ast.Decl{
			&ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{&ast.TypeSpec{Type: &ast.StructType{}}}},
		},
	}
	var iter = NewInterfaceIterator(source)
	var _, _, e = iter.Next()
	if e != ErrIteratorComplete {
		t.Fatalf("expected empty iterator but got %s", e)
	}
}

func TestIterFoundInterface(t *testing.T) {
	var name = "testiface"
	var iface = &ast.InterfaceType{}
	var tspec = &ast.TypeSpec{Name: ast.NewIdent(name), Type: iface}
	var source = &ast.File{
		Decls: []ast.Decl{
			&ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{tspec}},
		},
	}
	var iter = NewInterfaceIterator(source)
	var rName, rIface, e = iter.Next()
	if e != nil {
		t.Fatalf("expected nil error but got %s", e.Error())
	}
	if rName != name {
		t.Fatalf("expected name %s but got %s", name, rName)
	}
	if rIface != iface {
		t.Fatalf("expected iface %v but got %v", iface, rIface)
	}
	_, _, e = iter.Next()
	if e != ErrIteratorComplete {
		t.Fatalf("expected empty iterator but got %s", e)
	}
}
