package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/afoninsky/mcp-yeelight/yeelight"
)

func main() {
	// Check if script name is provided
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <script_name> [interval_ms] [timeout_s]")
		fmt.Println("\nEnvironment variables:")
		fmt.Println("  YEELIGHT_ADDR   : Yeelight address (default: 192.168.1.118:55443)")
		fmt.Println("  SCRIPTS_PATH    : Path to scripts folder (default: ./scripts)")
		return
	}

	// Get environment variables
	yeelightAddr := os.Getenv("YEELIGHT_ADDR")
	if yeelightAddr == "" {
		panic("YEELIGHT_ADDR env is not set")
	}

	scriptsPath := os.Getenv("SCRIPTS_PATH")
	if scriptsPath == "" {
		scriptsPath = "./scripts"
	}

	// Build script filename
	scriptName := os.Args[1]
	// Remove .txt extension if provided
	scriptName = strings.TrimSuffix(scriptName, ".txt")
	// Build full path
	scriptPath := filepath.Join(scriptsPath, scriptName+".txt")
	
	// Default interval (milliseconds)
	interval := 500 * time.Millisecond
	if len(os.Args) > 2 {
		ms, err := time.ParseDuration(os.Args[2] + "ms")
		if err == nil {
			interval = ms
		}
	}

	// Default timeout (0 = infinite)
	var timeout time.Duration
	if len(os.Args) > 3 {
		s, err := time.ParseDuration(os.Args[3] + "s")
		if err == nil {
			timeout = s
		}
	}

	// Initialize Yeelight
	yl := yeelight.Yeelight{Address: yeelightAddr}
	
	// Create script runner
	runner := yeelight.NewScriptRunner(&yl)

	// Run the script
	fmt.Printf("Running script: %s (interval: %v, timeout: %v)\n", scriptName, interval, timeout)
	if err := runner.RunScript(scriptPath, interval, timeout); err != nil {
		log.Fatalf("Failed to run script: %v", err)
	}

	// Wait for user to press enter to stop
	if timeout == 0 {
		fmt.Println("Press Enter to stop the script...")
		fmt.Scanln()
		
		// Stop the script
		if err := runner.StopScript(); err != nil {
			log.Printf("Failed to stop script: %v", err)
		}
	} else {
		// Wait for timeout
		time.Sleep(timeout)
	}

	fmt.Println("Script finished.")
}