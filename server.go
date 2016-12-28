package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/speps/go-hashids"
	"golang.org/x/net/websocket"
)

type Server struct {
	DownloadDir         string
	Interface           string
	workers             map[string]*Worker
	ConcurrentDownloads int
	canEnque            chan string
	workerQueue         chan string
	MaxWorkers          int
}

type Task struct {
	Url      string `json:"url"`
	Accept   string `json:"accept,omitempty"`
	Location string `json:"location,omitempty"`
}

func (s *Server) Run() {
	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Recover())

	s.workers = make(map[string]*Worker)

	g := e.Group("/api")
	g.POST("/tasks", s.AddTask)
	g.GET("/workers", s.GetWorkers)
	g.DELETE("/workers/:id", s.StopWorker)
	g.GET("/workers/:id/active", s.GetActive)
	g.GET("/workers/:id/ws", s.StreamStatus)

	e.Static("/web", "web")
	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, "/web")
	})

	s.canEnque = make(chan string, s.ConcurrentDownloads)
	//for i := 0; i < s.ConcurrentDownloads-1; i++ { // Initialize the channel with empty finished downloads
//		s.canEnque <- ""
//	}
	s.workerQueue = make(chan string, s.MaxWorkers)
	go func() {
		for {
			id := <-s.workerQueue
			e.Logger.Infof("Starting worker %s", id)
			go s.workers[id].Download(s.canEnque)
			done := <-s.canEnque
			if done != "" {
				e.Logger.Infof("Download Finished: %s", done)
			}
		}
	}()
	// Start server
	e.Logger.Fatal(e.Start(s.Interface))
}

func (s *Server) AddTask(c echo.Context) error {
	req := c.Request()
	decoder := json.NewDecoder(req.Body)
	var t Task
	err := decoder.Decode(&t)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	defer req.Body.Close()

	var accept *regexp.Regexp
	if t.Accept == "" {
		t.Accept = ".*"
	}
	accept, err = regexp.Compile(t.Accept)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	u, err := url.Parse(t.Url)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	if t.Location == "" {
		t.Location = s.DownloadDir
	}

	urls, err := getURLs(u, accept)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	go func() {
		for _, url := range urls {
			id := s.GetID()
			w, err := NewWorker(url, t.Location)
			if err != nil {
				c.Logger().Errorf("Failed to create worker! %s", err.Error())
				continue
			}
			s.workers[id] = w
			s.workerQueue <- id
		}
	}()
	return c.NoContent(http.StatusAccepted)
}

func (s *Server) GetWorkers(c echo.Context) error {
	workers := []struct {
		Id       string
		Url      string
		Filename string
		Status   Status
	}{}
	for id, w := range s.workers {
		workers = append(workers, struct {
			Id       string
			Url      string
			Filename string
			Status   Status
		}{Id: id, Url: w.Url, Status: w.Status(), Filename: w.Filename})
	}
	return c.JSON(http.StatusOK, workers)
}

func (s *Server) StopWorker(c echo.Context) error {
	worker := s.workers[c.Param("id")]
	if worker == nil {
		return c.NoContent(http.StatusNotFound)
	}
	worker.Stop()
	return c.NoContent(http.StatusOK)
}

func (s *Server) GetActive(c echo.Context) error {
	worker := s.workers[c.Param("id")]
	if worker.IsActive() {
		return c.NoContent(http.StatusOK)
	} else {
		return c.NoContent(http.StatusNotAcceptable)
	}
}

func (s *Server) StreamStatus(c echo.Context) error {
	worker := s.workers[c.Param("id")]
	if worker == nil {
		return c.NoContent(http.StatusNotFound)
	}
	if !worker.IsActive() {
		return c.NoContent(http.StatusNotFound)
	}
	lid := time.Now().UnixNano()
	sub := worker.AddListener(lid)
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		defer worker.RemoveListener(lid)
		for {
			status := <-sub
			b, err := json.Marshal(status)
			if err != nil {
				c.Logger().Error(err)
			}
			// Write
			err = websocket.Message.Send(ws, string(b))
			if err != nil {
				break
			}
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func (s *Server) GetID() string {
	hd := hashids.NewData()
	hd.Salt = "overload"
	hd.MinLength = 10
	h := hashids.NewWithData(hd)
	d := []int64{0}
	d[0] = time.Now().UnixNano()
	e, _ := h.EncodeInt64(d)
	return e
}
