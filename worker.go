package main

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/speps/go-hashids"
)

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
	s := Status{WorkerId: pt.Worker.Id, Total: pt.total, Length: pt.length}
	for _, c := range pt.subs {
		c <- Status{WorkerId: pt.Worker.Id, Total: pt.total, Length: pt.length}
	}
	pt.Worker.Status = s
	return n, err
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
	w.Id = GetID()
	w.Status = Status{WorkerId: w.Id}
	w.pt = &PassThru{Worker: &w}
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
	w.pt.Reader = response.Body
	w.pt.length = response.ContentLength
	w.pt.stop = w.stop
	io.Copy(file, w.pt)
	done <- w.Filename
	return nil
}

func (w *Worker) IsActive() bool {
	return w.pt.length != 0
}

func (w *Worker) AddListener(c chan Status) {
	w.pt.subs = append(w.pt.subs, c)
}

func (w *Worker) Stop() {
	w.stop <- true
}

func GetID() string {
	hd := hashids.NewData()
	hd.Salt = "overload"
	hd.MinLength = 10
	h := hashids.NewWithData(hd)
	d := []int64{0}
	d[0] = time.Now().UnixNano()
	e, _ := h.EncodeInt64(d)
	return e
}
