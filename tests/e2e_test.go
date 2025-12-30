package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestEndToEnd verifies the livereload functionality by creating a temporary
// directory, a dummy Go program, and running the livereload tool against it.
func TestEndToEnd(t *testing.T) {
	// 1. Setup temporary directory
	tempDir, err := os.MkdirTemp("", "livereload_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 2. Create a dummy "app" in the temp dir
	mainFile := filepath.Join(tempDir, "main.go")
	if err := writeMainFile(mainFile, "Hello Version 1"); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	// 3. Prepare to run the livereload tool
	livereloadBin := filepath.Join(tempDir, "livereload_bin")
	buildCmd := exec.Command("go", "build", "-o", livereloadBin, "..")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build livereload: %v", err)
	}

	// Now run livereload in the temp dir
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, livereloadBin,
		"--build", "go build -o app main.go",
		"--run", "./app",
		"--watch", ".",
	)
	cmd.Dir = tempDir

	// Capture stdout
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start livereload: %v", err)
	}

	// Helper to scan for expected output with timeout
	waitForOutput := func(expected string, timeout time.Duration) error {
		deadline := time.Now().Add(timeout)
		buf := make([]byte, 1024)
		for time.Now().Before(deadline) {
			n, _ := stdoutPipe.Read(buf)
			if n > 0 {
				output := string(buf[:n])
				if strings.Contains(output, expected) {
					return nil
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
		return fmt.Errorf("timeout waiting for %q", expected)
	}

	// 4. Verify Version 1
	t.Log("Waiting for Version 1...")
	if err := waitForOutput("Hello Version 1", 10*time.Second); err != nil {
		t.Fatalf("Failed to see Version 1: %v", err)
	}

	// 5. Modify file to Version 2
	t.Log("Modifying file to Version 2...")
	time.Sleep(1 * time.Second)
	if err := writeMainFile(mainFile, "Hello Version 2"); err != nil {
		t.Fatalf("Failed to update main.go: %v", err)
	}

	// 6. Verify Version 2
	t.Log("Waiting for Version 2...")
	if err := waitForOutput("Hello Version 2", 10*time.Second); err != nil {
		t.Fatalf("Failed to see Version 2: %v", err)
	}
}

func TestTOMLConfig(t *testing.T) {
	// 1. Setup temporary directory
	tempDir, err := os.MkdirTemp("", "livereload_toml_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 2. Create a dummy "app" in the temp dir
	mainFile := filepath.Join(tempDir, "main.go")
	if err := writeMainFile(mainFile, "Hello TOML"); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	// 3. Create livereload.toml
	tomlContent := `
build = "go build -o app_toml main.go"
run = "./app_toml"
watch = ["."]
ignore = [".git"]
`
	if err := os.WriteFile(filepath.Join(tempDir, "livereload.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to create livereload.toml: %v", err)
	}

	// 4. Build livereload binary
	livereloadBin := filepath.Join(tempDir, "livereload_bin")
	buildCmd := exec.Command("go", "build", "-o", livereloadBin, "..")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build livereload: %v", err)
	}

	// 5. Run livereload WITHOUT arguments
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, livereloadBin)
	cmd.Dir = tempDir

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start livereload: %v", err)
	}

	// Helper to scan for expected output
	waitForOutput := func(expected string, timeout time.Duration) error {
		deadline := time.Now().Add(timeout)
		buf := make([]byte, 1024)
		for time.Now().Before(deadline) {
			n, _ := stdoutPipe.Read(buf)
			if n > 0 {
				output := string(buf[:n])
				if strings.Contains(output, expected) {
					return nil
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
		return fmt.Errorf("timeout waiting for %q", expected)
	}

	t.Log("Waiting for Hello TOML...")
	if err := waitForOutput("Hello TOML", 10*time.Second); err != nil {
		t.Fatalf("Failed to see Hello TOML: %v", err)
	}
}

func writeMainFile(path, checkString string) error {
	content := fmt.Sprintf(`package main
import "fmt"
func main() {
	fmt.Println("%s")
}`, checkString)
	return os.WriteFile(path, []byte(content), 0644)
}
