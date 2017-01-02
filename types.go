package main

import (
	"io"
)

// PassThru wraps an existing io.Reader.
//
// It simply forwards the Read() call, while displaying
// the results from individual calls to it.
type PassThru struct {
	Worker *Worker
	io.Reader
	length int64
	total  int64 // Total # of bytes transferred
	stop   chan bool
	subs   []chan Status
}

type Server struct {
	Configuration Configuration
	workers       map[string]*Worker
	canEnque      chan string
	workerQueue   chan string
}

type Task struct {
	Url      string `json:"url"`
	Accept   string `json:"accept,omitempty"`
	Location string `json:"location,omitempty"`
}

type Configuration struct {
	Interface           string
	DownloadDir         string
	ConcurrentDownloads int
	MaxWorkers          int
}

type Status struct {
	WorkerId string `json:"workerid"`
	Length   int64  `json:"length"`
	Total    int64  `json:"total"`
}

type Worker struct {
	Id       string `json:"id"`
	Url      string `json:"url"`
	Location string `json:"location"`
	Filename string `json:"filename"`
	Status   Status `json:"status"`
	stop     chan bool
	pt       *PassThru
	filepath string
}
