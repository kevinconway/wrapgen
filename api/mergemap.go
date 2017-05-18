package wrapgen

import "go/ast"

func mergeMapString(dst map[string]string, src map[string]string) {
	for k, v := range src {
		dst[k] = v
	}
}

func mergeMapIface(dst map[string]*ast.InterfaceType, src map[string]*ast.InterfaceType) {
	for k, v := range src {
		dst[k] = v
	}
}
