package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"golang.org/x/net/websocket"
)

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
	g.GET("/status", s.StreamStatus)

	e.Static("/web", "web")
	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusTemporaryRedirect, "/web")
	})

	s.canEnque = make(chan string, s.Configuration.ConcurrentDownloads)
	for i := 0; i < s.Configuration.ConcurrentDownloads-1; i++ { // Initialize the channel with empty finished downloads
		s.canEnque <- ""
	}
	s.workerQueue = make(chan string, s.Configuration.MaxWorkers)
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
	e.Logger.Fatal(e.Start(s.Configuration.Interface))
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
		t.Location = s.Configuration.DownloadDir
	}

	urls, err := getURLs(u, accept)
	if err != nil {
		return c.NoContent(http.StatusInternalServerError)
	}
	go func() {
		for _, url := range urls {
			w, err := NewWorker(url, t.Location)
			if err != nil {
				c.Logger().Errorf("Failed to create worker! %s", err.Error())
				continue
			}
			s.workers[w.Id] = w
			s.workerQueue <- w.Id
		}
	}()
	return c.NoContent(http.StatusAccepted)
}

func (s *Server) GetWorkers(c echo.Context) error {
	var ws []*Worker
	for _, w := range s.workers {
		ws = append(ws, w)
	}
	return c.JSON(http.StatusOK, ws)
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
	stat := make(chan Status)
	for _, w := range s.workers {
		w.AddListener(stat)
	}
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			status := <-stat
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
