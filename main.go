package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/afoninsky/yeelight/yeelight"
)

var (
	// Global script runner for HTTP mode
	globalRunner *yeelight.ScriptRunner
	// Global Yeelight instance
	globalYeelight *yeelight.Yeelight
	// Scripts path
	scriptsPath string
)

func main() {
	// Parse command line flags
	httpMode := flag.Bool("http", false, "Run in HTTP server mode")
	flag.Parse()

	// Get environment variables
	yeelightAddr := os.Getenv("YEELIGHT_ADDR")
	if yeelightAddr == "" {
		log.Fatal("YEELIGHT_ADDR env is not set")
	}

	httpAddr := os.Getenv("YEELIGHT_HTTP")
	if httpAddr == "" {
		httpAddr = ":3048"
	}

	scriptsPath = os.Getenv("YEELIGHT_SCRIPTS")
	if scriptsPath == "" {
		scriptsPath = "./scripts"
	}

	// Initialize Yeelight
	globalYeelight = &yeelight.Yeelight{Address: yeelightAddr}
	globalRunner = yeelight.NewScriptRunner(globalYeelight)

	// Decide which mode to run
	if *httpMode || os.Getenv("YEELIGHT_HTTP") != "" {
		// Run in HTTP server mode
		runHTTPServer(httpAddr)
	} else {
		// Run in CLI mode
		runCLIMode()
	}
}

func runHTTPServer(addr string) {
	// Set up HTTP routes
	http.HandleFunc("/yeelight", handleListScripts)
	http.HandleFunc("/yeelight/", handleScriptActions)

	// Create server
	srv := &http.Server{
		Addr:         addr,
		Handler:      http.DefaultServeMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Printf("Starting HTTP server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down server...")

	// Stop any running script
	if err := globalRunner.StopScript(); err != nil {
		log.Printf("Failed to stop script during shutdown: %v", err)
	}

	// Shutdown server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func handleListScripts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read scripts directory
	files, err := ioutil.ReadDir(scriptsPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read scripts directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Build list of script names
	var scripts []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
			scriptName := strings.TrimSuffix(file.Name(), ".txt")
			scripts = append(scripts, scriptName)
		}
	}

	// Return plain text list
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, strings.Join(scripts, "\n"))
}

func handleScriptActions(w http.ResponseWriter, r *http.Request) {
	// Extract script name and action from URL
	path := strings.TrimPrefix(r.URL.Path, "/yeelight/")
	parts := strings.Split(path, "/")
	
	if len(parts) < 2 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	scriptName := parts[0]
	action := parts[1]

	switch action {
	case "run":
		handleRunScript(w, r, scriptName)
	case "stop":
		handleStopScript(w, r, scriptName)
	default:
		http.Error(w, "Unknown action", http.StatusNotFound)
	}
}

func handleRunScript(w http.ResponseWriter, r *http.Request, scriptName string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	intervalMs := 500
	timeoutSec := 0

	if intervalStr := r.URL.Query().Get("interval"); intervalStr != "" {
		if val, err := strconv.Atoi(intervalStr); err == nil && val > 0 {
			intervalMs = val
		}
	}

	if timeoutStr := r.URL.Query().Get("timeout"); timeoutStr != "" {
		if val, err := strconv.Atoi(timeoutStr); err == nil && val >= 0 {
			timeoutSec = val
		}
	}

	// Build script path
	scriptPath := filepath.Join(scriptsPath, scriptName+".txt")

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("Script not found: %s", scriptName), http.StatusNotFound)
		return
	}

	// Stop any currently running script
	globalRunner.StopScript()

	// Run the new script
	interval := time.Duration(intervalMs) * time.Millisecond
	timeout := time.Duration(timeoutSec) * time.Second

	if err := globalRunner.RunScript(scriptPath, interval, timeout); err != nil {
		http.Error(w, fmt.Sprintf("Failed to run script: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Script %s started (interval: %dms, timeout: %ds)\n", scriptName, intervalMs, timeoutSec)
}

func handleStopScript(w http.ResponseWriter, r *http.Request, scriptName string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Stop the script
	if err := globalRunner.StopScript(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to stop script: %v", err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Script %s stopped\n", scriptName)
}

func runCLIMode() {
	args := flag.Args()
	
	// Check if script name is provided
	if len(args) < 1 {
		fmt.Println("Usage: go run main.go [options] <script_name> [interval_ms] [timeout_s]")
		fmt.Println("\nOptions:")
		fmt.Println("  -http              Run in HTTP server mode")
		fmt.Println("\nEnvironment variables:")
		fmt.Println("  YEELIGHT_ADDR    : Yeelight address (required)")
		fmt.Println("  YEELIGHT_HTTP    : HTTP server address (default: :3048)")
		fmt.Println("  YEELIGHT_SCRIPTS     : Path to scripts folder (default: ./scripts)")
		fmt.Println("\nNote: If YEELIGHT_HTTP is set, the program will automatically start in HTTP mode")
		return
	}

	// Build script filename
	scriptName := args[0]
	// Remove .txt extension if provided
	scriptName = strings.TrimSuffix(scriptName, ".txt")
	// Build full path
	scriptPath := filepath.Join(scriptsPath, scriptName+".txt")
	
	// Default interval (milliseconds)
	interval := 500 * time.Millisecond
	if len(args) > 1 {
		ms, err := time.ParseDuration(args[1] + "ms")
		if err == nil {
			interval = ms
		}
	}

	// Default timeout (0 = infinite)
	var timeout time.Duration
	if len(args) > 2 {
		s, err := time.ParseDuration(args[2] + "s")
		if err == nil {
			timeout = s
		}
	}

	// Run the script
	fmt.Printf("Running script: %s (interval: %v, timeout: %v)\n", scriptName, interval, timeout)
	if err := globalRunner.RunScript(scriptPath, interval, timeout); err != nil {
		log.Fatalf("Failed to run script: %v", err)
	}

	// Wait for user to press enter to stop
	if timeout == 0 {
		fmt.Println("Press Enter to stop the script...")
		fmt.Scanln()
		
		// Stop the script
		if err := globalRunner.StopScript(); err != nil {
			log.Printf("Failed to stop script: %v", err)
		}
	} else {
		// Wait for timeout
		time.Sleep(timeout)
	}

	fmt.Println("Script finished.")
}