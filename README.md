# Livereload CLI Tool

A lightweight, customizable command-line tool that watches for file changes and automatically rebuilds and restarts your application. Designed to create a fast inner development loop.

## How It Works

The tool uses `fsnotify` to listen for file system events (create, write, remove) in the specified directories. When a change is detected:

1.  **Debounce**: It waits for a short period (100ms) to coalesce multiple events (e.g., "Save All").
2.  **Kill**: It terminates the currently running process (if any).
3.  **Build**: It runs the specified build command (optional).
4.  **Run**: It starts the application using the run command.
5.  **Health Check**: It waits for the server to be ready (via HTTP health check or delay).
6.  **Reload**: It notifies connected browsers to refresh via WebSocket.

## System Requirements

-   **OS**: macOS, Linux, or other Unix-like systems.
    -   *Note*: The current implementation uses process signaling that is optimized for Unix-based systems. Windows support is experimental or may require adjustments to process killing logic.
-   **Go**: Go 1.20+ (to build the tool itself).

## Installation

Clone the repository and build the tool:

```bash
go build -o livereload main.go
```

## Usage

### CLI Flags

```bash
./livereload --build "<build_command>" --run "<run_command>" [options]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--build` | Command to build your project | (none) |
| `--run` | **Required.** Command to run your executable | (none) |
| `--watch` | Comma-separated directories/files to watch | `.` |
| `--ignore` | Comma-separated directories/files to ignore | `.git,node_modules` |
| `--port` | Port for the livereload WebSocket server | `35729` |
| `--host` | Host for the livereload server to bind to | `localhost` |
| `--health-url` | URL to poll for health check before reloading | (none) |
| `--delay` | Fallback delay (ms) after restart if no health URL | `100` |

### Configuration File (livereload.toml)

Instead of CLI flags, you can use a `livereload.toml` file in the current directory:

```toml
build = "go build -o app main.go"
run = "./app"
watch = ["."]
ignore = [".git", "node_modules", "app"]
health_url = "http://localhost:8080"
delay = 100
```

CLI flags take precedence over the config file.

## Automatic Browser Reload

To enable automatic browser refreshing:

1.  Add the following script to your HTML file(s) or template:

    ```html
    <script src="http://localhost:35729/livereload.js"></script>
    ```

    *Note: The port `35729` is the default. If you change it with `--port`, update the script tag accordingly.*

2.  Run `livereload` as usual. The tool will automatically notify the browser to reload whenever the server restarts.

## Health Check vs Delay

The tool needs to know when your server is ready before telling the browser to reload. There are two mechanisms:

### HTTP Health Check (Recommended)

Set `health_url` to a URL your server responds to. The tool will poll this URL every 50ms until it returns a 2xx or 3xx status code, then trigger the reload.

```toml
health_url = "http://localhost:8080"
```

This ensures the browser reloads exactly when your server is ready—no flicker or failed requests.

### Fallback Delay

If no `health_url` is configured, the tool will wait for `delay` milliseconds after starting the process before triggering the reload. This is less reliable but works for simple cases.

```toml
delay = 200
```

## Examples

### Go Application with Health Check

```bash
./livereload \
  --build "go build -o myapp main.go" \
  --run "./myapp" \
  --watch "." \
  --ignore ".git,myapp" \
  --health-url "http://localhost:8080"
```

Or with `livereload.toml`:

```toml
build = "go build -o myapp main.go"
run = "./myapp"
watch = ["."]
ignore = [".git", "myapp"]
health_url = "http://localhost:8080"
```

### Node.js Application

Since Node.js doesn't need a build step, just use `run`:

```bash
./livereload \
  --run "node index.js" \
  --watch "." \
  --health-url "http://localhost:3000"
```

### Python Script

```bash
./livereload \
  --run "python3 main.py" \
  --watch "src" \
  --delay 500
```

## Example Project

See the `example/` directory for a complete working example with a Go server that embeds HTML using `go:embed`.

