package server

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/yanjiulab/lopa/internal/config"
	"github.com/yanjiulab/lopa/internal/logger"
	"github.com/yanjiulab/lopa/internal/measurement"
	"github.com/yanjiulab/lopa/internal/monitor"
)

// Start starts the HTTP server in a separate goroutine.
func Start() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	registerRoutes(e)

	go func() {
		addr := config.Global().HTTP.Addr
		if addr == "" {
			addr = ":8080"
		}
		logger.S().Infow("starting http server", "addr", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			logger.S().Errorw("http server error", "err", err)
		}
	}()

	return e
}

func registerRoutes(e *echo.Echo) {
	g := e.Group("/api/v1")

	g.POST("/tasks/ping", createPingTaskHandler)
	g.POST("/tasks/tcp", createTcpTaskHandler)
	g.POST("/tasks/udp", createUdpTaskHandler)
	g.POST("/tasks/twamp", createTwampTaskHandler)
	g.POST("/tasks/passive", createPassiveTaskHandler)
	g.GET("/tasks", listTasksHandler)
	g.GET("/tasks/:id", getTaskHandler)
	g.POST("/tasks/:id/stop", stopTaskHandler)
	g.DELETE("/tasks/:id", deleteTaskHandler)

	// Monitors (netlink-based: interface, IP changes)
	g.POST("/monitors", createMonitorHandler)
	g.GET("/monitors", listMonitorsHandler)
	g.GET("/monitors/:id", getMonitorHandler)
	g.PATCH("/monitors/:id", updateMonitorHandler)
	g.DELETE("/monitors/:id", deleteMonitorHandler)
	g.GET("/monitors/:id/events", getMonitorEventsHandler)
}

type createPingRequest struct {
	measurement.TaskParams `json:",inline"`
}

type createTaskResponse struct {
	ID string `json:"id"`
}

func createPingTaskHandler(c echo.Context) error {
	var req measurement.TaskParams
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	req.Type = "ping"

	engine := measurement.DefaultEngine()
	id, err := engine.CreatePingTask(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, createTaskResponse{ID: string(id)})
}

func createTcpTaskHandler(c echo.Context) error {
	var req measurement.TaskParams
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	req.Type = "tcp"

	engine := measurement.DefaultEngine()
	id, err := engine.CreateTcpTask(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, createTaskResponse{ID: string(id)})
}

func createUdpTaskHandler(c echo.Context) error {
	var req measurement.TaskParams
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	req.Type = "udp"

	engine := measurement.DefaultEngine()
	id, err := engine.CreateUdpTask(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, createTaskResponse{ID: string(id)})
}

func createTwampTaskHandler(c echo.Context) error {
	var req measurement.TaskParams
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	req.Type = "twamp"

	engine := measurement.DefaultEngine()
	id, err := engine.CreateTwampTask(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, createTaskResponse{ID: string(id)})
}

func createPassiveTaskHandler(c echo.Context) error {
	var req measurement.TaskParams
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	req.Type = "passive"

	engine := measurement.DefaultEngine()
	id, err := engine.CreatePassiveTask(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, createTaskResponse{ID: string(id)})
}

func listTasksHandler(c echo.Context) error {
	engine := measurement.DefaultEngine()
	results := engine.ListResults()
	return c.JSON(http.StatusOK, results)
}

func getTaskHandler(c echo.Context) error {
	id := measurement.TaskID(c.Param("id"))
	engine := measurement.DefaultEngine()
	res, ok := engine.GetResult(id)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "task not found"})
	}
	return c.JSON(http.StatusOK, res)
}

func stopTaskHandler(c echo.Context) error {
	id := measurement.TaskID(c.Param("id"))
	engine := measurement.DefaultEngine()
	engine.StopTask(id)
	return c.NoContent(http.StatusAccepted)
}

func deleteTaskHandler(c echo.Context) error {
	id := measurement.TaskID(c.Param("id"))
	engine := measurement.DefaultEngine()
	ok := engine.DeleteTask(id)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "task not found"})
	}
	return c.NoContent(http.StatusNoContent)
}

// Monitor handlers
func createMonitorHandler(c echo.Context) error {
	var req struct {
		Type       string        `json:"type"`
		Filter     monitor.Filter `json:"filter"`
		WebhookURL string        `json:"webhook_url"`
		Enabled    *bool         `json:"enabled"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	t := &monitor.Task{
		Type:       monitor.TaskType(req.Type),
		Filter:     req.Filter,
		WebhookURL: req.WebhookURL,
	}
	if req.Type == "" {
		t.Type = monitor.TypeInterface
	}
	if req.Enabled != nil {
		t.Enabled = *req.Enabled
	} else {
		t.Enabled = true
	}
	st := monitor.DefaultStore()
	id := st.AddTask(t)
	return c.JSON(http.StatusCreated, map[string]string{"id": id})
}

func listMonitorsHandler(c echo.Context) error {
	st := monitor.DefaultStore()
	tasks := st.ListTasks()
	return c.JSON(http.StatusOK, tasks)
}

func getMonitorHandler(c echo.Context) error {
	id := c.Param("id")
	st := monitor.DefaultStore()
	t, ok := st.GetTask(id)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}
	return c.JSON(http.StatusOK, t)
}

func updateMonitorHandler(c echo.Context) error {
	id := c.Param("id")
	var req struct {
		WebhookURL *string        `json:"webhook_url"`
		Enabled    *bool          `json:"enabled"`
		Filter     *monitor.Filter `json:"filter"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	st := monitor.DefaultStore()
	ok := st.UpdateTask(id, func(t *monitor.Task) {
		if req.WebhookURL != nil {
			t.WebhookURL = *req.WebhookURL
		}
		if req.Enabled != nil {
			t.Enabled = *req.Enabled
		}
		if req.Filter != nil {
			t.Filter = *req.Filter
		}
	})
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}
	return c.NoContent(http.StatusNoContent)
}

func deleteMonitorHandler(c echo.Context) error {
	id := c.Param("id")
	st := monitor.DefaultStore()
	ok := st.DeleteTask(id)
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}
	return c.NoContent(http.StatusNoContent)
}

func getMonitorEventsHandler(c echo.Context) error {
	id := c.Param("id")
	lastN := 0
	if n := c.QueryParam("last"); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v > 0 {
			lastN = v
		}
	}
	st := monitor.DefaultStore()
	if _, ok := st.GetTask(id); !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "monitor not found"})
	}
	events := st.GetEvents(id, lastN)
	return c.JSON(http.StatusOK, events)
}

