/*
* The MIT License (MIT)
*
* Copyright (c) 2017  aerth <aerth@riseup.net>
*
* Permission is hereby granted, free of charge, to any person obtaining a copy
* of this software and associated documentation files (the "Software"), to deal
* in the Software without restriction, including without limitation the rights
* to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
* copies of the Software, and to permit persons to whom the Software is
* furnished to do so, subject to the following conditions:
*
* The above copyright notice and this permission notice shall be included in all
* copies or substantial portions of the Software.
*
* THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
* IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
* FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
* AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
* LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
* OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
* SOFTWARE.
 */

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
  "bytes"
  "github.com/yuin/goldmark"
  "github.com/yuin/goldmark/extension"
  "github.com/yuin/goldmark/parser"
  "github.com/yuin/goldmark/renderer/html"
  "go.abhg.dev/goldmark/wikilink"
  "go.abhg.dev/goldmark/frontmatter"
)

// flags
var (
	addr          = flag.String("http", "127.0.0.1:5555", "address to listen on format 'address:port',\n\tif address is omitted will listen on all interfaces")
	logfile       = flag.String("log", os.Stderr.Name(), "redirect logs to this file")
	header        = flag.String("header", "", "html header for markdown requests")
	footer        = flag.String("footer", "", "html footer for markdown requests")
	fbheader     = flag.String("fbheader", "", "html header for the file browser page")
	favicon     = flag.String("favicon", "", "favicon filename")
  requestid = 0
)

// log to file
var logger = log.New(os.Stderr, "[mdd] ", log.LstdFlags)

const version = "1"
const sig = "[mdd v" + version + "] modified from: https://github.com/aerth/markdownd"
const serverheader = "mdd/" + version
const usage = `
USAGE

mdd [flags] [directory]

EXAMPLES

Serve current directory on 127.0.0.1:5555:
	mdd .

Serve current directory on all interfaces, port 8080, log to stderr:
	mdd -log /dev/stderr -http 0.0.0.0:8080 .

Serve 'docs' directory on port 8081, log to 'md.log':
	mdd -log md.log -http :8081 docs

Serve 'docs' with header and footers for markdown files. Disable Logs:
	mdd -log none -header bar.html -footer foo.html

Serve 'docs' only on localhost:
	mdd -http 127.0.0.1:8080 docs
FLAGS`

func init() {
	flag.Usage = func() {
		fmt.Println(usage)
		flag.PrintDefaults()
	}
}

type mddHandler struct {
	Root                      http.FileSystem // directory to serve
	RootString                string          // keep directory name for comparing prefix
	header, footer, fbheader []byte
}

type faviconHandler struct {
	FaviconPath               string
}

type customResolver struct{}

func (customResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
  var _html = []byte(".html")
  var _md = []byte(".md")
	var _hash = []byte{'#'}
	dest := make([]byte, len(n.Target)+len(_html)+len(_hash)+len(n.Fragment))
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

func main() {
	fmt.Println(sig)
	flag.Parse()
	serve(flag.Args())
}

func serve(args []string) {

	if len(args) != 1 {
		flag.Usage()
		os.Exit(111)
		return
	}

	// get absolute path of flag.Arg(0)
	dir := flag.Arg(0)
	dir = prepareDirectory(dir)

	mddhandler := &mddHandler{
		Root:       http.Dir(dir),
		RootString: dir,
	}

	h := http.DefaultServeMux
	h.Handle("/", mddhandler)

	// print absolute directory we are serving
	println("serving filesystem:", dir)

	// take care of opening log file
	openLogFile()
	println("logging to:", *logfile)

	if *header != "" {
		println("html header:", *header)
		b, err := os.ReadFile(*header)
		if err != nil {
			println(err.Error())
			os.Exit(111)
		}
		mddhandler.header = b
	} else {
		mddhandler.header = []byte("<!DOCTYPE html>\n")
	}

	if *footer != "" {
		println("html footer:", *footer)
		b, err := os.ReadFile(*footer)
		if err != nil {
			println(err.Error())
			os.Exit(111)
		}
		mddhandler.footer = b
	}

	if *fbheader == "" {
		mddhandler.fbheader = []byte("<!DOCTYPE html>\n")
	} else {
    println("fbheader:", *fbheader)
    b, err := os.ReadFile(*fbheader)
    if err != nil {
      println(err.Error())
      os.Exit(111)
    }
    mddhandler.fbheader = b
	}

	if *favicon != "" {
		println("favicon:", *favicon)
		_, err := os.ReadFile(*favicon)
		if err != nil {
			println(err.Error())
			os.Exit(111)
		}
		faviconhandler := &faviconHandler{
			FaviconPath: *favicon,
		}
		h.Handle("/favicon.ico", faviconhandler)
	}

	// create a http server
	server := &http.Server{
		Addr:              *addr,
		Handler:           h,
		ErrorLog:          logger,
		MaxHeaderBytes:    (1 << 10), // 1KB
		ReadTimeout:       (time.Second * 5),
		WriteTimeout:      (time.Second * 5),
		ReadHeaderTimeout: (time.Second * 5),
		IdleTimeout:       (time.Second * 5),
	}

	// disable keepalives
	server.SetKeepAlivesEnabled(false)

	// trick to show listening port
	go func() { <-time.After(time.Second); logger.Println("listening:", *addr) }()

	// start serving
	err := server.ListenAndServe()

	// print usage info, probably started wrong or port is occupied
	flag.Usage()

	// always non-nil
	logger.Println(err)

	// any exit is an error
	os.Exit(111)
}

func (h mddHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  requestid += 1

	// all we want is GET
	if r.Method != "GET" {
		logger.Println("bad method:", r.RemoteAddr, r.Method, r.URL.Path, r.UserAgent())
		http.NotFound(w, r)
		return
	}

	// deny requests containing '..'
	if strings.Contains(r.URL.Path, "..") {
		logger.Println("bad path:", r.RemoteAddr, r.Method, r.URL.Path, r.UserAgent())
		http.NotFound(w, r)
		return
	}

	// start timing
	t1 := time.Now()

	// Add Server header
	w.Header().Add("Server", serverheader)

	// Prevent page from being displayed in an iframe
	w.Header().Add("X-Frame-Options", "DENY")

	// log how long this takes
	defer func(t func() time.Time) {
		logger.Printf("%06x closed after %s", requestid, t().Sub(t1))
	}(time.Now)

	// abs is not absolute yet
	abs := r.URL.Path[1:] // remove slash prefix

	// still not absolute, prepend root directory to filesrc
	abs = h.RootString + abs

	// get absolute path of requested file (could not exist)
	abs, err := filepath.Abs(abs)
	if err != nil {
		logger.Printf("%06x error resolving absolute path: %s", requestid, err)
		http.NotFound(w, r)
		return
	}

	// log now that we have abs
	logger.Printf("%06x %s %s %s %s %s", requestid, r.RemoteAddr, r.Method, r.URL.Path, "->", abs)

	// .html suffix, but .md exists. choose to serve .md over .html
	if strings.HasSuffix(abs, ".html") {
		trymd := strings.TrimSuffix(abs, ".html") + ".md"
		_, err := os.Open(trymd)
		if err == nil {
			logger.Printf("%06x %s %s %s", requestid, abs, "->", trymd)
			abs = trymd
		}
	}

	// check if exists, or give 404
	_, err = os.Open(abs)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			logger.Printf("%06x %s %s", requestid, "404", abs)
			http.NotFound(w, r)
			return
		}

		// probably something about permissions
		logger.Printf("%06x %s %s %s", requestid, "error opening file:", err, abs)
		http.NotFound(w, r)
		return
	}

	// check if symlink ( to avoid /proc/self/root style attacks )
	if !fileisgood(abs) {
		logger.Printf("%06x error: %q is symlink. serving 404", requestid, abs)
		http.NotFound(w, r)
		return
	}

	if fileisdir(abs) {
		if !strings.HasSuffix(r.URL.Path, "/") {
			logger.Printf("%06x error attempt to access directory without slash suffix: %q", requestid, abs)
			http.NotFound(w, r)
			return
		}
		w.Write(h.fbheader)
		logger.Printf("%06x generated file browser page: %q", requestid, abs)
		http.ServeFile(w, r, abs)
		return
	}

	// read bytes (for detecting content type )
	b, err := os.ReadFile(abs)
	if err != nil {
		logger.Printf("%06x error reading file: %q", requestid, abs)
		http.NotFound(w, r)
		return
	}

	// detect content type and encoding
	ct := http.DetectContentType(b)

	// serve raw html if exists
	if strings.HasSuffix(abs, ".html") && strings.HasPrefix(ct, "text/html") && !strings.HasSuffix(r.URL.Path, "/") {
		logger.Printf("%06x serving raw html: %q", requestid, abs)
		w.Header().Add("Content-Type", "text/html")
		w.Write(b)
		return
	}

	// probably markdown
	if strings.HasSuffix(abs, ".md") && strings.HasPrefix(ct, "text/plain") && !strings.HasSuffix(r.URL.Path, "/") {
		if strings.Contains(r.URL.RawQuery, "raw") {
			logger.Printf("%06x raw markdown request: %q", requestid,abs)
			w.Write(b)
			return
		}
		logger.Printf("%06x serving markdown: %q", requestid, abs)

		md := markdown2html(b)
		if md == nil {
			w.WriteHeader(200)
			return
		}
		w.Header().Add("Content-Type", "text/html")
		w.Write(h.header)
		w.Write(md)
		w.Write(h.footer)
		return
	}

	// fallthrough with http.ServeFile
	logger.Printf("%06x serving %s file: %s", requestid, ct, abs)

	http.ServeFile(w, r, abs)
}

func (h faviconHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger.Printf("%s\n", h.FaviconPath)
	http.ServeFile(w, r, h.FaviconPath)
}

func fileisdir(abs string) bool {
	fileInfo, _ := os.Stat(abs)
	return fileInfo.IsDir()
}

// fileisgood returns false if symlink
// comparing absolute vs resolved path is apparently quick and effective
func fileisgood(abs string) bool {

	// sanity check
	if abs == "" {
		return false
	}

	// is absolute really absolute?
	var err error
	if !filepath.IsAbs(abs) {
		abs, err = filepath.Abs(abs)
	}
	if err != nil {
		println(err.Error())
		return false
	}

	// get real path after eval symlinks
	realpath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		println(err.Error())
		return false
	}

	// equality check
	return realpath == abs
}

// prepare root filesystem directory for serving
func prepareDirectory(dir string) string {
	// add slash to dot
	if dir == "." {
		dir += string(os.PathSeparator)
	}

	// become absolute
	var err error
	dir, err = filepath.Abs(dir)
	if err != nil {
		println(err.Error())
		os.Exit(111)
		return err.Error()
	}

	// add trailing slash (for comparing prefix)
	if !strings.HasSuffix(dir, string(os.PathSeparator)) {
		dir += string(os.PathSeparator)
	}

	return dir
}

func markdown2html(in []byte) []byte {
	if len(in) == 0 {
		return nil
	}
  var customResolver wikilink.Resolver = customResolver{}
  md := goldmark.New(
    goldmark.WithExtensions(extension.GFM, extension.DefinitionList,
      extension.Footnote, &wikilink.Extender{Resolver: customResolver},
      &frontmatter.Extender{}),
    goldmark.WithParserOptions(
      parser.WithAutoHeadingID(),
      ),
    goldmark.WithRendererOptions(
      html.WithHardWraps(),
      html.WithXHTML(),
      ),
    )
  var buf_fm bytes.Buffer
  ctx := parser.NewContext()
  md.Convert(in, &buf_fm, parser.WithContext(ctx))
  d := frontmatter.Get(ctx)
  var meta struct {
    Title string
    Keywords []string
  }
  if d != nil {
    if err := d.Decode(&meta); err != nil {
      logger.Println("There's a problem with decoding the frontmatter of some file...")
    }
  }

  var buf bytes.Buffer

  if meta.Title != "" {
    buf.WriteString(`<h1 id="title">` + meta.Title + "</h1>\n")
  }

  if len(meta.Keywords) > 1 {
    buf.WriteString("<p><strong>Keywords:</strong> " + meta.Keywords[0])
    for i := 1; i < len(meta.Keywords); i++ {
      buf.WriteString("<strong>;</strong> " + meta.Keywords[i])
    }
    buf.WriteString("</p>\n")
  } else if len(meta.Keywords) == 1 {
    buf.WriteString("<p><strong>Keywords:</strong> " + meta.Keywords[0] + "</p>\n")
  }

  if err := md.Convert(in, &buf); err != nil {
    panic(err)
  }
	return buf.Bytes()
}

// use logfile flag and set logger Logger
func openLogFile() {
	switch *logfile {
	case os.Stderr.Name(), "stderr":
		// already stderr
		*logfile = os.Stderr.Name()
	case os.Stdout.Name(), "stdout":
		logger.SetOutput(os.Stdout)
		*logfile = os.Stdout.Name()
	case "none", "no", "null", "/dev/null", "nil", "disabled":
		logger.SetOutput(io.Discard)
		*logfile = os.DevNull
	default:
		func() {
			logger.Printf("Opening log file: %q", *logfile)
			f, err := os.OpenFile(*logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
			if err != nil {
				logger.Fatalf("cant open log file: %s", err)
			}
			logger.SetOutput(f)
		}()
	}

}
