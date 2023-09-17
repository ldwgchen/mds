package main

import (
  "bytes"
	"flag"
  "log"
	"net/http"
  "io"
  "os"
  "path/filepath"
  "github.com/yuin/goldmark"
  "github.com/yuin/goldmark/extension"
  "github.com/yuin/goldmark/parser"
  "github.com/yuin/goldmark/renderer/html"
  "go.abhg.dev/goldmark/wikilink"
  "go.abhg.dev/goldmark/frontmatter"
)

type Conditions struct {
  addr *string
  root string
  favicon []byte
  header []byte
  footer []byte
}

type CustomWikilinkResolver struct {}

func (CustomWikilinkResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
  var _md = []byte(".md")
	var _hash = []byte{'#'}
	dest := make([]byte, len(n.Target)+len(_md)+len(_hash)+len(n.Fragment))
	var i int
	if len(n.Target) > 0 {
		i += copy(dest, n.Target)
		if filepath.Ext(string(n.Target)) == "" {
			i += copy(dest[i:], _md)
		}
	}
	if len(n.Fragment) > 0 {
		i += copy(dest[i:], _hash)
		i += copy(dest[i:], n.Fragment)
	}
	return dest[:i], nil
}

var (
  c Conditions
  wikilinkResolver CustomWikilinkResolver
)

func makeConditions(c *Conditions) {
  c.addr = flag.String("addr", "127.0.0.1:5555", "address to listen on, defaulted to localhost:5555")
  flag.Parse()

  var err error
  
  if c.root, err = filepath.Abs(flag.Arg(0)); err != nil {
    log.Fatalln("FATAL setting root path")
  }
  log.Println("root path is set to", c.root)
  if c.favicon, err = Asset("data/favicon.ico"); err != nil {
    log.Fatalln("FATAL retrieving favicon")
  }
  if c.header, err = Asset("data/header.html"); err != nil {
    log.Fatalln("FATAL retrieving header")
  }
  if c.footer, err = Asset("data/footer.html"); err != nil {
    log.Fatalln("FATAL retrieving footer")
  }
}

func main() {
  makeConditions(&c)
  mux := http.NewServeMux()
  mux.HandleFunc("/favicon.ico", handleFavicon)
  mux.HandleFunc("/", handleFile)
  server := &http.Server{
    Addr: *c.addr,
    Handler: mux,
  }
  server.ListenAndServe()
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
  w.Write(c.favicon)
  log.Println("served favicon to", r.RemoteAddr)
}

func handleFile(w http.ResponseWriter, r *http.Request) {
  log.Println(r.RemoteAddr, "->", r.URL.Path)
  p := filepath.Join(c.root, r.URL.Path)
  fileInfo, err := os.Stat(p)
  if err != nil {
    log.Println("error accessing", p)
		http.NotFound(w, r)
    return
  }

  if fileInfo.IsDir() {
		w.Header().Add("Content-Type", "text/html")
    w.Write(c.header)
    http.ServeFile(w, r, p)
    w.Write(c.footer)
    log.Println("served directory", p)
    return
  }

  if filepath.Ext(p) == ".md" {
    b, _ := os.ReadFile(p)
		w.Header().Add("Content-Type", "text/html")
    w.Write(c.header)
    markdownToHTML(b, w)
    w.Write(c.footer)
    log.Println("served markdown", filepath.Base(p))
    return
  }

  http.ServeFile(w, r, p)
  log.Println("served file", filepath.Base(p))
}

func markdownToHTML(in []byte, w io.Writer) {
  md := goldmark.New(
    goldmark.WithExtensions(extension.GFM, extension.DefinitionList,
      extension.Footnote, &wikilink.Extender{Resolver: wikilinkResolver},
      &frontmatter.Extender{}),
    goldmark.WithParserOptions(
      parser.WithAutoHeadingID(),
      ),
    goldmark.WithRendererOptions(
      html.WithHardWraps(),
      html.WithXHTML(),
      ),
    )
  var b, prependix bytes.Buffer
  ctx := parser.NewContext()
  if err := md.Convert(in, &b, parser.WithContext(ctx)); err != nil {
    log.Println("problem converting markdown")
    return
  }

  d := frontmatter.Get(ctx)
  if d != nil {
    var meta struct {
      Title string
      Keywords []string
    }
    if err := d.Decode(&meta); err != nil {
      log.Println("problem decoding frontmatter")
    }
    if meta.Title != "" {
      prependix.WriteString(`<h1 id="title">` + meta.Title + "</h1>\n")
    }
    if len(meta.Keywords) > 1 {
      prependix.WriteString("<p><strong>Keywords:</strong> " + meta.Keywords[0])
      for i := 1; i < len(meta.Keywords); i++ {
        prependix.WriteString("<strong>;</strong> " + meta.Keywords[i])
      }
      prependix.WriteString("</p>\n")
    } else if len(meta.Keywords) == 1 {
      prependix.WriteString("<p><strong>Keywords:</strong> " + meta.Keywords[0] + "</p>\n")
    }
  }

  if prependix.Len() != 0 {
    prependix.WriteTo(w)
  }
  b.WriteTo(w)
}
