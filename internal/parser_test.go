package wrapgen

import (
	"context"
	"fmt"
	"reflect"
	"sort"
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
		t.Fatalf("did not inject a source package alias: %v", getImportsPaths(pkg.Imports))
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
		t.Fatalf("did not inject a source package alias: %v", getImportsPaths(pkg.Imports))
	}
}

func TestParserUniqueImports(t *testing.T) {
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
	importsMap := map[string]bool{}
	for _, imp := range pkg.Imports {
		importsMap[imp.Package] = true
	}
	if len(pkg.Imports) != len(importsMap) {
		t.Fatalf("expected imports to be deduplicated: %v", getImportsPaths(pkg.Imports))
	}
}

func TestParserImportsWithoutSource(t *testing.T) {
	ctx := context.Background()
	path := "./test/happy"
	names := []string{
		"ExportedInterfaceWithRemoteEmbedded",
	}
	pkg, err := LoadPackage(ctx, path, "", names)
	if err != nil {
		t.Fatal(err.Error())
	}
	importsPaths := getImportsPaths(pkg.Imports)
	expectedImportsPaths := []string{
		"io:io",
	}
	if !reflect.DeepEqual(importsPaths, expectedImportsPaths) {
		t.Fatalf("unexpected imports: %v", importsPaths)
	}
	importsWithSourcePaths := getImportsPaths(pkg.ImportsWithSource)
	expectedImportsWithSourcePaths := []string{
		"io:io",
	}
	if !reflect.DeepEqual(importsWithSourcePaths, expectedImportsWithSourcePaths) {
		t.Fatalf("unexpected imports with source: %v", importsWithSourcePaths)
	}
}

func TestParserImportsWithoutSourceDest(t *testing.T) {
	ctx := context.Background()
	path := "./test/happy"
	names := []string{
		"ExportedInterfaceWithRemoteEmbedded",
	}
	pkg, err := LoadPackage(ctx, path, "wrappers", names)
	if err != nil {
		t.Fatal(err.Error())
	}
	importsPaths := getImportsPaths(pkg.Imports)
	expectedImportsPaths := []string{
		"io:io",
	}
	if !reflect.DeepEqual(importsPaths, expectedImportsPaths) {
		t.Fatalf("unexpected imports: %v", importsPaths)
	}
	importsWithSourcePaths := getImportsPaths(pkg.ImportsWithSource)
	expectedImportsWithSourcePaths := []string{
		"io:io",
		"srcPkgAlias:github.com/kevinconway/wrapgen/v2/internal/test/happy",
	}
	if !reflect.DeepEqual(importsWithSourcePaths, expectedImportsWithSourcePaths) {
		t.Fatalf("unexpected imports with source: %v", importsWithSourcePaths)
	}
}

func TestParserImportsWithSource(t *testing.T) {
	ctx := context.Background()
	path := "./test/happy"
	names := []string{
		"ExportedInterface",
	}
	pkg, err := LoadPackage(ctx, path, "", names)
	if err != nil {
		t.Fatal(err.Error())
	}
	importsPaths := getImportsPaths(pkg.Imports)
	expectedImportsPaths := []string{
		"http:net/http",
		"os:os",
		"pflag:github.com/spf13/pflag",
	}
	if !reflect.DeepEqual(importsPaths, expectedImportsPaths) {
		t.Fatalf("unexpected imports: %v", importsPaths)
	}
	importsWithSourcePaths := getImportsPaths(pkg.ImportsWithSource)
	expectedImportsWithSourcePaths := []string{
		"http:net/http",
		"os:os",
		"pflag:github.com/spf13/pflag",
	}
	if !reflect.DeepEqual(importsWithSourcePaths, expectedImportsWithSourcePaths) {
		t.Fatalf("unexpected imports with source: %v", importsWithSourcePaths)
	}
}

func TestParserImportsWithSourceDest(t *testing.T) {
	ctx := context.Background()
	path := "./test/happy"
	names := []string{
		"ExportedInterface",
	}
	pkg, err := LoadPackage(ctx, path, "wrappers", names)
	if err != nil {
		t.Fatal(err.Error())
	}
	importsPaths := getImportsPaths(pkg.Imports)
	expectedImportsPaths := []string{
		"http:net/http",
		"os:os",
		"pflag:github.com/spf13/pflag",
		"srcPkgAlias:github.com/kevinconway/wrapgen/v2/internal/test/happy",
	}
	if !reflect.DeepEqual(importsPaths, expectedImportsPaths) {
		t.Fatalf("unexpected imports: %v", importsPaths)
	}
	importsWithSourcePaths := getImportsPaths(pkg.ImportsWithSource)
	expectedImportsWithSourcePaths := []string{
		"http:net/http",
		"os:os",
		"pflag:github.com/spf13/pflag",
		"srcPkgAlias:github.com/kevinconway/wrapgen/v2/internal/test/happy",
	}
	if !reflect.DeepEqual(importsWithSourcePaths, expectedImportsWithSourcePaths) {
		t.Fatalf("unexpected imports with source: %v", importsWithSourcePaths)
	}
}

func getImportsPaths(imports []*Import) []string {
	importsPaths := make([]string, len(imports))
	for i, imp := range imports {
		importsPaths[i] = fmt.Sprintf("%s:%s", imp.Package, imp.Path)
	}
	sort.Strings(importsPaths)
	return importsPaths
}
