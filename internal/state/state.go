package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
)

type State struct {
	Version int                  `json:"version"`
	Loaded  map[string]LoadEntry `json:"loaded"`
}

type LoadEntry struct {
	DirName  string    `json:"dir_name"`
	Source   string    `json:"source"`
	Bundles  []string  `json:"bundles"`
	LoadedAt time.Time `json:"loaded_at"`
}

type StateManager struct {
	path     string
	lockPath string
	lockFile *os.File
}

func NewManager() (*StateManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".local", "state", "skl")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &StateManager{
		path:     filepath.Join(dir, "state.json"),
		lockPath: filepath.Join(dir, "state.lock"),
	}, nil
}

func (m *StateManager) Lock() error {
	f, err := os.OpenFile(m.lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("opening lock file: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return fmt.Errorf("acquiring lock: %w", err)
	}
	m.lockFile = f
	return nil
}

func (m *StateManager) Unlock() {
	if m.lockFile != nil {
		syscall.Flock(int(m.lockFile.Fd()), syscall.LOCK_UN)
		m.lockFile.Close()
		m.lockFile = nil
	}
}

func (m *StateManager) Load() (*State, error) {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Version: 1, Loaded: map[string]LoadEntry{}}, nil
		}
		return nil, fmt.Errorf("reading state: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}
	if s.Loaded == nil {
		s.Loaded = map[string]LoadEntry{}
	}
	if s.Version == 0 {
		s.Version = 1
	}
	return &s, nil
}

func (m *StateManager) Save(s *State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}
	return os.Rename(tmp, m.path)
}

func (s *State) AddBundleClaim(skillID, dirName, src, bundle string) {
	entry, ok := s.Loaded[skillID]
	if !ok {
		s.Loaded[skillID] = LoadEntry{
			DirName:  dirName,
			Source:   src,
			Bundles:  []string{bundle},
			LoadedAt: time.Now().UTC(),
		}
		return
	}
	for _, b := range entry.Bundles {
		if b == bundle {
			return
		}
	}
	entry.Bundles = append(entry.Bundles, bundle)
	sort.Strings(entry.Bundles)
	s.Loaded[skillID] = entry
}

// RemoveBundleClaim drops the bundle from the skill's claim list. Returns true
// if no bundles remain — caller should remove the skill from disk and call
// RemoveLoaded.
func (s *State) RemoveBundleClaim(skillID, bundle string) (orphaned bool) {
	entry, ok := s.Loaded[skillID]
	if !ok {
		return false
	}
	filtered := make([]string, 0, len(entry.Bundles))
	for _, b := range entry.Bundles {
		if b != bundle {
			filtered = append(filtered, b)
		}
	}
	if len(filtered) == 0 {
		return true
	}
	entry.Bundles = filtered
	s.Loaded[skillID] = entry
	return false
}

func (s *State) RemoveLoaded(skillID string) {
	delete(s.Loaded, skillID)
}

func (s *State) LoadedBundles() []string {
	seen := map[string]bool{}
	for _, e := range s.Loaded {
		for _, b := range e.Bundles {
			seen[b] = true
		}
	}
	out := make([]string, 0, len(seen))
	for b := range seen {
		out = append(out, b)
	}
	sort.Strings(out)
	return out
}

func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
