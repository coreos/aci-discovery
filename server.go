package main

import (
	"log"
	"net/http"
	"text/template"
)

var t = template.Must(template.New("name").Parse(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="ac-discovery" content="{{.PrefixMatch}} {{.ACITemplateURL}}">
    <meta name="ac-discovery-pubkeys" content="{{.PrefixMatch}} {{.PubkeysURL}}">
  <head>
<html>
`))

func discover(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Query().Get("ac-discovery") != "1" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	data := struct {
		PrefixMatch    string
		ACITemplateURL string
		PubkeysURL     string
	}{
		"example.com/hello",
		"http://example.com/images/{name}-{version}-{os}-{arch}.{ext}",
		"http://example.com/pubkeys.gpg",
	}
	err := t.Execute(w, data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/", discover)
	http.Handle("/pubkeys.gpg", http.StripPrefix("/", http.FileServer(http.Dir("/opt/images"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("/opt/images"))))
	log.Fatal(http.ListenAndServe(":80", nil))
}
