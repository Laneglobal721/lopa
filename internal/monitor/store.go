package monitor

import (
	"fmt"
	"sync"
	"time"

	"github.com/yanjiulab/lopa/internal/config"
	"github.com/yanjiulab/lopa/internal/node"
)

var (
	defaultStore *Store
	storeOnce    sync.Once
)

// DefaultStore returns the singleton monitor store (buffer size from config).
func DefaultStore() *Store {
	storeOnce.Do(func() {
		size := 100
		if c := config.Global(); c != nil && c.Monitor.EventBufferSize > 0 {
			size = c.Monitor.EventBufferSize
		}
		defaultStore = NewStore(size)
	})
	return defaultStore
}

// Store holds monitor tasks and optional event buffers. Thread-safe.
type Store struct {
	mu      sync.RWMutex
	tasks   map[string]*Task
	events  map[string][]Event // taskID -> ring of events (newest last)
	bufSize int
	nextSeq uint64
}

// NewStore creates a store with optional event buffer size per task (0 = no buffer).
func NewStore(eventBufferSize int) *Store {
	if eventBufferSize <= 0 {
		eventBufferSize = 100
	}
	return &Store{
		tasks:   make(map[string]*Task),
		events:  make(map[string][]Event),
		bufSize: eventBufferSize,
	}
}

// AddTask adds a monitor task and returns its ID.
func (s *Store) AddTask(t *Task) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t.ID == "" {
		t.ID = node.NextTaskID()
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Type == "" {
		t.Type = TypeInterface
	}
	if !t.Enabled {
		t.Enabled = true
	}
	s.tasks[t.ID] = t
	return t.ID
}

// GetTask returns a task by ID.
func (s *Store) GetTask(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, false
	}
	// Return a copy so callers don't mutate
	t2 := *t
	return &t2, true
}

// ListTasks returns all tasks.
func (s *Store) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		t2 := *t
		out = append(out, &t2)
	}
	return out
}

// UpdateTask updates an existing task (webhook, enabled, filter).
func (s *Store) UpdateTask(id string, fn func(*Task)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return false
	}
	fn(t)
	t.UpdatedAt = time.Now()
	return true
}

// DeleteTask removes a task and its event buffer.
func (s *Store) DeleteTask(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return false
	}
	delete(s.tasks, id)
	delete(s.events, id)
	return true
}

// TasksForType returns enabled tasks that watch the given event type (caller holds no lock; returns copies).
func (s *Store) TasksForType(typ TaskType) []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Task
	for _, t := range s.tasks {
		if t.Enabled && t.Type == typ {
			t2 := *t
			out = append(out, &t2)
		}
	}
	return out
}

// AppendEvent records an event for a task and returns the event with ID/At set (for webhook).
func (s *Store) AppendEvent(taskID string, evt Event) Event {
	s.nextSeq++
	evt.ID = fmt.Sprintf("evt-%d-%d", time.Now().UnixNano(), s.nextSeq)
	evt.TaskID = taskID
	evt.At = time.Now()

	s.mu.Lock()
	buf := s.events[taskID]
	buf = append(buf, evt)
	if len(buf) > s.bufSize {
		buf = buf[len(buf)-s.bufSize:]
	}
	s.events[taskID] = buf
	s.mu.Unlock()
	return evt
}

// GetEvents returns the last N events for a task (newest last).
func (s *Store) GetEvents(taskID string, lastN int) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	buf := s.events[taskID]
	if lastN <= 0 || lastN > len(buf) {
		lastN = len(buf)
	}
	if lastN == 0 {
		return nil
	}
	start := len(buf) - lastN
	out := make([]Event, lastN)
	copy(out, buf[start:])
	return out
}
