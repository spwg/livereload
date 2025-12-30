package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pelletier/go-toml/v2"
	"github.com/spwg/livereload/internal/livereload"
)

//go:embed js/livereload.js
var LivereloadJs []byte

type Config struct {
	Build     string   `toml:"build"`
	Run       string   `toml:"run"`
	Watch     []string `toml:"watch"`
	Ignore    []string `toml:"ignore"`
	Delay     int      `toml:"delay"`
	HealthURL string   `toml:"health_url"`
}

func main() {
	var (
		buildCmd   string
		runCmd     string
		watchPaths string
		ignoreDirs string
		port       int
		host       string
		delay      int
		healthURL  string
	)

	flag.StringVar(&buildCmd, "build", "", "Command to build the project")
	flag.StringVar(&runCmd, "run", "", "Command to run the executable")
	flag.StringVar(&watchPaths, "watch", "", "Comma-separated list of directories/files to watch")
	flag.StringVar(&ignoreDirs, "ignore", "", "Comma-separated list of directories to ignore")
	flag.IntVar(&port, "port", 35729, "Port for the livereload server")
	flag.StringVar(&host, "host", "localhost", "Host for the livereload server")
	flag.IntVar(&delay, "delay", 100, "Delay in milliseconds after restart before reload (fallback if no health_url)")
	flag.StringVar(&healthURL, "health-url", "", "URL to poll for health check before reloading")
	flag.Parse()

	// Load config from file
	var cfg Config
	if data, err := os.ReadFile("livereload.toml"); err == nil {
		if err := toml.Unmarshal(data, &cfg); err != nil {
			log.Fatalf("Failed to parse livereload.toml: %v", err)
		}
		fmt.Println("Loaded configuration from livereload.toml")
	}

	// Override with flags if set
	if buildCmd != "" {
		cfg.Build = buildCmd
	}
	if runCmd != "" {
		cfg.Run = runCmd
	}
	if watchPaths != "" {
		cfg.Watch = strings.Split(watchPaths, ",")
	}
	if ignoreDirs != "" {
		cfg.Ignore = strings.Split(ignoreDirs, ",")
	}
	if delay >= 0 {
		cfg.Delay = delay
	}
	if healthURL != "" {
		cfg.HealthURL = healthURL
	}

	// Defaults if nothing set
	if len(cfg.Watch) == 0 {
		cfg.Watch = []string{"."}
	}
	if len(cfg.Ignore) == 0 {
		cfg.Ignore = []string{".git", "node_modules"}
	}

	if cfg.Run == "" {
		log.Fatal("Error: --run flag or 'run' in livereload.toml is required")
	}

	ignoreMap := make(map[string]bool)
	for _, dir := range cfg.Ignore {
		ignoreMap[strings.TrimSpace(dir)] = true
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	realWatcher := &livereload.RealWatcher{Watcher: fsWatcher}
	defer realWatcher.Close()

	// Recursively add paths using the helper from the package
	if err := livereload.AddRecursiveWatch(realWatcher, cfg.Watch, ignoreMap); err != nil {
		log.Fatal(err)
	}

	app := livereload.NewLivereload(cfg.Build, cfg.Run, ignoreMap, realWatcher, port, host, LivereloadJs)
	app.RestartDelay = time.Duration(cfg.Delay) * time.Millisecond
	app.HealthURL = cfg.HealthURL

	fmt.Printf("Livereload started.\n")
	fmt.Printf("Build command: %s\n", cfg.Build)
	fmt.Printf("Run command: %s\n", cfg.Run)
	fmt.Printf("Watching: %v\n", cfg.Watch)
	fmt.Printf("Livereload Server: http://%s:%d/livereload.js\n", host, port)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
