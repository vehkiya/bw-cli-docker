package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var (
	// Used to ensure the setup process only runs once.
	setupOnce sync.Once

	// Stores the session key after a successful unlock.
	sessionKey string

	// Indicates whether the initial setup was successful.
	isSetupSuccessful bool
)

func main() {
	// Check for required environment variables.
	if os.Getenv("BW_CLIENTID") == "" || os.Getenv("BW_CLIENTSECRET") == "" {
		fmt.Fprintln(os.Stderr, "Fatal: BW_CLIENTID and BW_CLIENTSECRET environment variables must be set.")
		os.Exit(1)
	}
	if os.Getenv("BW_PASSWORD") == "" {
		fmt.Fprintln(os.Stderr, "Fatal: BW_PASSWORD environment variable must be set.")
		os.Exit(1)
	}

	// Start the Bitwarden setup and serve process in the background.
	go setupAndServe()

	// Start an HTTP server to handle health checks.
	http.HandleFunc("/healthz", healthCheckHandler)
	fmt.Println("Starting health check server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: Health check server failed: %v", err)
		os.Exit(1)
	}
}

// setupAndServe runs the initial Bitwarden configuration and starts the 'bw serve' command.
func setupAndServe() {
	setupOnce.Do(func() {
		mustRun("Configuring server", "bw", "config", "server", os.Getenv("BW_HOST"))
		mustRun("Logging in with API Key", "bw", "login", "--apikey")

		fmt.Println("Unlocking vault to get session key...")
		unlockCmd := exec.Command("bw", "unlock", "--raw", "--passwordenv", "BW_PASSWORD")
		sessionKeyBytes, err := unlockCmd.Output()
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				fmt.Fprintln(os.Stderr, "Unlock failed:", string(exitError.Stderr))
			}
			return // Don't proceed if unlock fails.
		}
		sessionKey = strings.TrimSpace(string(sessionKeyBytes))
		os.Setenv("BW_SESSION", sessionKey)

		mustRun("Checking vault status", "bw", "unlock", "--check")

		// The initial setup is complete.
		isSetupSuccessful = true
		fmt.Println("Initial setup complete. Starting Bitwarden server...")

		// Start the Bitwarden server.
		cmd := exec.Command("bw", "serve", "--hostname", "0.0.0.0", "--port", "8087")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Fatal: 'bw serve' command failed: %v", err)
			isSetupSuccessful = false // Mark as unhealthy if the server crashes.
		}
	})
}

// healthCheckHandler responds to Kubernetes health probes.
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if !isSetupSuccessful {
		http.Error(w, "Setup not complete or failed", http.StatusServiceUnavailable)
		return
	}

	// Run 'bw unlock --check' to ensure the vault is accessible.
	cmd := exec.Command("bw", "unlock", "--check")
	cmd.Env = append(os.Environ(), "BW_SESSION="+sessionKey)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v", err)
		http.Error(w, "Health check failed", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

// mustRun executes a command and exits the program if it fails.
func mustRun(stepName string, commandName string, args ...string) {
	fmt.Printf("--> %s", stepName)
	cmd := exec.Command(commandName, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: Command failed during '%s': %v", stepName, err)
		os.Exit(1) // Exit here because these are critical setup steps.
	}
}
