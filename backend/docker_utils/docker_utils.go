package docker_utils

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// EnsureChromaContainer starts the Chroma container if not running (existing code)
func EnsureChromaContainer() error {
	// 1️⃣ Check if container exists
	cmd := exec.Command("docker", "ps", "-a", "--filter", "name=chroma-kibo", "--format", "{{.Names}}")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("docker ps failed: %w", err)
	}

	names := strings.Split(strings.TrimSpace(string(out)), "\n")
	exists := false
	for _, n := range names {
		if n == "chroma-kibo" {
			exists = true
			break
		}
	}

	if exists {
		// Check if it is running
		cmd = exec.Command("docker", "inspect", "-f", "{{.State.Running}}", "chroma-kibo")
		out, err = cmd.Output()
		if err != nil {
			return fmt.Errorf("docker inspect failed: %w", err)
		}
		if strings.TrimSpace(string(out)) == "true" {
			fmt.Println("Chroma container already running")
			return nil
		}

		// Start existing container
		cmd = exec.Command("docker", "start", "chroma-kibo")
		return cmd.Run()
	}

	// Container doesn't exist → create and run
	wd, _ := os.Getwd()

	// Assume you want the mount at project root: "../data/chroma"
	chromaPath := fmt.Sprintf("%s/../data/chroma", wd)

	cmd = exec.Command(
		"docker", "run", "-d",
		"--name", "chroma-kibo",
		"-p", "8000:8000",
		"-v", fmt.Sprintf("%s:/chroma", chromaPath),
		"chromadb/chroma",
	)
	return cmd.Run()
}

// ListChromaContainers prints all Docker containers and highlights the one we use
func ListChromaContainers() error {
	// List all containers (running and stopped)
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{.ID}}\t{{.Names}}\t{{.Status}}")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("docker ps failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println("No Docker containers found.")
		return nil
	}

	fmt.Println("Docker containers on this system:")
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			continue
		}
		id, name, status := fields[0], fields[1], fields[2]

		if name == "chroma-kibo" {
			fmt.Printf("→ [USING] %s (%s) Status: %s\n", name, id, status)
		} else {
			fmt.Printf("   %s (%s) Status: %s\n", name, id, status)
		}
	}

	return nil
}

// WaitForChromaReady polls the Chroma heartbeat endpoint until it's ready or times out
// it will wait to safely ensure Chroma is ready to accept requests
func WaitForChromaReady() error {
	url := "http://localhost:8000/api/v2/heartbeat"
	timeout := 20 * time.Second
	start := time.Now()

	fmt.Println("⏳ Waiting for Chroma server to be ready...")

	for time.Since(start) < timeout {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == 200 {
			fmt.Println("✅ Chroma is ready!")
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return errors.New("❌ Chroma server did not become ready in time")
}
