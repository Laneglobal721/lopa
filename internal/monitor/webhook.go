package monitor

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/yanjiulab/lopa/internal/logger"
)

// Notify sends the monitor event to the task's webhook URL (async, non-blocking).
func Notify(webhookURL string, evt Event) {
	if webhookURL == "" {
		return
	}
	go func() {
		body, err := json.Marshal(evt)
		if err != nil {
			logger.S().Warnw("monitor event payload marshal failed", "err", err)
			return
		}
		req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
		if err != nil {
			logger.S().Warnw("monitor webhook request build failed", "err", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			logger.S().Warnw("monitor webhook request failed", "url", webhookURL, "err", err)
			return
		}
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			logger.S().Warnw("monitor webhook returned non-2xx", "url", webhookURL, "status", resp.StatusCode)
			return
		}
		logger.S().Infow("monitor webhook sent", "url", webhookURL, "task_id", evt.TaskID, "type", evt.Type, "change", evt.Change)
	}()
}
