package wrapgen

import (
	"testing"
)

func TestModels(t *testing.T) {
	var cases = []struct {
		name     string
		typ      Type
		expected string
	}{
		{"builtin", TypeBuiltin("bool"), "bool"},
		{"exported", &TypeExported{Package: "testpkg", Type: TypeBuiltin("test")}, "testpkg.test"},
		{"array as slice", &TypeArray{Len: -1, Type: TypeBuiltin("bool")}, "[]bool"},
		{"array with len", &TypeArray{Len: 10, Type: TypeBuiltin("bool")}, "[10]bool"},
		{"chan no direction", &TypeChan{ReadOnly: false, WriteOnly: false, Type: TypeBuiltin("bool")}, "chan bool"},
		{"chan read", &TypeChan{ReadOnly: true, WriteOnly: false, Type: TypeBuiltin("bool")}, "<-chan bool"},
		{"chan write", &TypeChan{ReadOnly: false, WriteOnly: true, Type: TypeBuiltin("bool")}, "chan<- bool"},
		{"variadic", &TypeVariadic{Type: TypeBuiltin("bool")}, "...bool"},
		{"func no in or out", &TypeFunc{}, "func()"},
		{"func with in", &TypeFunc{In: []Type{TypeBuiltin("bool"), TypeBuiltin("int")}}, "func(bool, int)"},
		{"func with out", &TypeFunc{Out: []Type{TypeBuiltin("bool"), TypeBuiltin("error")}}, "func() (bool, error)"},
		{"map", &TypeMap{Key: TypeBuiltin("string"), Value: TypeBuiltin("int")}, "map[string]int"},
		{"pointer", &TypePointer{Type: TypeBuiltin("uint64")}, "*uint64"},
	}

	for _, tcase := range cases {
		t.Run(tcase.name, func(t *testing.T) {
			var result = tcase.typ.String()
			if result != tcase.expected {
				t.Errorf("expected '%s' but got '%s'", tcase.expected, result)
			}
		})
	}
}
