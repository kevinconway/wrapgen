package wrapgen

import (
	"path/filepath"
	"testing"
)

func TestParseSelf(t *testing.T) {
	var parser = NewParser()
	var path = filepath.Join("github.com", "kevinconway", "wrapgen", "test")
	var pkg, e = parser.ParsePackage(path)
	if e != nil {
		t.Fatal(e.Error())
	}
	if pkg.Name != "wrapgentest" {
		t.Fatalf("wrong package name %s", pkg.Name)
	}
	if len(pkg.Imports) < 3 {
		t.Fatalf("wrong number of imports %d", len(pkg.Imports))
	}
	if len(pkg.Interfaces) != 3 {
		t.Fatalf("wrong number of ifaces %d", len(pkg.Interfaces))
	}
	var iface *Interface
	var embedded *Interface
	var embeddedRemote *Interface
	for o, i := range pkg.Interfaces {
		if i.Name == "ExportedInterface" {
			iface = pkg.Interfaces[o]
			continue
		}
		if i.Name == "ExportedInterfaceWithEmbedded" {
			embedded = pkg.Interfaces[o]
			continue
		}
		if i.Name == "ExportedInterfaceWithRemoteEmbedded" {
			embeddedRemote = pkg.Interfaces[o]
			continue
		}
		t.Fatalf("unexpected iface %s", i.Name)
	}
	if iface == nil || embedded == nil || embeddedRemote == nil {
		t.Fatalf("missing interface. iface:%v embedded:%v embeddedRemote:%v", iface, embedded, embeddedRemote)
	}
}
