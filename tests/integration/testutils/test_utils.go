package testutils

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	ServerBinary = "../../bin/consent-server"
	ServerPort   = "9000"
)

var serverCmd *exec.Cmd

// BuildServer checks if the consent-server binary exists
// The binary should be built using ./build.sh build from the project root
func BuildServer() error {
	fmt.Println("Checking for consent server binary...")

	// Check if binary exists
	if _, err := os.Stat(ServerBinary); os.IsNotExist(err) {
		return fmt.Errorf("server binary not found at %s. Please run './build.sh build' from project root", ServerBinary)
	}

	fmt.Println("✓ Found server binary at", ServerBinary)
	return nil
}

// SetupDatabase runs database migration scripts
func SetupDatabase() error {
	fmt.Println("Setting up test database...")
	// For now, we assume the database is already set up
	// In production, this would run migration scripts
	return nil
}

// StartServer starts the consent-server in background
func StartServer() error {
	fmt.Println("Starting consent server...")
	cmd := exec.Command(ServerBinary)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables for test mode
	cmd.Env = append(os.Environ(),
		"SERVER_PORT="+ServerPort,
		"LOG_LEVEL=debug",
	)

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	serverCmd = cmd
	return nil
}

// StopServer gracefully stops the consent-server
func StopServer() error {
	if serverCmd == nil || serverCmd.Process == nil {
		return nil
	}

	fmt.Println("Stopping server...")

	// Send interrupt signal
	err := serverCmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	// Wait for process to exit
	_, err = serverCmd.Process.Wait()
	return err
}

// WaitForServer waits for the server to be ready
func WaitForServer() error {
	fmt.Println("Waiting for server to be ready...")
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get("http://localhost:" + ServerPort + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			fmt.Println("✓ Server is ready!")
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("server did not start within timeout")
}
