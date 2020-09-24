package wrapgen

import (
	"testing"
	"context"
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
	_, interfaces, err := LoadInterfaces(ctx, path, "", names)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(interfaces) < len(names) {
		t.Fatalf("did not render all interfaces: %v", interfaces)
	}
}

func TestParserFailure(t *testing.T) {
	testCases := []struct{
		name string
		path string
		names []string
	}{

	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, _, err := LoadInterfaces(context.Background(), testCase.path, "", testCase.names)
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
	_, interfaces, err := LoadInterfaces(ctx, path, "", names)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(interfaces) < len(names) {
		t.Fatalf("did not render all interfaces: %v", interfaces)
	}
}