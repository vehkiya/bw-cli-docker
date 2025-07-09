package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func main() {
	// Check if debug mode is enabled.
	debugEnabled := os.Getenv("DEBUG_ENABLED") != ""

	if os.Getenv("BW_CLIENTID") == "" || os.Getenv("BW_CLIENTSECRET") == "" {
		fmt.Fprintln(os.Stderr, "Fatal: BW_CLIENTID and BW_CLIENTSECRET environment variables must be set.")
		os.Exit(1)
	}
	if os.Getenv("BW_PASSWORD") == "" {
		fmt.Fprintln(os.Stderr, "Fatal: BW_PASSWORD environment variable must be set.")
		os.Exit(1)
	}

	mustRun("Configuring server", "bw", "config", "server", os.Getenv("BW_HOST"))
	mustRun("Logging in with API Key", "bw", "login", "--apikey")

	fmt.Println("Unlocking vault to get session key...")
	unlockCmd := exec.Command("bw", "unlock", "--raw", "--passwordenv", "BW_PASSWORD")
	sessionKeyBytes, err := unlockCmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			fmt.Fprintln(os.Stderr, "Unlock failed:", string(exitError.Stderr))
		}
		os.Exit(1)
	}
	sessionKey := strings.TrimSpace(string(sessionKeyBytes))

	// If debug mode is enabled, print the session key.
	if debugEnabled {
		fmt.Printf("DEBUG: Session key obtained: %s\n", sessionKey)
	}

	os.Setenv("BW_SESSION", sessionKey)
	mustRun("Checking vault status", "bw", "unlock", "--check")

	fmt.Println("Starting Bitwarden server on port 8087...")
	bwPath, err := exec.LookPath("bw")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fatal: Could not find 'bw' executable in PATH")
		os.Exit(1)
	}

	args := []string{"bw", "serve", "--hostname", "0.0.0.0", "--port", "8087"}
	if err := syscall.Exec(bwPath, args, os.Environ()); err != nil {
		fmt.Fprintln(os.Stderr, "Fatal: Failed to exec 'bw serve':", err)
	}
}

func mustRun(stepName string, commandName string, args ...string) {
	fmt.Printf("--> %s\n", stepName)
	cmd := exec.Command(commandName, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: Command failed during '%s': %v\n", stepName, err)
		os.Exit(1)
	}
}
