# Markdown to HTML cli tool

## usage

If you have Go installed, install with:

    go get -u github.com/ttys3/md2html

To run:

    md2html [options] inputfile [outputfile]

or

    cat inputfile | md2html [options]

Run `md2html -h` to see all options.

style preview: https://xyproto.github.io/splash/docs/

available styles:

```
abap
algol
algol_nu
api
arduino
autumn
borland
bw
colorful
dracula
emacs
friendly
fruity
github
igor
lovelace
manni
monokai
monokailight
murphy
native
paraiso-dark
paraiso-light
pastie
perldoc
pygments
rainbow_dash
rrt
solarized-dark
solarized-dark256
solarized-light
swapoff
tango
trac
vim
vs
xcode
```

## thanks

[yuin/goldmark](https://github.com/yuin/goldmark) library

[alecthomas/chroma](https://github.com/alecthomas/chroma) library


this project is inspired by [mdtohtml](https://github.com/gomarkdown/mdtohtml)
