package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"text/template"
)

func main() {
	// don't need timestamps running under systemd
	log.SetFlags(0)

	fs := flag.NewFlagSet("aci-server", flag.ExitOnError)
	domain := fs.String("domain", "", "user-facing domain routable to this serve")
	listen := fs.String("listen", "0.0.0.0:80", "IP & port to bind")
	imagesURL := fs.String("images", "file:///opt/aci/images", "")
	keysURL := fs.String("keys", "file:///opt/aci/pubkeys.gpg", "")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalf("Failed parsing flags: %v", err)
	}

	if *domain == "" {
		log.Fatalf("--domain must be set")
	}

	ep := url.URL{Scheme: "http", Host: *domain}

	ir, err := NewImageRepo(ep, *imagesURL)
	if err != nil {
		log.Fatalf("Unable to create image repo: %v", err)
	}

	kr, err := NewKeyRepo(ep, *keysURL)
	if err != nil {
		log.Fatalf("Unable to create key repo: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleDiscoverFunc(*domain, ir, kr))

	ir.Register(mux)
	kr.Register(mux)

	srv := http.Server{
		Addr:    *listen,
		Handler: mux,
	}

	log.Printf("Serving ACI discovery on %s...", *listen)
	log.Fatal(srv.ListenAndServe())
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

func handleDiscoverFunc(domain string, ir ImageRepo, kr KeyRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ac-discovery") != "1" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		name := path.Base(r.URL.Path)
		meta := entry{
			PrefixMatch:    fmt.Sprintf("%s/%s", domain, name),
			ACITemplateURL: ir.URL(name),
			PubkeysURL:     kr.URL(),
		}

		if err := tmpl.Execute(w, meta); err != nil {
			log.Printf("Failed serving metadata: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

type ImageRepo interface {
	Register(*http.ServeMux)
	URL(string) string
}

func NewImageRepo(ep url.URL, u string) (ImageRepo, error) {
	iu, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	if iu.Scheme != "file" {
		return nil, errors.New("unsupported scheme, must be file://")
	}

	r := &localImageRepo{
		ep:  ep,
		dir: iu.Path,
	}

	return r, nil
}

type localImageRepo struct {
	ep  url.URL
	dir string
}

func (r *localImageRepo) URL(name string) string {
	//NOTE(bcwaldon): not using path.Join here since URL.String()
	// url-encodes the curly brackets
	return fmt.Sprintf("%s/repo/%s", r.ep.String(), fmt.Sprintf("{os}/{arch}/%s-{version}.{ext}", name))
}

func (r *localImageRepo) Register(mux *http.ServeMux) {
	mux.Handle("/repo/", http.StripPrefix("/repo/", http.FileServer(http.Dir(r.dir))))
}

type KeyRepo interface {
	Register(*http.ServeMux)
	URL() string
}

func NewKeyRepo(ep url.URL, u string) (KeyRepo, error) {
	ku, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	if ku.Scheme != "file" {
		return nil, errors.New("unsupported scheme, must be file://")
	}

	data, err := ioutil.ReadFile(ku.Path)
	if err != nil {
		return nil, err
	}

	r := &localKeyRepo{
		ep:   ep,
		data: data,
	}

	return r, nil
}

type localKeyRepo struct {
	ep   url.URL
	data []byte
}

func (kr *localKeyRepo) Register(mux *http.ServeMux) {
	mux.HandleFunc("/pubkeys.gpg", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(kr.data)
	})
}

func (kr *localKeyRepo) URL() string {
	ep := kr.ep
	ep.Path = path.Join(ep.Path, "pubkeys.gpg")
	return ep.String()
}
