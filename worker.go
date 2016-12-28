package main

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

// PassThru wraps an existing io.Reader.
//
// It simply forwards the Read() call, while displaying
// the results from individual calls to it.
type PassThru struct {
	io.Reader
	length int64
	total  int64 // Total # of bytes transferred
	stop   chan bool
	subs   map[int64]chan Status
}

// Read 'overrides' the underlying io.Reader's Read method.
// This is the one that will be called by io.Copy(). We simply
// use it to keep track of byte counts and then forward the call.
func (pt *PassThru) Read(p []byte) (int, error) {
	select {
	case <-pt.stop:
		return 0, errors.New("EOF")
	default:
		break
	}
	n, err := pt.Reader.Read(p)
	pt.total += int64(n)
	for _, c := range pt.subs {
		c <- Status{Total: pt.total, Length: pt.length}
	}
	return n, err
}

type Status struct {
	Length int64
	Total  int64
}

type Worker struct {
	Url      string
	Location string
	stop     chan bool
	pt       *PassThru
	Filename string
	filepath string
}

func NewWorker(rawurl, location string) (*Worker, error) {
	w := Worker{Url: rawurl, Location: location}
	w.stop = make(chan bool)
	fileURL, err := url.Parse(w.Url)
	if err != nil {
		return nil, err
	}

	upath := fileURL.Path

	segments := strings.Split(upath, "/")
	name := segments[len(segments)-1]
	w.Filename = name
	w.filepath = path.Join(w.Location, name)
	return &w, nil
}

func (w *Worker) Download(done chan string) error {
	file, err := os.Create(w.filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	client := &http.Client{}
	response, err := client.Get(w.Url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	w.pt = &PassThru{Reader: response.Body, length: response.ContentLength, stop: w.stop, subs: make(map[int64]chan Status)}
	io.Copy(file, w.pt)
	done <- w.Filename
	return nil
}

func (w *Worker) Status() Status {
	if w.pt != nil {
		return Status{Length: w.pt.length, Total: w.pt.total}
	} else {
		return Status{Length: 0, Total: 0}
	}
}

func (w *Worker) IsActive() bool {
	return w.pt != nil
}

func (w *Worker) AddListener(lid int64) chan Status {
	c := make(chan Status, 1024)
	w.pt.subs[lid] = c
	return c
}

func (w *Worker) RemoveListener(lid int64) {
	delete(w.pt.subs, lid)
}

func (w *Worker) Stop() {
	w.stop <- true
}
