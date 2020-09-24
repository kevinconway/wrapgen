package wrapgen

import (
	"context"
	"fmt"
	"strings"
)

// Type is a Go type definition that can be rendered into a valid
// Go code snippet.
type Type interface {
	String() string
}

// TypeBuiltin is a built in Go type such as "string" or "bool".
type TypeBuiltin string

func (t TypeBuiltin) String() string { return string(t) }

// TypeExported is a user defined type that is exported from a package.
type TypeExported struct {
	Package string
	Type    Type
}

func (t *TypeExported) String() string {
	return fmt.Sprintf("%s.%s", t.Package, t.Type.String())
}

// TypeArray is a slice or array type.
type TypeArray struct {
	Len  int
	Type Type
}

func (t *TypeArray) String() string {
	var result = "[]"
	if t.Len > -1 {
		result = fmt.Sprintf("[%d]", t.Len)
	}
	return result + t.Type.String()
}

// TypeChan is a channel type.
type TypeChan struct {
	ReadOnly  bool
	WriteOnly bool
	Type      Type
}

func (t *TypeChan) String() string {
	var result = "chan "
	if t.ReadOnly {
		result = "<-chan "
	}
	if t.WriteOnly {
		result = "chan<- "
	}
	return result + t.Type.String()
}

// TypeVariadic is any type that is prefixed by Ellipsis.
type TypeVariadic struct {
	Type Type
}

func (t *TypeVariadic) String() string {
	return "..." + t.Type.String()
}

// TypeFunc is an input type of a function.
type TypeFunc struct {
	In  []Type
	Out []Type
}

func (t *TypeFunc) String() string {
	var outParams []string
	var inParams []string

	for _, param := range t.In {
		inParams = append(inParams, param.String())
	}
	for _, param := range t.Out {
		outParams = append(outParams, param.String())
	}
	var outString = strings.Join(outParams, ", ")
	if len(outParams) <= 1 {
		outString = " " + outString
	}
	if len(outParams) > 1 {
		outString = " (" + outString + ")"
	}
	var inString = strings.Join(inParams, ", ")
	return strings.TrimSpace("func(" + inString + ")" + outString)
}

// TypeMap is a user defined map type.
type TypeMap struct {
	Key   Type
	Value Type
}

func (t *TypeMap) String() string {
	return fmt.Sprintf("map[%s]%s", t.Key.String(), t.Value.String())
}

// TypePointer is a pointer to another type.
type TypePointer struct {
	Type Type
}

func (t *TypePointer) String() string {
	return "*" + t.Type.String()
}

// Package is a container for all exported interfaces of a Go package.
type Package struct {
	Name       string
	Source     *Import
	Interfaces []*Interface
	Imports    []*Import
}

// Import is a package name and path that is imported by another package.
type Import struct {
	Package string
	Path    string
}

// Interface is an exported interface defined in a package.
type Interface struct {
	SrcType Type // e.g. srcPkgAlias.ExportedType
	Name    string
	Methods []*Method
}

// Method is a named function attached to an interface.
type Method struct {
	Name string
	In   []*Parameter
	Out  []*Parameter
}

// Parameter is a named parameter used by a Method.
type Parameter struct {
	Name string
	Type Type
}

// TemplateFetcher is used to load a template from some source.
type TemplateFetcher interface {
	FetchTemplate(ctx context.Context, path string) (string, error)
}
