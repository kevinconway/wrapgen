package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	wrapgen "github.com/kevinconway/wrapgen/v2/internal"
	"github.com/spf13/pflag"
)

func main() {
	ctx := context.Background()
	fs := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	srcPkg := fs.String("source", "", "The import path of the package to render.")
	destPkg := fs.String("package", "", "The destination package path or name that the resulting file will be in. Defaults to the source package.")
	templatePath := fs.String("template", "", "The template to render.")
	ifaceName := fs.StringSlice("interface", nil, "The name of the interface to render.")
	leftDelim := fs.String("leftdelim", "#!", "Left-hand side delimiter for the template.")
	rightDelim := fs.String("rightdelim", "!#", "Right-hand side delimiter for the template.")
	timeout := fs.Duration("timeout", time.Minute, "Maximum runtime allowed for rendering.")
	destination := fs.String("destination", "-", "Filename for the rendered template. Defaults to STDOUT.")
	_ = fs.Parse(os.Args[1:])

	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			_, _ = fmt.Fprintln(os.Stderr, "command timed out")
			os.Exit(1)
		}
	}()

	if *templatePath == "" {
		fmt.Fprintln(os.Stderr, "no --template value set")
		os.Exit(1)
	}
	if *srcPkg == "" {
		fmt.Fprintln(os.Stderr, "no --source value set")
		os.Exit(1)
	}
	var output io.Writer = os.Stdout
	if len(*ifaceName) < 1 {
		fmt.Fprintln(os.Stderr, "no --interface value set")
		os.Exit(1)
	}

	fetcher := wrapgen.MultiTemplateFetcher{
		&wrapgen.HTTPTemplateFetcher{
			Client: http.DefaultClient,
		},
		&wrapgen.FSTemplateFetcher{
			ReadFn: ioutil.ReadFile,
		},
	}
	templateString, err := fetcher.FetchTemplate(ctx, *templatePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch template: %v\n", err)
		os.Exit(1)
	}
	tmpl, err := template.New("wrapgen").Funcs(sprig.TxtFuncMap()).Delims(*leftDelim, *rightDelim).Parse(templateString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse template: %v\n", err)
		os.Exit(1)
	}

	pkg, err := wrapgen.LoadPackage(ctx, *srcPkg, *destPkg, *ifaceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to interpret package: %v\n", err)
		os.Exit(1)
	}

	var buff bytes.Buffer
	if err := tmpl.Execute(&buff, pkg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to render template: %v\n", err)
		os.Exit(1)
	}

	if *destination != "-" {
		f, err := os.Create(*destination)
		if err != nil {
			fmt.Fprintf(os.Stderr, "faild to create destination file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		output = f
	}

	_, _ = io.Copy(output, &buff)
}
