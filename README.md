# Livereload CLI Tool

A lightweight, customizable command-line tool that watches for file changes and automatically rebuilds and restarts your application. Designed to create a fast inner development loop.

## how it works

The tool uses `fsnotify` to listen for file system events (create, write, remove) in the specified directories. When a change is detected:

1.  **Debounce**: It waits for a short period (100ms) to coalesce multiple events (e.g., "Save All").
2.  **Kill**: It terminates the currently running process (if any).
3.  **Build**: It runs the specified build command (optional).
4.  **Run**: It starts the application using the run command.

## system requirements

-   **OS**: macOS, Linux, or other Unix-like systems.
    -   *Note*: The current implementation uses process signaling that is optimized for Unix-based systems. Windows support is experimental or may require adjustments to process killing logic.
-   **Go**: Go 1.20+ (to build the tool itself).

## installation

Clone the repository and build the tool:

```bash
go build -o livereload main.go
```

## usage

```bash
./livereload --build "<build_command>" --run "<run_command>" [options]
```

### options

-   `--build`: The command to build your project (e.g., `go build -o app main.go`). If omitted, the tool skips the build step and just runs.
-   `--run` (Required): The command to run your executable (e.g., `./app`).
-   `--watch`: Comma-separated list of directories or files to watch. Defaults to the current directory (`.`).
-   `--ignore`: Comma-separated list of directories to ignore. Defaults to `.git,node_modules`.

### examples

**Go Application:**

```bash
./livereload \
  --build "go build -o myapp main.go" \
  --run "./myapp" \
  --watch "."
```

**Node.js Application:**

Since Node.js doesn't need a build step, just use `--run`:

```bash
./livereload \
  --run "node index.js" \
  --watch "."
```

**Python Script:**

```bash
./livereload \
  --run "python3 main.py" \
  --watch "src"
```
