package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yanjiulab/lopa/internal/measurement"
)

func init() {
	passiveCmd := &cobra.Command{
		Use:   "passive [interface]",
		Short: "Passive interface counter measurement",
		Long:  "Observe interface bytes/packets in and out without sending probes. Target is the interface name (e.g. eth0).",
		Args:  cobra.ExactArgs(1),
		RunE:  runPassive,
	}

	passiveCmd.Flags().String("mode", string(measurement.ModeDuration), "mode: duration|continuous")
	passiveCmd.Flags().Duration("duration", 60*time.Second, "duration for duration mode; window length in seconds for continuous")
	passiveCmd.Flags().Duration("interval", 10*time.Second, "sampling interval")

	rootCmd.AddCommand(passiveCmd)
}

func runPassive(cmd *cobra.Command, args []string) error {
	iface := args[0]

	modeStr, _ := cmd.Flags().GetString("mode")
	duration, _ := cmd.Flags().GetDuration("duration")
	interval, _ := cmd.Flags().GetDuration("interval")

	mode := measurement.Mode(modeStr)
	if mode != measurement.ModeDuration && mode != measurement.ModeContinuous {
		mode = measurement.ModeDuration
	}

	params := measurement.TaskParams{
		Type:     "passive",
		Target:   iface,
		Interval: interval,
		Duration: duration,
		Mode:     mode,
	}

	base := strings.TrimRight(DaemonAddr(), "/")
	client := &http.Client{Timeout: 10 * time.Second}

	createURL := base + "/api/v1/tasks/passive"
	body, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", createURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to contact daemon at %s: %w", createURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("daemon returned status %s when creating passive task", resp.Status)
	}

	var createResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return fmt.Errorf("failed to decode daemon response: %w", err)
	}
	if createResp.ID == "" {
		return fmt.Errorf("daemon returned empty task id")
	}

	id := measurement.TaskID(createResp.ID)
	fmt.Printf("started passive task %s on interface %s (mode=%s) via daemon %s\n", id, iface, mode, base)

	switch mode {
	case measurement.ModeDuration:
		for {
			getURL := fmt.Sprintf("%s/api/v1/tasks/%s", base, id)
			resp, err := client.Get(getURL)
			if err != nil {
				return fmt.Errorf("failed to query task %s: %w", id, err)
			}
			if resp.StatusCode == http.StatusNotFound {
				resp.Body.Close()
				return fmt.Errorf("task not found: %s", id)
			}
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				return fmt.Errorf("daemon returned status %s when querying task", resp.Status)
			}
			var res measurement.Result
			if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
				resp.Body.Close()
				return fmt.Errorf("failed to decode task result: %w", err)
			}
			resp.Body.Close()
			if res.Status == "finished" || res.Status == "failed" || res.Status == "stopped" {
				printResult(res)
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	case measurement.ModeContinuous:
		for {
			getURL := fmt.Sprintf("%s/api/v1/tasks/%s", base, id)
			resp, err := client.Get(getURL)
			if err != nil {
				return fmt.Errorf("failed to query task %s: %w", id, err)
			}
			if resp.StatusCode == http.StatusNotFound {
				resp.Body.Close()
				return fmt.Errorf("task not found: %s", id)
			}
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				return fmt.Errorf("daemon returned status %s when querying task", resp.Status)
			}
			var res measurement.Result
			if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
				resp.Body.Close()
				return fmt.Errorf("failed to decode task result: %w", err)
			}
			resp.Body.Close()
			if res.Window != nil {
				fmt.Printf("task=%s iface=%s window=%ds bytes_in=%d bytes_out=%d pkts_in=%d pkts_out=%d\n",
					res.TaskID, res.Target, res.Window.WindowSeconds,
					res.Window.Stats.BytesIn, res.Window.Stats.BytesOut,
					res.Window.Stats.PacketsIn, res.Window.Stats.PacketsOut)
			}
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}
