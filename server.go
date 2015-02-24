package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"text/template"
)

func main() {
	fs := flag.NewFlagSet("aci-server", flag.ExitOnError)
	domain := fs.String("domain", "", "")
	listen := fs.String("listen", ":80", "")
	dir := fs.String("image-dir", "/opt/images", "")
	images := fs.String("images", "", "comma-delimited list of images to serve from --image-dir")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalf("Failed parsing flags: %v", err)
	}

	if *domain == "" {
		log.Fatalf("--domain must be set")
	}

	spImages := strings.Split(*images, ",")
	if len(spImages) == 0 {
		log.Fatalf("--images must be set")
	}

	log.Printf("Serving images: %v", spImages)

	http.HandleFunc("/", handleDiscoverFunc(*domain, spImages))
	http.Handle("/pubkeys.gpg", http.StripPrefix("/", http.FileServer(http.Dir(*dir))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(*dir))))
	log.Fatal(http.ListenAndServe(*listen, nil))
}

var tmpl = template.Must(template.New("name").Parse(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="ac-discovery" content="{{.PrefixMatch}} {{.ACITemplateURL}}">
    <meta name="ac-discovery-pubkeys" content="{{.PrefixMatch}} {{.PubkeysURL}}">
  <head>
<html>
`))

type entry struct {
	PrefixMatch    string
	ACITemplateURL string
	PubkeysURL     string
}

func handleDiscoverFunc(domain string, images []string) http.HandlerFunc {
	im := make(map[string]struct{}, len(images))
	for _, image := range images {
		im[image] = struct{}{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ac-discovery") != "1" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		lookup := path.Base(r.URL.Path)
		if _, ok := im[lookup]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		data := entry{
			PrefixMatch:    fmt.Sprintf("%s/%s", domain, lookup),
			ACITemplateURL: fmt.Sprintf("http://%s/images/%s-{version}-{os}-{arch}.{ext}", domain, lookup),
			PubkeysURL:     fmt.Sprintf("http://%s/pubkeys.gpg", domain),
		}

		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("Failed serving discovery resource: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
