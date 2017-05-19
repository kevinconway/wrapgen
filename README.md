# wrapgen - A code generator for Google-golang interfaces

This project is a fork/derivative of <https://github.com/golang/mock> which
contains a tool called `mockgen` that can consume any valid Google-golang
interface and generate a mock version of it. `wrapgen` extends that concept
to allow for any custom code generation based on interfaces. Given a valid
template, using the `text/template` format, and a valid Google-golang package
`wrapgen` will parse out all interfaces in the package and inject their content into
your template.

## Usage

```bash
go get github.com/kevinconway/wrapgen
go get golang.org/x/tools/cmd/goimports

go run wrapgen.go \
  -t "${GOPATH}/src/github.com/kevinconway/wrapgen/basetemplate.txt" \
  -p "github.com/kevinconway/wrapgen/api" \
  | ${GOPATH}/bin/goimports > wrappers.go
```

All output are written to stdout. Depending on the template, code may not
be rendered with the correct format or with extraneous import statements.
Because of this, you very likely will need to run the output through a
formatter like `gofmt` or `goimports` before the output will build.

Template paths can either be a path on the filesystem or an HTTP(S)
location.

## Writing Templates

This project comes with a few templates in the root of the repository. The
`basicdecorator.txt` template will create a generic shell that wraps every
exposed method of the interfaces found in a package and the `gomock.txt`
template will generate an equivalent ouput to the `mockgen` command from the
gomock project.

In order to make templates that conflict the least amount with actual
Google-golang code, the default template delimiters are set to `#!` and `!#`.
However, these values are only the defaults and can be overridden by using
the `--rightdelim` and `--leftdelim` flags to match the delimiters in your
own template.

The `basicdecorator.txt` template is provided as a starting point for creating
interface decorators. For example, you can wrap your interface in debug logging
during development by adding some content like the following:

```
package wrappers

import (
	"#! .Source !#"
	#! range .Package.Imports !##! .Package !# "#! .Path !#"
	#! end !#
)

#! $pkgName := .Package.Name !##! range .Package.Interfaces !#type Wraps#! .Name !# struct {
	wrapped #! $pkgName !#.#! .Name !#
}
#! $ifaceRef := . !##! range .Methods !#func (w *Wraps#! $ifaceRef.Name !#) #! .Name !#(#! $methodRef := . !##! range $x, $e := .In !##! $e.Name !# #! $e.Type !##! if ne $x (add (len $methodRef.In) -1)!#, #! end !##! end !#) (#! $methodRef := . !##! range $x, $e := .Out !##! $e.Type !##! if ne $x (add (len $methodRef.Out) -1)!#, #! end !##! end !#) {

  var start = time.Now()
  log.Println("starting func #! .Name !#")
  defer func(start time.Time){
      log.Printf("ended func #! .Name !# after %f seconds \n", time.Since(start).Seconds())
  }(start)

	#! if ne (len $methodRef.Out) 0!#return #! end !#w.#! .Name !#(#! $methodRef := . !##! range $x, $e := .In !##! $e.Name !##! if ne $x (add (len $methodRef.In) -1)!#, #! end !##! end !#)
}
#! end !#
#! end !#
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

Templates have a top level object injected that contains a `#!.Source!#` and
`#!.Package!#` attribute. The `#!.Source!#` attribute is a string that
contains the import path of the package given at runtime that contains all of
the source interfaces. The `#!.Package!#` attribute is a `Package` model which
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

## License

This project is available under the Apache2.0 license. See the `LICENSE` file
in this repository for the complete license text.

