package main

import (
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/BurntSushi/toml"
	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Configuration struct {
	Interface           string
	DownloadDir         string
	ConcurrentDownloads int
	MaxWorkers          int
}

func main() {
	confLocation := "overload.toml"
	if len(os.Args) > 1 {
		confLocation = os.Args[2]
	}
	var c Configuration
	toml.DecodeFile(confLocation, &c)
	s := Server{
		Interface:           c.Interface,
		DownloadDir:         c.DownloadDir,
		ConcurrentDownloads: c.ConcurrentDownloads,
		MaxWorkers:          c.MaxWorkers,
	}
	s.Run()
}

func getURLs(dir *url.URL, accept *regexp.Regexp) ([]string, error) {
	resp, err := http.Get(dir.String())
	if err != nil {
		return nil, err
	}
	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	// define a matcher
	matcher := func(n *html.Node) bool {
		return n.DataAtom == atom.A && accept.MatchString(scrape.Attr(n, "href"))
	}
	// grab all articles and print them
	tokens := scrape.FindAll(root, matcher)
	var links []string
	for _, link := range tokens {
		ref := scrape.Attr(link, "href")
		u, err := url.Parse(ref)
		if err != nil {
			continue //Skip invalid urls
		}
		ref = dir.ResolveReference(u).String()
		if !contains(links, ref) {
			links = append(links, ref)
		}
	}
	return links, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
