package main

import (
	"bytes"
	"flag"
	"fmt"
	chromahtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"html/template"
	"io/ioutil"
	"os"
	"runtime/pprof"
	"strings"
)

const defaultTitle = "untitled"

func main() {
	var page bool
	var css, cpuprofile string
	var chromaStyle string

	flag.BoolVar(&page, "page", false,
		"Generate a standalone HTML page")
	flag.StringVar(&css, "css", "",
		"Link to a CSS stylesheet (implies -page)")
	flag.StringVar(&chromaStyle, "style", "monokai",
		"Chroma style, see https://xyproto.github.io/splash/docs/ for full list")
	flag.StringVar(&cpuprofile, "cpuprofile", "",
		"Write cpu profile to a file")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Markdown Processor "+
			"\nAvailable at http://github.com/ttys3/md2html \n\n"+
			"Copyright © 2020 荒野無燈 <https://ttys3.net>\n"+
			"Distributed under the Simplified BSD License\n"+
			"Usage:\n"+
			"  %s [options] [inputfile [outputfile]]\n\n"+
			"Options:\n",
			os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

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
					chromahtml.WithClasses(page),
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
		tmpl, err := template.New("header").Parse(strings.ReplaceAll(header, "\n", ""))
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parse template:", err)
		}
		if err := tmpl.Execute(out, struct {
			Title string
			Style string
		}{Title: title, Style: css}); err != nil {
			fmt.Fprintln(os.Stderr, "Error execute template:", err)
		}
	}
	if _, err = out.Write(output.Bytes()); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing output:", err)
		os.Exit(-1)
	}
	if page {
		out.WriteString(footer)
	}
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
	/* Basic adjustments for text and link colors, etc. */
	body {
		margin: 0;
		padding: 0;
	}

	.markdown-body {
		color: #373a3c;
	}
	.markdown-body a {
		color: #0275d8;
		text-decoration: none;
	}
	.markdown-body a:focus, .markdown-body a:hover {
		color: #014c8c;
		text-decoration: underline;
	}
	
	code,pre{tab-size:4}
	
	tt,code,pre{
		font-family: 'JetBrains Mono', Consolas, Menlo, 'Fira Code',
					'Fantasque Sans Mono', 'Eco Coding', 'Envy Code R',
					'CosmicSansNeueMono', Monaco, 'Andale Mono', 'Ubuntu Mono',
					Courier, 'Courier New',
					monospace;
		font-size:14px
	}
	pre{margin-top:0;margin-bottom:0}
	
	blockquote{margin:0}
	
	table{border-collapse:collapse;border-spacing:0}
	td,th{padding:0}
	
	.markdown-body{
		overflow:hidden;
		font-family:"Helvetica Neue",Helvetica,"Segoe UI",Arial,freesans,sans-serif;
		font-size:14px;
		line-height:1.6;
		word-wrap:break-word
	}
	
	.markdown-body>*:first-child{
		margin-top:0 !important
	}
	
	.markdown-body>*:last-child{
		margin-bottom:0 !important
	}
	
	.markdown-body .absent{color:#c00}
	
	.markdown-body .anchor{position:absolute;top:10;bottom:0;left:0;display:block;padding-right:6px;padding-left:30px;margin-left:-30px}
	
	.markdown-body .anchor:focus{outline:none}
	
	.markdown-body h1,.markdown-body h2,.markdown-body h3,.markdown-body h4,.markdown-body h5,.markdown-body h6
	{
	position:relative;margin-top:1em;margin-bottom:16px;font-weight:bold;line-height:1.4}
	
	.markdown-body h1 .octicon-link,.markdown-body h2 .octicon-link,
	.markdown-body h3 .octicon-link,.markdown-body h4 .octicon-link,
	.markdown-body h5 .octicon-link,.markdown-body h6 .octicon-link {
		display:none;color:#000;vertical-align:middle
	}
	
	.markdown-body h1:hover .anchor,.markdown-body h2:hover .anchor,
	.markdown-body h3:hover .anchor,.markdown-body h4:hover .anchor,
	.markdown-body h5:hover .anchor,.markdown-body h6:hover .anchor {
		height:1em;padding-left:8px;margin-left:-30px;line-height:1;text-decoration:none
	}
	.markdown-body h1:hover .anchor .octicon-link,
	.markdown-body h2:hover .anchor .octicon-link,
	.markdown-body h3:hover .anchor .octicon-link,
	.markdown-body h4:hover .anchor .octicon-link,
	.markdown-body h5:hover .anchor .octicon-link,
	.markdown-body h6:hover .anchor .octicon-link {
		display:inline-block
	}
	.markdown-body h1 tt,.markdown-body h1 code,
	.markdown-body h2 tt,.markdown-body h2 code,
	.markdown-body h3 tt,.markdown-body h3 code,
	.markdown-body h4 tt,
	.markdown-body h4 code,
	.markdown-body h5 tt,.markdown-body h5 code,
	.markdown-body h6 tt,.markdown-body h6 code{
		font-size:inherit
	}
	.markdown-body h1{padding-bottom:0.3em;font-size:2.25em;line-height:1.2;border-bottom:1px solid #eee}
	.markdown-body h2{padding-bottom:0.3em;font-size:1.75em;line-height:1.225;border-bottom:1px solid #eee}
	.markdown-body h3{font-size:1.5em;line-height:1.43}
	.markdown-body h4{font-size:1.25em}
	.markdown-body h5{font-size:1em}
	.markdown-body h6{font-size:1em;color:#777}
	.markdown-body p,.markdown-body blockquote,.markdown-body ul,.markdown-body ol,
	.markdown-body dl,.markdown-body table,
	.markdown-body pre{ margin-top:0;margin-bottom:16px }
	.markdown-body hr{height:4px;padding:0;margin:16px 0;background-color:#e7e7e7;border:0 none}
	.markdown-body ul,.markdown-body ol{padding-left:2em}
	.markdown-body ul.no-list,.markdown-body ol.no-list{padding:0;list-style-type:none}
	.markdown-body ul ul,.markdown-body ul ol,.markdown-body ol ol,.markdown-body ol ul{margin-top:0;margin-bottom:0}
	.markdown-body li>p{margin-top:16px}
	.markdown-body dl{padding:0}
	.markdown-body dl dt{padding:0;margin-top:16px;font-size:1em;font-style:italic;font-weight:bold}
	.markdown-body dl dd{padding:0 16px;margin-bottom:16px}
	.markdown-body blockquote{padding:0 15px;color:#777;border-left:4px solid #ddd}
	.markdown-body blockquote>:first-child{margin-top:0}
	.markdown-body blockquote>:last-child{margin-bottom:0}
	.markdown-body table{display:block;width:100%;overflow:auto;word-break:normal;word-break:keep-all}
	.markdown-body table th{font-weight:bold}
	.markdown-body table th,.markdown-body table td{padding:6px 13px;border:1px solid #ddd}
	.markdown-body table tr{background-color:#fff;border-top:1px solid #ccc}
	.markdown-body table tr:nth-child(2n){background-color:#f8f8f8}
	.markdown-body img{max-width:100%;-moz-box-sizing:border-box;box-sizing:border-box}
	.markdown-body span.frame{display:block;overflow:hidden}
	.markdown-body span.frame>span{display:block;float:left;width:auto;padding:7px;margin:13px 0 0;overflow:hidden;border:1px solid #ddd}
	.markdown-body span.frame span img{display:block;float:left}
	.markdown-body span.frame span span{display:block;padding:5px 0 0;clear:both;color:#333}
	.markdown-body span.align-center{display:block;overflow:hidden;clear:both}
	.markdown-body span.align-center>span{display:block;margin:13px auto 0;overflow:hidden;text-align:center}
	.markdown-body span.align-center span img{margin:0 auto;text-align:center}
	.markdown-body span.align-right{display:block;overflow:hidden;clear:both}
	.markdown-body span.align-right>span{display:block;margin:13px 0 0;overflow:hidden;text-align:right}
	.markdown-body span.align-right span img{margin:0;text-align:right}
	.markdown-body span.float-left{display:block;float:left;margin-right:13px;overflow:hidden}
	.markdown-body span.float-left span{margin:13px 0 0}
	.markdown-body span.float-right{display:block;float:right;margin-left:13px;overflow:hidden}
	.markdown-body span.float-right>span{display:block;margin:13px auto 0;overflow:hidden;text-align:right}
	.markdown-body code,.markdown-body tt{padding:0;padding-top:0.2em;padding-bottom:0.2em;margin:0;font-size:100%;background-color:rgba(0,0,0,0.04);border-radius:3px}
	.markdown-body code:before,.markdown-body code:after,.markdown-body tt:before,.markdown-body tt:after{letter-spacing:-0.2em;content:"\00a0"}
	.markdown-body code br,.markdown-body tt br{display:none}
	.markdown-body del code{text-decoration:inherit}
	.markdown-body pre>code{padding:0;margin:0;font-size:100%;word-break:normal;white-space:pre;background:transparent;border:0}
	.markdown-body .highlight{margin-bottom:16px}
	.markdown-body .highlight pre,.markdown-body pre{padding:16px;overflow:auto;font-size:100%;line-height:1.45;background-color:#f7f7f7;border-radius:3px}
	.markdown-body .highlight pre{margin-bottom:0;word-break:normal}
	.markdown-body pre{word-wrap:normal}
	.markdown-body pre code,.markdown-body pre tt {
		display:inline;max-width:initial;padding:0;margin:0;
		overflow:initial;line-height:inherit;word-wrap:normal;
		background-color:transparent;border:0
	}
	.markdown-body pre code:before,.markdown-body pre code:after,.markdown-body pre tt:before,.markdown-body pre tt:after{content:normal}
	
	.highlight>.chroma {
	margin:1em 0;overflow-x:auto;position:relative;border:2px solid #ddd;line-height:1.6
	}
	
	.highlight>.chroma code{padding:0;color:inherit}
	.highlight>.chroma pre{margin:0}
	.highlight>.chroma table{position:relative;padding:.8em 0}
	.highlight>.chroma table::after{position:absolute;top:0;right:0;padding:0 7px;font-size:.8em;font-weight:700;color:#b1b1b1;content:'Code'}
	.highlight>.chroma>table::after{content:attr(data-lang);text-transform:capitalize}
	.highlight>.chroma .lnt{color:#cacaca}
	
	.chroma{color:#586e75;background-color:#f8f5ec}
	.chroma .lntd{vertical-align:top;padding:0;margin:0;border:0}
	.chroma .lntable{border-spacing:0;padding:0;margin:0;border:0;width:auto;overflow:auto;display:block}
	.chroma .hl{display:block;width:100%;background-color:#ffc}
	.chroma .lntd:first-of-type{margin-right:.4em;padding:0 .8em 0 .4em}
	.chroma .ln{margin-right:.4em;padding:0 .4em}
	
	.chroma .k, 
	.chroma .kd, 
	.chroma .kp, 
	.chroma .kr { color:#859900 }
	.chroma .kc, 
	.chroma .kn, 
	.chroma .kt { color:#859900;font-weight:700 }
	
	.chroma .n, 
	.chroma .na, 
	.chroma .bp, 
	.chroma .no, 
	.chroma .nd, 
	.chroma .ni, 
	.chroma .ne, 
	.chroma .nf, 
	.chroma .fm, 
	.chroma .nl, 
	.chroma .nn, 
	.chroma .nx, 
	.chroma .py, 
	.chroma .nv, 
	.chroma .vc, 
	.chroma .vg, 
	.chroma .vi, 
	.chroma .vm {color:#268bd2}
	
	.chroma .nt{color:#268bd2;font-weight:700}
	.chroma .nb{color:#cb4b16}
	.chroma .nc{color:#cb4b16}
	
	.chroma .l, 
	.chroma .ld, 
	.chroma .s, 
	.chroma .sa, 
	.chroma .sb, 
	.chroma .sc, 
	.chroma .dl, 
	.chroma .sd, 
	.chroma .s2, 
	.chroma .se, 
	.chroma .sh, 
	.chroma .si, 
	.chroma .sx, 
	.chroma .sr, 
	.chroma .s1, 
	.chroma .ss{color:#2aa198}
	
	.chroma .m, 
	.chroma .mb, 
	.chroma .mf, 
	.chroma .mh, 
	.chroma .mi, 
	.chroma .il, 
	.chroma .mo{color:#2aa198;font-weight:700}
	
	.chroma .ow{color:#859900}
	
	.chroma .c, 
	.chroma .ch, 
	.chroma .cm, 
	.chroma .c1, 
	.chroma .cs, 
	.chroma .cp, 
	.chroma .cpf{color:#93a1a1;font-style:italic}
	
	.chroma .g, 
	.chroma .gd, 
	.chroma .ge, 
	.chroma .gr, 
	.chroma .gh, 
	.chroma .gi, 
	.chroma .go, 
	.chroma .gp, 
	.chroma .gs, 
	.chroma .gu, 
	.chroma .gt{color:#d33682}
	</style>
	{{.Style}}
	</head>
	<body>
	<article class="markdown-body entry-content" style="padding: 30px;">
`
const footer = `
</article></body></html>
`
