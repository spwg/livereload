package livereload

import (
	"io"
	"log"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

// === Mocks ===

type MockWatcher struct {
	events chan fsnotify.Event
	errors chan error
}

func NewMockWatcher() *MockWatcher {
	return &MockWatcher{
		events: make(chan fsnotify.Event),
		errors: make(chan error),
	}
}

func (m *MockWatcher) Add(name string) error { return nil }
func (m *MockWatcher) Close() error          { return nil }
func (m *MockWatcher) Events() chan fsnotify.Event {
	return m.events
}
func (m *MockWatcher) Errors() chan error {
	return m.errors
}

type MockCommandRunner struct {
	RunHistory   []string
	StartHistory []string
	RunError     error
	StartError   error
	MockProcess  *MockProcess
}

func (m *MockCommandRunner) Run(cmd string) error {
	m.RunHistory = append(m.RunHistory, cmd)
	return m.RunError
}

func (m *MockCommandRunner) Start(cmd string) (Process, error) {
	m.StartHistory = append(m.StartHistory, cmd)
	if m.StartError != nil {
		return nil, m.StartError
	}
	if m.MockProcess == nil {
		return &MockProcess{}, nil
	}
	return m.MockProcess, nil
}

type MockProcess struct {
	KillCalled bool
	WaitCalled bool
}

func (m *MockProcess) Kill() error {
	m.KillCalled = true
	return nil
}

func (m *MockProcess) Wait() error {
	m.WaitCalled = true
	return nil
}

// === Tests ===

func TestLivereload_Unit(t *testing.T) {
	mockWatcher := NewMockWatcher()
	mockRunner := &MockCommandRunner{}
	mockProcess := &MockProcess{}
	mockRunner.MockProcess = mockProcess

	app := &Livereload{
		Watcher:      mockWatcher,
		Runner:       mockRunner,
		BuildCmd:     "go build",
		RunCmd:       "./app",
		IgnoreMap:    make(map[string]bool),
		DebounceTime: 10 * time.Millisecond, // Short debounce for test
		RestartDelay: 10 * time.Millisecond,
		Log:          log.New(io.Discard, "", 0),
		Hub:          NewReloadHub(),
	}

	// run app.Run() in a goroutine because it blocks
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run()
	}()

	// Wait for initial run
	// Since Run() executes "Initial run" on start, we should see StartHistory have 1 item eventually
	// We need to wait a small bit for the goroutine to proceed
	time.Sleep(50 * time.Millisecond)

	if len(mockRunner.StartHistory) != 1 {
		t.Fatalf("Expected 1 start, got %d", len(mockRunner.StartHistory))
	}
	if mockRunner.StartHistory[0] != "./app" {
		t.Errorf("Expected run command './app', got '%s'", mockRunner.StartHistory[0])
	}
	// Initial run checks build command too if buildCmd is present
	if len(mockRunner.RunHistory) != 0 {
		// Wait, app.BuildCmd IS set.
		// Wait, the order: check build first
	}

	if len(mockRunner.RunHistory) != 1 {
		t.Fatalf("Expected 1 build, got %d", len(mockRunner.RunHistory))
	}
	if mockRunner.RunHistory[0] != "go build" {
		t.Errorf("Expected build command 'go build', got '%s'", mockRunner.RunHistory[0])
	}

	// Reset history for next check
	mockRunner.RunHistory = nil
	mockRunner.StartHistory = nil

	// Trigger a file event
	mockWatcher.events <- fsnotify.Event{Name: "main.go", Op: fsnotify.Write}

	// Wait for debounce + processing
	time.Sleep(100 * time.Millisecond)

	// Should have killed the old process, built, and run again
	if !mockProcess.KillCalled {
		t.Error("Expected previous process to be killed")
	}
	if !mockProcess.WaitCalled {
		t.Error("Expected previous process to be waited on")
	}

	if len(mockRunner.RunHistory) != 1 {
		t.Errorf("Expected 1 build after edit, got %d", len(mockRunner.RunHistory))
	}
	if len(mockRunner.StartHistory) != 1 {
		t.Errorf("Expected 1 run after edit, got %d", len(mockRunner.StartHistory))
	}

	// Stop the loop (not implemented in Run() strictly, so we just kill test)
}
