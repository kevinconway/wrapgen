# wrapgen - A code wrapper generator for Google-golang interfaces

This project is a fork/derivative of <https://github.com/golang/mock> which
contains a tool called `mockgen` that can consume any valid Google-golang
interface and generate a mock version of it. `wrapgen` extends that concept
to allow for custom code generation based on interfaces. Given a valid template,
using the `text/template` format, and a valid Google-golang package `wrapgen`
will parse out all interfaces in the package and inject their content into
your template.

The goal of this project is to make it easy to apply generic code patterns
such as logging, stats, retry policies, circuit breaking, etc. Teams should be
able to organise these operational patterns into one or more wrappers that
apply seamlessly to existing code.

## Usage

```bash
go get github.com/kevinconway/wrapgen
go get golang.org/x/tools/cmd/goimports

go run wrapgen.go \
  -t "${GOPATH}/src/github.com/kevinconway/wrapgen/basetemplate.txt" \
  -p "github.com/kevinconway/wrapgen/api" \
  | ${GOPATH}/bin/goimports > wrappers.go
```

All output are written to stdout. It's suggested to run the result through
a formatter like `gofmt` or `goimports`

## Writing Templates

This project comes with a basic template in the root of the repository. This
base template will create a generic shell that wraps every exposed method of
the interface. From this base, you can extend the template with your own
content. For example, you could add debug logging around your method calls by
doing something like this to the template:

```
package wrappers

import (
	"{{ .Source }}"
	{{ range .Package.Imports }}{{ .Package }} "{{ .Path }}"
	{{ end }}
)

{{ $pkgName := .Package.Name }}{{ range .Package.Interfaces }}type Wraps{{ .Name }} struct {
	wrapped {{ $pkgName }}.{{ .Name }}
}
{{ $ifaceRef := . }}{{ range .Methods }}func (w *Wraps{{ $ifaceRef.Name }}) {{ .Name }}({{ $methodRef := . }}{{ range $x, $e := .In }}{{ $e.Name }} {{ $e.Type }}{{ if ne $x (add (len $methodRef.In) -1)}}, {{ end }}{{ end }}) ({{ $methodRef := . }}{{ range $x, $e := .Out }}{{ $e.Type }}{{ if ne $x (add (len $methodRef.Out) -1)}}, {{ end }}{{ end }}) {

  var start = time.Now()
  log.Println("starting func {{ .Name }}")
  defer func(start time.Time){
      log.Printf("ended func {{ .Name }} after %f seconds \n", time.Since(start).Seconds())
  }(start)

	{{ if ne (len $methodRef.Out) 0}}return {{ end }}w.{{ .Name }}({{ $methodRef := . }}{{ range $x, $e := .In }}{{ $e.Name }}{{ if ne $x (add (len $methodRef.In) -1)}}, {{ end }}{{ end }})
}
{{ end }}
{{ end }}
```

That template, given an interface like:

```go
package sourcepkg

type Doer interface {
  Do(r *http.Request) (*http.Response, error)
}
```

will result in code like:

```go
package wrappers

import (
  "time"
  "net/http"
  "sourcepkg"
)

type DoerWrapper struct {
  wrapped sourcepkg.Doer
}

func (w *DoerWrapper) Do(r *http.Request) (*http.Response, error) {
  var start = time.Now()
  log.Println("starting func Do")
  defer func(start time.Time){
      log.Printf("ended func Do after %f seconds \n", time.Since(start).Seconds())
  }(start)
  return w.wrapped.Do(r)
}
```

Templates have a top level object injected that contains a `{{.Source}}` and
`{{.Package}}` attribute. The `{{.Source}}` attribute is a string that
contains the import path of the package given at runtime that contains all of
the source interfaces. The `{{.Package}}` attribute is a `Package` model which
contains the following:

```
type Package struct {
	Name       string
	Interfaces []*Interface
	Imports    []*Import
}
    Package is a container for all exported interfaces of a Google-golang
    package.

type Import struct {
	Package string
	Path    string
}
    Import is a package name and path that is imported by another package.

type Interface struct {
	Name    string
	Methods []*Method
}
    Interface is an exported interface defined in a package.

type Method struct {
	Name string
	In   []*Parameter
	Out  []*Parameter
}
    Method is a named function attached to an interface.

type Parameter struct {
	Name string
	Type Type
}
    Parameter is a named parameter used by a Method.

type Type interface {
	String() string
}
    Type is a Google-golang type definition that can be rendered into a valid
    Google-golang code snippet.
```

