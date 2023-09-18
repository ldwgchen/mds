package main

import (
  "os"
  "log"
  "bytes"
  "net/http"
  "path/filepath"
  "github.com/yuin/goldmark"
  "github.com/yuin/goldmark/extension"
  "github.com/yuin/goldmark/parser"
  "github.com/yuin/goldmark/renderer/html"
  "go.abhg.dev/goldmark/wikilink"
  "go.abhg.dev/goldmark/frontmatter"
)

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

func serveMarkdown(w http.ResponseWriter, abs string) {
  in, _ := os.ReadFile(abs)
  w.Header().Add("Content-Type", "text/html")
  w.Write(c.header)
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
  w.Write(c.footer)
}
