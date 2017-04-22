package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	wrapgen "github.com/kevinconway/wrapgen/api"
	"github.com/urfave/cli"
)

func defaultGOPATH() string {
	var env = "HOME"
	if runtime.GOOS == "windows" {
		env = "USERPROFILE"
	}
	if runtime.GOOS == "plan9" {
		env = "home"
	}
	if home := os.Getenv(env); home != "" {
		var def = filepath.Join(home, "go")
		if filepath.Clean(def) == filepath.Clean(runtime.GOROOT()) {
			return ""
		}
		return def
	}
	return ""
}

func gopath() string {
	var p = os.Getenv("GOPATH")
	if p == "" {
		return defaultGOPATH()
	}
	return p
}

type pkgWrapper struct {
	Source  string
	Package *wrapgen.Package
}

func render(templateString string, sourcePath string, pkg *wrapgen.Package) (string, error) {
	var t, e = template.New("wrapgen").Funcs(sprig.TxtFuncMap()).Parse(templateString)
	if e != nil {
		return "", e
	}
	var result = &bytes.Buffer{}
	e = t.Execute(result, pkgWrapper{Package: pkg, Source: sourcePath})
	return result.String(), e
}

func getPackage(sourcePath string) (*wrapgen.Package, error) {
	var fullPath = filepath.Join(gopath(), "src", sourcePath)
	var parser = wrapgen.NewParser()
	var pkg, e = parser.ParsePackage(fullPath)
	return pkg, e
}

func renderLocal(templatePath string, sourcePath string, pkg *wrapgen.Package) (string, error) {
	var file, e = os.Open(templatePath)
	if e != nil {
		return "", e
	}
	var templateString []byte
	templateString, e = ioutil.ReadAll(file)
	if e != nil {
		return "", e
	}
	return render(string(templateString), sourcePath, pkg)
}

func renderRemote(href string, sourcePath string, pkg *wrapgen.Package) (string, error) {
	var resp, e = http.Get(href)
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	var templateString []byte
	templateString, e = ioutil.ReadAll(resp.Body)
	if e != nil {
		return "", e
	}
	return render(string(templateString), sourcePath, pkg)
}

func main() {
	var app = cli.NewApp()
	app.Name = "wrapgen"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "template,t",
			Value: "https://raw.githubusercontent.com/kevinconway/wrapgen/master/basetemplate.txt",
			Usage: "The HREF or source path that contains a valid `TEMPLATE`",
		},
		cli.StringFlag{
			Name:  "package,p",
			Value: "",
			Usage: "The import path of the `PACKAGE` to render",
		},
	}
	app.Action = func(c *cli.Context) error {
		var templatePath = c.String("template")
		var sourcePath = c.String("package")
		if c.NArg() > 0 {
			templatePath = c.Args().Get(0)
		}
		if c.NArg() > 1 {
			templatePath = c.Args().Get(1)
		}
		var pkg, e = getPackage(sourcePath)
		if e != nil {
			return cli.NewExitError(e.Error(), 1)
		}
		var renderer func(string, string, *wrapgen.Package) (string, error)
		renderer = renderLocal
		if strings.HasPrefix(strings.ToLower(templatePath), "http") {
			renderer = renderRemote
		}
		var result string
		result, e = renderer(templatePath, sourcePath, pkg)
		if e != nil {
			return cli.NewExitError(e.Error(), 1)
		}
		_, _ = os.Stdout.Write([]byte(result))
		return nil
	}

	_ = app.Run(os.Args)
}
