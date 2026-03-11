package measurement

import (
	"context"
	"sync"
	"time"

	"github.com/yanjiulab/lopa/internal/logger"
	"github.com/yanjiulab/lopa/internal/node"
	"github.com/yanjiulab/lopa/internal/passive"
	"github.com/yanjiulab/lopa/internal/passive/source"
	"github.com/yanjiulab/lopa/internal/protocol"
)

// Engine manages measurement tasks and results in memory.
type Engine struct {
	mu      sync.RWMutex
	tasks   map[TaskID]*Task
	results map[TaskID]*Result
	cancel  map[TaskID]context.CancelFunc
}

var (
	defaultEngine *Engine
	once          sync.Once
)

// DefaultEngine returns the singleton engine instance.
func DefaultEngine() *Engine {
	once.Do(func() {
		defaultEngine = &Engine{
			tasks:   make(map[TaskID]*Task),
			results: make(map[TaskID]*Result),
			cancel:  make(map[TaskID]context.CancelFunc),
		}
	})
	return defaultEngine
}

// CreatePingTask creates and starts a ping task with given parameters.
func (e *Engine) CreatePingTask(params TaskParams) (TaskID, error) {
	id := TaskID(node.NextTaskID())
	now := time.Now()

	t := &Task{
		ID:        id,
		Params:    params,
		NodeID:    node.ID(),
		CreatedAt: now,
	}

	r := &Result{
		TaskID:  id,
		NodeID:  t.NodeID,
		Target:  params.Target,
		Type:    params.Type,
		Mode:    params.Mode,
		Started: now,
		Status:  "running",
		Rounds:  make([]RoundResult, 0),
		Total:   Stats{},
		Window:  nil,
		Error:   "",
	}

	e.mu.Lock()
	e.tasks[id] = t
	e.results[id] = r
	e.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	e.mu.Lock()
	e.cancel[id] = cancel
	e.mu.Unlock()

	go e.runProbe(ctx, t, r)

	return id, nil
}

// CreateTcpTask creates and starts a TCP connect (TCPING) task.
func (e *Engine) CreateTcpTask(params TaskParams) (TaskID, error) {
	id := TaskID(node.NextTaskID())
	now := time.Now()

	t := &Task{
		ID:        id,
		Params:    params,
		NodeID:    node.ID(),
		CreatedAt: now,
	}

	r := &Result{
		TaskID:  id,
		NodeID:  t.NodeID,
		Target:  params.Target,
		Type:    params.Type,
		Mode:    params.Mode,
		Started: now,
		Status:  "running",
		Rounds:  make([]RoundResult, 0),
		Total:   Stats{},
		Window:  nil,
		Error:   "",
	}

	e.mu.Lock()
	e.tasks[id] = t
	e.results[id] = r
	e.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	e.mu.Lock()
	e.cancel[id] = cancel
	e.mu.Unlock()

	go e.runTcp(ctx, t, r)

	return id, nil
}

// CreateUdpTask creates and starts a UDP probe task (target must be a reflector host:port).
func (e *Engine) CreateUdpTask(params TaskParams) (TaskID, error) {
	id := TaskID(node.NextTaskID())
	now := time.Now()

	t := &Task{
		ID:        id,
		Params:    params,
		NodeID:    node.ID(),
		CreatedAt: now,
	}

	r := &Result{
		TaskID:  id,
		NodeID:  t.NodeID,
		Target:  params.Target,
		Type:    params.Type,
		Mode:    params.Mode,
		Started: now,
		Status:  "running",
		Rounds:  make([]RoundResult, 0),
		Total:   Stats{},
		Window:  nil,
		Error:   "",
	}

	e.mu.Lock()
	e.tasks[id] = t
	e.results[id] = r
	e.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	e.mu.Lock()
	e.cancel[id] = cancel
	e.mu.Unlock()

	go e.runTwamp(ctx, t, r)

	return id, nil
}

// CreateTwampTask creates and starts a TWAMP-light task (target must be a standard Session-Reflector, typically host:862).
func (e *Engine) CreateTwampTask(params TaskParams) (TaskID, error) {
	id := TaskID(node.NextTaskID())
	now := time.Now()

	t := &Task{
		ID:        id,
		Params:    params,
		NodeID:    node.ID(),
		CreatedAt: now,
	}

	r := &Result{
		TaskID:  id,
		NodeID:  t.NodeID,
		Target:  params.Target,
		Type:    params.Type,
		Mode:    params.Mode,
		Started: now,
		Status:  "running",
		Rounds:  make([]RoundResult, 0),
		Total:   Stats{},
		Window:  nil,
		Error:   "",
	}

	e.mu.Lock()
	e.tasks[id] = t
	e.results[id] = r
	e.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	e.mu.Lock()
	e.cancel[id] = cancel
	e.mu.Unlock()

	go e.runTwamp(ctx, t, r)

	return id, nil
}

// CreatePassiveTask creates and starts a passive (interface counter) task.
// Target in params is the interface name (e.g. "eth0"). Modes: duration, continuous.
func (e *Engine) CreatePassiveTask(params TaskParams) (TaskID, error) {
	if params.Type == "" {
		params.Type = "passive"
	}
	params.Type = "passive"
	id := TaskID(node.NextTaskID())
	now := time.Now()

	t := &Task{
		ID:        id,
		Params:    params,
		NodeID:    node.ID(),
		CreatedAt: now,
	}

	r := &Result{
		TaskID:  id,
		NodeID:  t.NodeID,
		Target:  params.Target,
		Type:    "passive",
		Mode:    params.Mode,
		Started: now,
		Status:  "running",
		Rounds:  make([]RoundResult, 0),
		Total:   Stats{},
		Window:  nil,
		Error:   "",
	}

	e.mu.Lock()
	e.tasks[id] = t
	e.results[id] = r
	e.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	e.mu.Lock()
	e.cancel[id] = cancel
	e.mu.Unlock()

	go e.runPassive(ctx, t, r)

	return id, nil
}

// StopTask stops a running task.
func (e *Engine) StopTask(id TaskID) {
	e.mu.Lock()
	cancel, ok := e.cancel[id]
	e.mu.Unlock()
	if ok {
		cancel()
	}
}

// GetResult returns the latest result for a task.
func (e *Engine) GetResult(id TaskID) (*Result, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	r, ok := e.results[id]
	return r, ok
}

// ListResults returns a snapshot of all task results.
func (e *Engine) ListResults() []*Result {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]*Result, 0, len(e.results))
	for _, r := range e.results {
		out = append(out, r)
	}
	return out
}

// DeleteTask removes a task and its result from the engine.
// If the task is still running, it will be stopped first.
func (e *Engine) DeleteTask(id TaskID) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if cancel, ok := e.cancel[id]; ok {
		cancel()
		delete(e.cancel, id)
	}

	_, taskExists := e.tasks[id]
	_, resExists := e.results[id]
	if !taskExists && !resExists {
		return false
	}

	delete(e.tasks, id)
	delete(e.results, id)
	return true
}

func (e *Engine) runProbe(ctx context.Context, task *Task, result *Result) {
	log := logger.S()
	params := task.Params

	pinger := &protocol.ICMPPinger{
		Addr:      params.Target,
		IPVersion: params.IPVersion,
		Timeout:   params.Timeout,
		Size:      params.PacketSize,
	}

	switch params.Mode {
	case ModeCount:
		e.runPingCount(ctx, pinger, task, result)
	case ModeDuration:
		e.runPingDuration(ctx, pinger, task, result)
	case ModeContinuous:
		e.runPingContinuous(ctx, pinger, task, result)
	default:
		log.Warnf("unknown mode %v for task %s", params.Mode, task.ID)
	}
}

func (e *Engine) runTcp(ctx context.Context, task *Task, result *Result) {
	log := logger.S()
	params := task.Params

	network := "tcp"
	if params.IPVersion == "ipv4" {
		network = "tcp4"
	} else if params.IPVersion == "ipv6" {
		network = "tcp6"
	}
	pinger := &protocol.TCPPinger{
		Target:    params.Target,
		Timeout:   params.Timeout,
		Network:   network,
		SourceIP:  params.SourceIP,
		Interface: params.Interface,
	}

	switch params.Mode {
	case ModeCount:
		e.runPingCount(ctx, pinger, task, result)
	case ModeDuration:
		e.runPingDuration(ctx, pinger, task, result)
	case ModeContinuous:
		e.runPingContinuous(ctx, pinger, task, result)
	default:
		log.Warnf("unknown mode %v for task %s", params.Mode, task.ID)
	}
}

func (e *Engine) runUdp(ctx context.Context, task *Task, result *Result) {
	log := logger.S()
	params := task.Params

	network := "udp"
	if params.IPVersion == "ipv4" {
		network = "udp4"
	} else if params.IPVersion == "ipv6" {
		network = "udp6"
	}
	size := params.PacketSize
	if size < 8 {
		size = 8
	}
	pinger := &protocol.UDPProber{
		Target:     params.Target,
		Timeout:    params.Timeout,
		PacketSize: size,
		Network:    network,
		SourceIP:   params.SourceIP,
		Interface:  params.Interface,
	}

	switch params.Mode {
	case ModeCount:
		e.runPingCount(ctx, pinger, task, result)
	case ModeDuration:
		e.runPingDuration(ctx, pinger, task, result)
	case ModeContinuous:
		e.runPingContinuous(ctx, pinger, task, result)
	default:
		log.Warnf("unknown mode %v for task %s", params.Mode, task.ID)
	}
}

func (e *Engine) runTwamp(ctx context.Context, task *Task, result *Result) {
	log := logger.S()
	params := task.Params

	network := "udp"
	if params.IPVersion == "ipv4" {
		network = "udp4"
	} else if params.IPVersion == "ipv6" {
		network = "udp6"
	}
	size := params.PacketSize
	if size < 16 {
		size = 16
	}
	pinger := &protocol.TWAMPPinger{
		Target:     params.Target,
		Timeout:    params.Timeout,
		PacketSize: size,
		Network:    network,
		SourceIP:   params.SourceIP,
		Interface:  params.Interface,
	}

	switch params.Mode {
	case ModeCount:
		e.runPingCount(ctx, pinger, task, result)
	case ModeDuration:
		e.runPingDuration(ctx, pinger, task, result)
	case ModeContinuous:
		e.runPingContinuous(ctx, pinger, task, result)
	default:
		log.Warnf("unknown mode %v for task %s", params.Mode, task.ID)
	}
}

var passiveStatsSource source.Source = source.ProcNetDevSource{}

func (e *Engine) runPassive(ctx context.Context, task *Task, result *Result) {
	params := task.Params
	if params.Interval <= 0 {
		params.Interval = 10 * time.Second
	}
	switch params.Mode {
	case ModeDuration:
		e.runPassiveDuration(ctx, task, result)
	case ModeContinuous:
		e.runPassiveContinuous(ctx, task, result)
	default:
		e.runPassiveDuration(ctx, task, result)
	}
}

func (e *Engine) runPassiveDuration(ctx context.Context, task *Task, result *Result) {
	log := logger.S()
	params := task.Params
	if params.Duration <= 0 {
		params.Duration = 60 * time.Second
	}
	endTime := time.Now().Add(params.Duration)
	prev, err := passiveStatsSource.InterfaceStats(params.Target)
	if err != nil {
		log.Warnf("passive task %s initial read: %v", task.ID, err)
		e.updateResult(task.ID, func(res *Result) {
			res.Status = "failed"
			res.Finished = time.Now()
			res.Error = err.Error()
		})
		return
	}

	ticker := time.NewTicker(params.Interval)
	defer ticker.Stop()

	for time.Now().Before(endTime) {
		select {
		case <-ctx.Done():
			e.updateResult(task.ID, func(res *Result) {
				res.Status = "stopped"
				res.Finished = time.Now()
			})
			return
		case <-ticker.C:
		}

		snap, err := passiveStatsSource.InterfaceStats(params.Target)
		if err != nil {
			log.Warnf("passive task %s: %v", task.ID, err)
			e.updateResult(task.ID, func(res *Result) {
				res.Error = err.Error()
			})
			time.Sleep(params.Interval)
			continue
		}
		if prev != nil {
			delta := passive.Delta(prev, snap)
			roundStats := snapshotToStats(delta)
			e.updateResult(task.ID, func(res *Result) {
				addSnapshotToStats(&res.Total, delta)
				res.Rounds = append(res.Rounds, RoundResult{
					Index: len(res.Rounds) + 1,
					From:  time.Now().Add(-params.Interval),
					To:    time.Now(),
					Stats: roundStats,
				})
				res.Status = "running"
			})
		}
		prev = snap
	}

	e.updateResult(task.ID, func(res *Result) {
		res.Status = "finished"
		res.Finished = time.Now()
	})
	log.Infof("passive duration task finished: %s", task.ID)
}

func (e *Engine) runPassiveContinuous(ctx context.Context, task *Task, result *Result) {
	log := logger.S()
	params := task.Params
	windowSeconds := 60
	if params.Duration > 0 {
		windowSeconds = int(params.Duration.Seconds())
		if windowSeconds < 10 {
			windowSeconds = 10
		}
	}
	windowDur := time.Duration(windowSeconds) * time.Second
	type windowEntry struct {
		t time.Time
		d *source.InterfaceSnapshot
	}
	var prev *source.InterfaceSnapshot
	var windowEntries []windowEntry

	ticker := time.NewTicker(params.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			e.updateResult(task.ID, func(res *Result) {
				res.Status = "stopped"
				res.Finished = time.Now()
			})
			log.Infof("passive continuous task stopped: %s", task.ID)
			return
		case <-ticker.C:
		}

		snap, err := passiveStatsSource.InterfaceStats(params.Target)
		if err != nil {
			log.Warnf("passive task %s: %v", task.ID, err)
			e.updateResult(task.ID, func(res *Result) { res.Error = err.Error() })
			continue
		}
		if prev != nil {
			delta := passive.Delta(prev, snap)
			now := time.Now()
			windowEntries = append(windowEntries, windowEntry{t: now, d: delta})
			cutoff := now.Add(-windowDur)
			i := 0
			for _, e := range windowEntries {
				if !e.t.Before(cutoff) {
					windowEntries[i] = e
					i++
				}
			}
			windowEntries = windowEntries[:i]

			var windowSum source.InterfaceSnapshot
			for _, e := range windowEntries {
				windowSum.BytesIn += e.d.BytesIn
				windowSum.BytesOut += e.d.BytesOut
				windowSum.PacketsIn += e.d.PacketsIn
				windowSum.PacketsOut += e.d.PacketsOut
				windowSum.ErrorsIn += e.d.ErrorsIn
				windowSum.ErrorsOut += e.d.ErrorsOut
				windowSum.DropsIn += e.d.DropsIn
				windowSum.DropsOut += e.d.DropsOut
			}

			e.updateResult(task.ID, func(res *Result) {
				addSnapshotToStats(&res.Total, delta)
				if res.Window == nil {
					res.Window = &WindowStats{WindowSeconds: windowSeconds, Stats: snapshotToStats(&windowSum)}
				} else {
					res.Window.WindowSeconds = windowSeconds
					res.Window.Stats = snapshotToStats(&windowSum)
				}
				res.Status = "running"
			})
		}
		prev = snap
	}
}

func snapshotToStats(s *source.InterfaceSnapshot) Stats {
	return Stats{
		BytesIn: s.BytesIn, BytesOut: s.BytesOut,
		PacketsIn: s.PacketsIn, PacketsOut: s.PacketsOut,
		ErrorsIn: s.ErrorsIn, ErrorsOut: s.ErrorsOut,
		DropsIn: s.DropsIn, DropsOut: s.DropsOut,
	}
}

func addSnapshotToStats(st *Stats, s *source.InterfaceSnapshot) {
	st.BytesIn += s.BytesIn
	st.BytesOut += s.BytesOut
	st.PacketsIn += s.PacketsIn
	st.PacketsOut += s.PacketsOut
	st.ErrorsIn += s.ErrorsIn
	st.ErrorsOut += s.ErrorsOut
	st.DropsIn += s.DropsIn
	st.DropsOut += s.DropsOut
}

func (e *Engine) updateResult(id TaskID, fn func(*Result)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if r, ok := e.results[id]; ok {
		fn(r)
		r.LastUpdated = time.Now()
	}
}

// computeStats updates statistics given a new probe RTT.
// Jitter is the running average of |rtt - previous_rtt| (consecutive delay variation).
func computeStats(s *Stats, rtt time.Duration, ok bool) {
	s.Sent++
	if ok {
		s.Received++
		if s.MinRTT == 0 || rtt < s.MinRTT {
			s.MinRTT = rtt
		}
		if rtt > s.MaxRTT {
			s.MaxRTT = rtt
		}
		n := s.Received
		if n == 1 {
			s.AvgRTT = rtt
			s.lastRTT = rtt
		} else {
			s.AvgRTT = ((s.AvgRTT * time.Duration(n-1)) + rtt) / time.Duration(n)
			diff := rtt - s.lastRTT
			if diff < 0 {
				diff = -diff
			}
			s.Jitter = (s.Jitter*time.Duration(n-2) + diff) / time.Duration(n-1)
			s.lastRTT = rtt
		}
	}
	if s.Sent > 0 {
		s.LossRate = float64(s.Sent-s.Received) / float64(s.Sent)
	}
}
