package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"runtime/pprof"
	"strings"

	chromahtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed default-style.css
var defaultStyle string

const defaultTitle = "untitled"

const defaultCodeStyle = "monokailight"

var appVersion = "dev"

func main() {
	var page, showVersion bool
	var css, cpuprofile string
	var chromaStyle string

	flag.BoolVar(&page, "page", true,
		"Generate a standalone HTML page")
	flag.BoolVar(&showVersion, "v", false,
		"Show app version")
	flag.StringVar(&css, "css", "",
		"Link to a CSS stylesheet (implies -page)")
	flag.StringVar(&chromaStyle, "style", "",
		"Chroma style, see https://xyproto.github.io/splash/docs/ for full list")
	flag.StringVar(&cpuprofile, "cpuprofile", "",
		"Write cpu profile to a file")
	flag.Usage = func() {
		printVersion(os.Stderr)
		fmt.Fprintf(os.Stderr,
			"Usage:\n"+
				"  %s [options] [inputfile [outputfile]]\n\n"+
				"Options:\n",
			os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if showVersion {
		printVersion(os.Stdout)
		return
	}

	// enforce implied options
	if css != "" {
		page = true
		css = fmt.Sprintf(`<link crossorigin="anonymous" media="all" rel="stylesheet" href="%s" />`, css)
	}

	// turn on profiling?
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// read the input
	var input []byte
	var err error
	// the non-flag command-line arguments
	args := flag.Args()

	switch len(args) {
	case 0:
		if input, err = ioutil.ReadAll(os.Stdin); err != nil {
			fmt.Fprintln(os.Stderr, "Error reading from Stdin:", err)
			os.Exit(-1)
		}
	case 1, 2:
		if input, err = ioutil.ReadFile(args[0]); err != nil {
			fmt.Fprintln(os.Stderr, "Error reading from", args[0], ":", err)
			os.Exit(-1)
		}
	default:
		flag.Usage()
		os.Exit(-1)
	}

	title := ""
	if page {
		title = getTitle(input)
	}

	// parse and render
	inlineCodeCss := chromaStyle != ""
	markdown := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle(chromaStyle),
				highlighting.WithFormatOptions(
					chromahtml.WithLineNumbers(false),
					chromahtml.LineNumbersInTable(true),
					chromahtml.TabWidth(4),
					chromahtml.WithClasses(!inlineCodeCss),
				),
			),
		),
		goldmark.WithExtensions(extension.Typographer),
	)
	var output bytes.Buffer
	if err := markdown.Convert(input, &output); err != nil {
		panic(err)
	}

	// output the result
	var out *os.File
	if len(args) == 2 {
		if out, err = os.Create(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating %s: %v", args[1], err)
			os.Exit(-1)
		}
		defer out.Close()
	} else {
		out = os.Stdout
	}

	if page {
		tmpl, err := template.New("header").Parse(header)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parse template:", err)
		}
		if err := tmpl.Execute(out, struct {
			Title        string
			DefaultStyle template.CSS
			StyleLink    template.HTML
		}{Title: title, DefaultStyle: template.CSS(defaultStyle), StyleLink: template.HTML(css)}); err != nil {
			fmt.Fprintln(os.Stderr, "Error execute template:", err)
		}
	}
	out.WriteString(`<article class="markdown-body">`)
	if _, err = out.Write(output.Bytes()); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing output:", err)
		os.Exit(-1)
	}
	out.WriteString(`</article>`)
	if page {
		out.WriteString(footer)
	}
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "md2html %s"+
		"\nAvailable at http://github.com/ttys3/md2html \n\n"+
		"Copyright © 2020 荒野無燈 <https://ttys3.net>\n"+
		"Distributed under the Simplified BSD License\n\n", appVersion)
}

// try to guess the title from the input buffer
// just check if it starts with an <h1> element and use that
func getTitle(input []byte) string {
	i := 0

	// skip blank lines
	for i < len(input) && (input[i] == '\n' || input[i] == '\r') {
		i++
	}
	if i >= len(input) {
		return defaultTitle
	}
	if input[i] == '\r' && i+1 < len(input) && input[i+1] == '\n' {
		i++
	}

	// find the first line
	start := i
	for i < len(input) && input[i] != '\n' && input[i] != '\r' {
		i++
	}
	line1 := input[start:i]
	if input[i] == '\r' && i+1 < len(input) && input[i+1] == '\n' {
		i++
	}
	i++

	// check for a prefix header
	if len(line1) >= 3 && line1[0] == '#' && (line1[1] == ' ' || line1[1] == '\t') {
		return strings.TrimSpace(string(line1[2:]))
	}

	// check for an underlined header
	if i >= len(input) || input[i] != '=' {
		return defaultTitle
	}
	for i < len(input) && input[i] == '=' {
		i++
	}
	for i < len(input) && (input[i] == ' ' || input[i] == '\t') {
		i++
	}
	if i >= len(input) || (input[i] != '\n' && input[i] != '\r') {
		return defaultTitle
	}

	return strings.TrimSpace(string(line1))
}

const header = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
	<title>{{.Title}}</title>
	<style>
	   {{ .DefaultStyle }}
	</style>

	{{.StyleLink}}

	</head>
	<body>
`

const footer = `
</body></html>
`
