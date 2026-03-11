package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/yanjiulab/lopa/internal/monitor"
)

func init() {
	monitorCmd := &cobra.Command{
		Use:   "monitor",
		Short: "Manage netlink monitor tasks (interface, IP change events)",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List monitor tasks",
		RunE:  runMonitorList,
	}
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a monitor task",
		RunE:  runMonitorAdd,
	}
	addCmd.Flags().String("type", string(monitor.TypeInterface), "type: interface|ip|route")
	addCmd.Flags().String("interface", "", "filter by interface name (e.g. eth0)")
	addCmd.Flags().Int("index", 0, "filter by interface index (0=any)")
	addCmd.Flags().String("prefix", "", "filter by IP prefix/CIDR (for type=ip)")
	addCmd.Flags().Int("table", 0, "filter by routing table (0=any, for type=route)")
	addCmd.Flags().String("route-dst", "", "filter by route destination CIDR (e.g. 0.0.0.0/0, for type=route)")
	addCmd.Flags().String("webhook-url", "", "webhook URL to POST on event")
	addCmd.Flags().Bool("enabled", true, "enable the task")

	deleteCmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a monitor task",
		Args:  cobra.ExactArgs(1),
		RunE:  runMonitorDelete,
	}
	eventsCmd := &cobra.Command{
		Use:   "events [id]",
		Short: "Show recent events for a monitor task",
		Args:  cobra.ExactArgs(1),
		RunE:  runMonitorEvents,
	}
	eventsCmd.Flags().Int("last", 20, "number of recent events to show")

	monitorCmd.AddCommand(listCmd, addCmd, deleteCmd, eventsCmd)
	rootCmd.AddCommand(monitorCmd)
}

func runMonitorList(cmd *cobra.Command, args []string) error {
	client := newHTTPClient()
	url := baseURL() + "/api/v1/monitors"
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to list monitors: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon returned %s", resp.Status)
	}
	var tasks []*monitor.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if len(tasks) == 0 {
		fmt.Println("no monitor tasks")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tINTERFACE\tPREFIX\tTABLE\tROUTE_DST\tENABLED\tWEBHOOK")
	for _, t := range tasks {
		enabled := "yes"
		if !t.Enabled {
			enabled = "no"
		}
		webhook := "-"
		if t.WebhookURL != "" {
			webhook = "set"
		}
		tableStr := ""
		if t.Filter.RouteTable != 0 {
			tableStr = fmt.Sprintf("%d", t.Filter.RouteTable)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			t.ID, t.Type, t.Filter.InterfaceName, t.Filter.Prefix, tableStr, t.Filter.RouteDst, enabled, webhook)
	}
	_ = w.Flush()
	return nil
}

func runMonitorAdd(cmd *cobra.Command, args []string) error {
	typ, _ := cmd.Flags().GetString("type")
	iface, _ := cmd.Flags().GetString("interface")
	index, _ := cmd.Flags().GetInt("index")
	prefix, _ := cmd.Flags().GetString("prefix")
	table, _ := cmd.Flags().GetInt("table")
	routeDst, _ := cmd.Flags().GetString("route-dst")
	webhookURL, _ := cmd.Flags().GetString("webhook-url")
	enabled, _ := cmd.Flags().GetBool("enabled")

	taskType := monitor.TaskType(typ)
	if taskType != monitor.TypeInterface && taskType != monitor.TypeIP && taskType != monitor.TypeRoute {
		taskType = monitor.TypeInterface
	}
	filter := map[string]interface{}{
		"interface_name":  iface,
		"interface_index": index,
		"prefix":          prefix,
		"route_table":     table,
		"route_dst":       routeDst,
	}
	body := map[string]interface{}{
		"type":        string(taskType),
		"filter":      filter,
		"webhook_url": webhookURL,
		"enabled":     enabled,
	}
	raw, _ := json.Marshal(body)
	client := newHTTPClient()
	url := baseURL() + "/api/v1/monitors"
	req, err := http.NewRequest("POST", url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create monitor: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("daemon returned %s", resp.Status)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	fmt.Printf("monitor task created: %s\n", out.ID)
	return nil
}

func runMonitorDelete(cmd *cobra.Command, args []string) error {
	id := args[0]
	client := newHTTPClient()
	url := fmt.Sprintf("%s/api/v1/monitors/%s", baseURL(), id)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete monitor: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("monitor not found: %s", id)
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("daemon returned %s", resp.Status)
	}
	fmt.Printf("monitor %s deleted\n", id)
	return nil
}

func runMonitorEvents(cmd *cobra.Command, args []string) error {
	id := args[0]
	last, _ := cmd.Flags().GetInt("last")
	client := newHTTPClient()
	url := fmt.Sprintf("%s/api/v1/monitors/%s/events?last=%d", baseURL(), id, last)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to get events: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("monitor not found: %s", id)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon returned %s", resp.Status)
	}
	var events []monitor.Event
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if len(events) == 0 {
		fmt.Println("no events")
		return nil
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tCHANGE\tAT\tDETAIL")
	for _, e := range events {
		detailStr := formatDetail(e.Detail)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.ID, e.Type, e.Change, e.At.Format("2006-01-02 15:04:05"), detailStr)
	}
	_ = w.Flush()
	return nil
}

func formatDetail(d interface{}) string {
	if d == nil {
		return "-"
	}
	if m, ok := d.(map[string]interface{}); ok {
		var parts []string
		if v, ok := m["name"]; ok {
			parts = append(parts, fmt.Sprintf("name=%v", v))
		}
		if v, ok := m["index"]; ok {
			parts = append(parts, fmt.Sprintf("index=%v", v))
		}
		if v, ok := m["address"]; ok {
			parts = append(parts, fmt.Sprintf("address=%v", v))
		}
		if v, ok := m["interface_name"]; ok {
			parts = append(parts, fmt.Sprintf("iface=%v", v))
		}
		if v, ok := m["table"]; ok {
			parts = append(parts, fmt.Sprintf("table=%v", v))
		}
		if v, ok := m["dst"]; ok {
			parts = append(parts, fmt.Sprintf("dst=%v", v))
		}
		if v, ok := m["gw"]; ok && v != "" {
			parts = append(parts, fmt.Sprintf("gw=%v", v))
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
	}
	return fmt.Sprintf("%v", d)
}
