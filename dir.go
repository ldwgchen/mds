package main

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"strings"
)

func serveDir(w http.ResponseWriter, r *http.Request, abs string) {
	urlPath := r.URL.Path
	if !strings.HasSuffix(urlPath, "/") {
		http.Redirect(w, r, urlPath+"/", 301)
		return
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
		if strings.HasPrefix(dirEntry.Name(), ".") {
			continue
		}
		var optional string
		if dirEntry.IsDir() {
			optional = "/"
		}
		name := dirEntry.Name() + optional
		strings.ReplaceAll(name, "\n", "")
		b.WriteString("<a href=\"" + urlPath + name + "\">" + name + "</a><br>")
	}
	b.WriteString("</div>\n")

	w.Write(c.header)
	w.Write([]byte("<body>\n"))
	b.WriteTo(w)
	w.Write(c.footer)
	w.Write([]byte("</body>\n"))
}
