package wrapgen

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

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

func TestParseSelf(t *testing.T) {
	var parser = NewParser()
	var fullPath = filepath.Join(gopath(), "src", "github.com", "kevinconway", "wrapgen", "test")
	var pkg, e = parser.ParsePackage(fullPath)
	if e != nil {
		t.Fatal(e.Error())
	}
	if pkg.Name != "wrapgentest" {
		t.Fatalf("wrong package name %s", pkg.Name)
	}
	if len(pkg.Imports) != 1 {
		t.Fatalf("wrong number of imports %d", len(pkg.Imports))
	}
	if pkg.Imports[0].Package != "os" {
		t.Fatalf("wrong import %s", pkg.Imports[0].Package)
	}
	if len(pkg.Interfaces) != 2 {
		t.Fatalf("wrong number of ifaces %d", len(pkg.Interfaces))
	}
	var iface *Interface
	var embedded *Interface
	for o, i := range pkg.Interfaces {
		if i.Name == "ExportedInterface" {
			iface = pkg.Interfaces[o]
			continue
		}
		if i.Name == "ExportedInterfaceWithEmbedded" {
			embedded = pkg.Interfaces[o]
			continue
		}
		t.Fatalf("unexpected iface %s", i.Name)
	}
	if iface == nil || embedded == nil {
		t.Fatalf("missing interface. iface:%v embedded:%v", iface, embedded)
	}
}
