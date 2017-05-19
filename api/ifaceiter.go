package wrapgen

import (
	"errors"
	"go/ast"
	"go/token"
)

// ErrIteratorComplete is used to check for when the iterator has exhausted.
var ErrIteratorComplete = errors.New("iterator complete")

// InterfaceIterator scans a file and produces ast.InterfaceType nodes.
type InterfaceIterator interface {
	Next() (string, *ast.InterfaceType, error)
}

// NewInterfaceIterator consumes the ast File object and produces an iterator
// for the contents.
func NewInterfaceIterator(source *ast.File) InterfaceIterator {
	return &interfaceIterator{
		source:       source,
		sourceOffset: 0,
		decl:         nil,
		declOffset:   0,
	}
}

type interfaceIterator struct {
	source       *ast.File
	sourceOffset int
	decl         ast.Decl
	declOffset   int
}

func (i *interfaceIterator) Next() (string, *ast.InterfaceType, error) {
	for {
		if i.sourceOffset >= len(i.source.Decls) {
			return "", nil, ErrIteratorComplete
		}
		if i.decl == nil {
			i.decl = i.source.Decls[i.sourceOffset]
			i.sourceOffset = i.sourceOffset + 1
			i.declOffset = 0
		}

		var gd, ok = i.decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			i.decl = nil
			continue
		}
		// Specs are type definitions. filter to only those
		for {
			var ok bool
			if i.declOffset >= len(gd.Specs) {
				i.decl = nil
				break
			}
			var spec = gd.Specs[i.declOffset]
			i.declOffset = i.declOffset + 1
			var ts *ast.TypeSpec
			ts, ok = spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			// Filter for only type definitions that are also interfaces
			var it *ast.InterfaceType
			it, ok = ts.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}
			return ts.Name.String(), it, nil
		}
	}
}

// InterfaceMapper converts and InterfaceIterator into a map of name -> *ast.InterfaceType.
type InterfaceMapper interface {
	Map(InterfaceIterator) map[string]*ast.InterfaceType
}

// NewInterfaceMapper generates a default implementation of the InterfaceMapper.
func NewInterfaceMapper() InterfaceMapper {
	return &interfaceMapper{}
}

type interfaceMapper struct{}

func (m *interfaceMapper) Map(iterator InterfaceIterator) map[string]*ast.InterfaceType {
	var results = make(map[string]*ast.InterfaceType)
	for name, iface, e := iterator.Next(); e != ErrIteratorComplete; name, iface, e = iterator.Next() {
		results[name] = iface
	}
	return results
}
