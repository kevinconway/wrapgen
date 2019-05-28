// Package happy contains all possible success cases that we expect to see.
// nolint
package happy

import (
	"io"
	nethttp "net/http"
	"os"

	"github.com/spf13/pflag"
)


type unexportedStruct struct {
	A string
	B int
	C chan bool
	D []uint64
}

type unexportedInterface interface {
	A()
	B(int, bool) (string, error)
	C(one int, two bool) (string, error)
	D(one unexportedStruct, two *unexportedStruct) error
}

type unexportedInterfaceWithEmbedded interface {
	unexportedInterface
}

type ExportedStruct struct {
	A string
	B int
	C chan bool
	D []uint64
}

type ExportedInterface interface {
	A()
	B(int, bool) (string, error)
	C(one int, two bool) (string, error)
	D(one ExportedStruct, two *ExportedStruct) error
	E(one func(), two func(int) bool) error
	F(one chan bool, two <-chan bool, three chan<- bool) error
	G(one []string, two [100]string) error
	H(one os.File, two *os.File) error
	I(one os.FileInfo) error
	J(one map[string]string) error
	K(one ...string) error
	L(one interface{}, two struct{}) error
	M(one nethttp.Handler) error
	N(one *pflag.FlagSet) error
}

type ExportedInterfaceWithEmbedded interface {
	ExportedInterface
}

type ExportedInterfaceWithRemoteEmbedded interface {
	io.Reader
}

type ExportedInterfaceWith3rdPartyEmbedded interface {
	pflag.Value
}

type InterfaceExtension ExportedInterface
type InterfaceAlias = ExportedInterface

type NonInterfaceExtension *pflag.FlagSet
type NonInterfaceAlias = pflag.FlagSet

type RemoteInterfaceExtension io.Reader
type RemoteInterfaceAlias = io.Reader

type ThirdPartyInterfaceExtension pflag.Value
type ThirdPartyInterfaceAlias = pflag.Value

type IndirectThirdPartyInterfaceExtension ThirdPartyInterfaceExtension
type IndirectThirdPartyInterfaceAlias ThirdPartyInterfaceAlias