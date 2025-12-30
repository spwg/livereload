# LiveReload Example

This is a repeatable example demonstrating how to use the `livereload` tool with a Go application that embeds HTML.

## Structure

- `main.go`: A simple Go web server using `go:embed`.
- `index.html`: The HTML template served by `main.go`. It includes the livereload script.
- `livereload.toml`: Configuration for the `livereload` tool.

## How to use

1.  **Build the `livereload` tool** (if not already done):
    ```bash
    go build -o livereload main.go
    ```

2.  **Run the example** from this directory:
    ```bash
    ../livereload
    ```
    This will start the livereload server (on :35729) and the example application (on :8080).

3.  **Open the app**:
    Navigate to `http://localhost:8080` in your browser.

4.  **Trigger a reload**:
    Modify `index.html` or `main.go`. The tool will automatically:
    - Rebuild the application (`go build -o app main.go`)
    - Restart the process
    - Notify the browser to reload the page via WebSockets

## Configuration

The example uses `livereload.toml`:
```toml
run = "./app"
build = "go build -o app main.go"
watch = ["."]
ignore = [".git", "app"]
```
Note that `app` is ignored to prevent the build output itself from triggering an infinite reload loop.
