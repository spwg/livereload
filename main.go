package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spwg/livereload/internal/livereload"
)

func main() {
	var (
		buildCmd   string
		runCmd     string
		watchPaths string
		ignoreDirs string
	)

	flag.StringVar(&buildCmd, "build", "", "Command to build the project")
	flag.StringVar(&runCmd, "run", "", "Command to run the executable")
	flag.StringVar(&watchPaths, "watch", ".", "Comma-separated list of directories/files to watch")
	flag.StringVar(&ignoreDirs, "ignore", ".git,node_modules", "Comma-separated list of directories to ignore")
	flag.Parse()

	if runCmd == "" {
		log.Fatal("Error: --run flag is required")
	}

	watchList := strings.Split(watchPaths, ",")
	ignoreList := strings.Split(ignoreDirs, ",")
	ignoreMap := make(map[string]bool)
	for _, dir := range ignoreList {
		ignoreMap[strings.TrimSpace(dir)] = true
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	// We need to implement or expose RealWatcher in the package or just pass fsWatcher if the interface matches.
	// The previous main.go had RealWatcher.
	// In pkg/livereload, we have RealWatcher exported if we capitalized it?
	// Wait, I defined RealWatcher in pkg/livereload but the field Watcher on Livereload expects FileWatcher interface.
	// RealWatcher implements FileWatcher.
	// Let's check `pkg/livereload/livereload.go` again.
	// Yes, RealWatcher is exported.
	realWatcher := &livereload.RealWatcher{Watcher: fsWatcher}
	defer realWatcher.Close()

	// Recursively add paths using the helper from the package
	if err := livereload.AddRecursiveWatch(realWatcher, watchList, ignoreMap); err != nil {
		log.Fatal(err)
	}

	app := livereload.NewLivereload(buildCmd, runCmd, ignoreMap, realWatcher)

	fmt.Printf("Livereload started.\n")
	fmt.Printf("Build command: %s\n", buildCmd)
	fmt.Printf("Run command: %s\n", runCmd)
	fmt.Printf("Watching: %v\n", watchList)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
