package main

import (
	"net/http"
	"net/url"
	"regexp"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func main() {
	s := Server{
		Interface:           ":8000",
		DownloadDir:         "/tmp/downloads",
		ConcurrentDownloads: 3,
		MaxWorkers:          1024,
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
