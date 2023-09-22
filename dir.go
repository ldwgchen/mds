package main

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"strings"
)

func serveDir(w http.ResponseWriter, abs string, urlPath string) {
	if !strings.HasSuffix(urlPath, "/") {
		urlPath += "/"
	}
	w.Header().Add("Content-Type", "text/html")

	var b bytes.Buffer
	dirEntries, err := os.ReadDir(abs)
	if err != nil {
		log.Println("error reading dir", abs)
		return
	}

	b.WriteString("<div class=\"entries\">")
	for _, dirEntry := range dirEntries {
		var optional string
		if dirEntry.IsDir() {
			optional = "/"
		}
		name := dirEntry.Name() + optional
		strings.ReplaceAll(name, "\n", "")
		b.WriteString("<a href=\"" + urlPath + name + "\">" + name + "</a><br>")
	}
	b.WriteString("</div>")

	w.Write(c.header)
	b.WriteTo(w)
	w.Write(c.footer)
}
