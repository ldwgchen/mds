package main

import (
  "os"
  "log"
	"flag"
	"net/http"
  "path/filepath"
)

type Conditions struct {
  addr *string
  root string
  favicon []byte
  header []byte
  footer []byte
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
  urlPath := r.URL.Path
  log.Println(r.RemoteAddr, "->", urlPath)
  abs := filepath.Join(c.root, urlPath)
  fileInfo, err := os.Stat(abs)
  if err != nil {
    log.Println("error accessing", abs)
		http.NotFound(w, r)
    return
  }

  if fileInfo.IsDir() {
    serveDir(w, abs, urlPath)
    log.Println("served directory", abs)
    return
  }

  if filepath.Ext(abs) == ".md" {
    serveMarkdown(w, abs)
    log.Println("served markdown", abs)
    return
  }

  http.ServeFile(w, r, abs)
  log.Println("served file", abs)
}
