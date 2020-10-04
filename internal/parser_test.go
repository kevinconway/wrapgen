package wrapgen

import (
	"context"
	"testing"
)

func TestParserSuccess(t *testing.T) {
	ctx := context.Background()
	path := "./test/happy"
	names := []string{
		"ExportedInterface",
		"ExportedInterfaceWithEmbedded",
		"ExportedInterfaceWithRemoteEmbedded",
		"ExportedInterfaceWith3rdPartyEmbedded",
		"InterfaceExtension",
		"InterfaceAlias",
		"RemoteInterfaceExtension",
		"RemoteInterfaceAlias",
		"ThirdPartyInterfaceExtension",
		"ThirdPartyInterfaceAlias",
		"IndirectThirdPartyInterfaceExtension",
		"IndirectThirdPartyInterfaceAlias",
	}
	pkg, err := LoadPackage(ctx, path, "", names)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(pkg.Interfaces) < len(names) {
		t.Fatalf("did not render all interfaces: %v", pkg.Interfaces)
	}
}

func TestParserSuccessCustomPackageName(t *testing.T) {
	ctx := context.Background()
	path := "github.com/kevinconway/wrapgen/v2/internal/test/happy"
	names := []string{
		"ExportedInterface",
		"ExportedInterfaceWithEmbedded",
		"ExportedInterfaceWithRemoteEmbedded",
		"ExportedInterfaceWith3rdPartyEmbedded",
		"InterfaceExtension",
		"InterfaceAlias",
		"RemoteInterfaceExtension",
		"RemoteInterfaceAlias",
		"ThirdPartyInterfaceExtension",
		"ThirdPartyInterfaceAlias",
		"IndirectThirdPartyInterfaceExtension",
		"IndirectThirdPartyInterfaceAlias",
	}
	pkg, err := LoadPackage(ctx, path, "custom", names)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(pkg.Interfaces) < len(names) {
		t.Fatalf("did not render all interfaces: %v", pkg.Interfaces)
	}
	if pkg.Name != "custom" {
		t.Fatalf("did not use correct output package name: %s", pkg.Name)
	}
	found := false
	for _, imp := range pkg.Imports {
		if imp.Path == path && imp.Package == sourceAlias {
			found = true
		}
	}
	if !found {
		t.Fatalf("did not inject a source package alias: %v", pkg.Imports)
	}
}

func TestParserFailure(t *testing.T) {
	testCases := []struct {
		name  string
		path  string
		names []string
	}{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := LoadPackage(context.Background(), testCase.path, "", testCase.names)
			if err == nil {
				t.FailNow()
			}
		})
	}
}

func TestParserSuccessSub(t *testing.T) {
	ctx := context.Background()
	path := "github.com/kevinconway/wrapgen/v2/internal/test/sub/happy"
	names := []string{
		"Demo",
	}
	pkg, err := LoadPackage(ctx, path, "happy", names)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(pkg.Interfaces) < len(names) {
		t.Fatalf("did not render all interfaces: %v", pkg.Interfaces)
	}
	if pkg.Name != "happy" {
		t.Fatalf("did not use correct output package name: %s", pkg.Name)
	}
	found := false
	for _, imp := range pkg.Imports {
		if imp.Path == path && imp.Package == sourceAlias {
			found = true
		}
	}
	if !found {
		t.Fatalf("did not inject a source package alias: %v", pkg.Imports)
	}
}
