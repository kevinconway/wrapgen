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
	path := "./test/sub/happy"
	names := []string{
		"Demo",
	}
	pkg, err := LoadPackage(ctx, path, "", names)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(pkg.Interfaces) < len(names) {
		t.Fatalf("did not render all interfaces: %v", pkg.Interfaces)
	}
}
