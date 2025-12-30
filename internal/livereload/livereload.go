package livereload

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher interface abstraction for fsnotify.Watcher
type FileWatcher interface {
	Add(name string) error
	Close() error
	Events() chan fsnotify.Event
	Errors() chan error
}

// RealWatcher wraps fsnotify.Watcher to implement FileWatcher
type RealWatcher struct {
	*fsnotify.Watcher
}

func (w *RealWatcher) Events() chan fsnotify.Event {
	return w.Watcher.Events
}

func (w *RealWatcher) Errors() chan error {
	return w.Watcher.Errors
}

// CommandRunner interface for running commands
type CommandRunner interface {
	Run(cmd string) error
	Start(cmd string) (Process, error)
}

// Process interface for controlling a running process
type Process interface {
	Kill() error
	Wait() error
}

// RealCommandRunner implements CommandRunner using exec.Command
type RealCommandRunner struct{}

func (r *RealCommandRunner) Run(cmdStr string) error {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *RealCommandRunner) Start(cmdStr string) (Process, error) {
	cmd := exec.Command("sh", "-c", cmdStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &RealProcess{cmd}, nil
}

// RealProcess wraps exec.Cmd
type RealProcess struct {
	cmd *exec.Cmd
}

func (p *RealProcess) Kill() error {
	return p.cmd.Process.Kill()
}

func (p *RealProcess) Wait() error {
	return p.cmd.Wait()
}

// Livereload logic struct
type Livereload struct {
	Watcher        FileWatcher
	Runner         CommandRunner
	BuildCmd       string
	RunCmd         string
	IgnoreMap      map[string]bool
	DebounceTime   time.Duration
	RestartDelay   time.Duration
	Log            *log.Logger
	ReloadPort     int
	ReloadHost     string
	Hub            *ReloadHub
	LivereloadJS   []byte
	HealthURL      string
	HealthTimeout  time.Duration
	HealthInterval time.Duration
}

func NewLivereload(buildCmd, runCmd string, ignoreMap map[string]bool, watcher FileWatcher, reloadPort int, reloadHost string, livereloadJS []byte) *Livereload {
	return &Livereload{
		Watcher:        watcher,
		Runner:         &RealCommandRunner{},
		BuildCmd:       buildCmd,
		RunCmd:         runCmd,
		IgnoreMap:      ignoreMap,
		DebounceTime:   100 * time.Millisecond,
		RestartDelay:   100 * time.Millisecond,
		Log:            log.New(os.Stdout, "", log.LstdFlags),
		ReloadPort:     reloadPort,
		ReloadHost:     reloadHost,
		Hub:            NewReloadHub(),
		LivereloadJS:   livereloadJS,
		HealthURL:      "",
		HealthTimeout:  5 * time.Second,
		HealthInterval: 50 * time.Millisecond,
	}
}

// waitForHealth polls the HealthURL until it returns 200 OK or times out.
// Returns nil if health check passes, error if it times out.
func (app *Livereload) waitForHealth() error {
	if app.HealthURL == "" {
		// No health URL configured, fall back to delay
		if app.RestartDelay > 0 {
			time.Sleep(app.RestartDelay)
		}
		return nil
	}

	deadline := time.Now().Add(app.HealthTimeout)
	client := &http.Client{Timeout: app.HealthInterval}

	for time.Now().Before(deadline) {
		resp, err := client.Get(app.HealthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return nil
			}
		}
		time.Sleep(app.HealthInterval)
	}

	app.Log.Printf("Warning: Health check timed out after %v", app.HealthTimeout)
	return nil // Still proceed with reload even if health check fails
}

func (app *Livereload) Run() error {
	// Start the reload server
	app.StartServer()

	// Channel to signal a rebuild/restart is needed
	restartCh := make(chan bool, 1)

	// Debounce timer
	var debounceTimer *time.Timer

	go func() {
		for {
			select {
			case event, ok := <-app.Watcher.Events():
				if !ok {
					return
				}
				// Skip ignored files
				if app.IgnoreMap[filepath.Base(event.Name)] {
					continue
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Remove == fsnotify.Remove {
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(app.DebounceTime, func() {
						app.Log.Printf("Modified file: %s", event.Name)
						select {
						case restartCh <- true:
						default:
						}
					})
				}
			case err, ok := <-app.Watcher.Errors():
				if !ok {
					return
				}
				app.Log.Println("error:", err)
			}
		}
	}()

	// Initial run
	restartCh <- true

	var currentProcess Process

	for range restartCh {
		if currentProcess != nil {
			if err := currentProcess.Kill(); err != nil {
				// Optimization: ignore "process already finished" errors
				if !strings.Contains(err.Error(), "process already finished") && !strings.Contains(err.Error(), "os: process already finished") {
					app.Log.Printf("Failed to kill process: %v", err)
				}
			}
			if err := currentProcess.Wait(); err != nil {
				// Ignore signal killed errors as they are expected
				if !strings.Contains(err.Error(), "signal: killed") && !strings.Contains(err.Error(), "process already finished") {
					app.Log.Printf("Process finished with error: %v", err)
				}
			}
		}

		if app.BuildCmd != "" {
			fmt.Println(">> Building...")
			if err := app.Runner.Run(app.BuildCmd); err != nil {
				fmt.Printf(">> Build failed: %v\n", err)
				continue // Don't run if build fails
			}
		}

		fmt.Println(">> Running...")
		p, err := app.Runner.Start(app.RunCmd)
		if err != nil {
			fmt.Printf(">> Run failed: %v\n", err)
			continue
		}
		currentProcess = p

		// Wait for the server to be ready
		if err := app.waitForHealth(); err != nil {
			fmt.Printf(">> Health check failed: %v\n", err)
			continue
		}

		// Notify clients to reload after the server has restarted
		app.Hub.broadcast <- []byte("reload")

		// Wait for process in a goroutine so we don't block the loop
		// We don't necessarily need to Wait() here for the loop logic since we handle headers in the restart
		// but standard practice to avoid zombies happens in restart logic via Wait()
	}
	return nil
}

func AddRecursiveWatch(watcher FileWatcher, paths []string, ignoreMap map[string]bool) error {
	for _, p := range paths {
		p = strings.TrimSpace(p)
		err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if ignoreMap[info.Name()] {
					return filepath.SkipDir
				}
				err = watcher.Add(path)
				if err != nil {
					log.Printf("Failed to watch %s: %v", path, err)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
