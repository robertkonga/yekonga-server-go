package yekonga

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/kardianos/service"
)

// Global loggers for clarity.
var statusLog *log.Logger
var programLog *log.Logger

type ServiceCallback func()
type ServiceConfig struct {
	Name        string
	DisplayName string
	Description string
}
type YekongaService struct {
	runServiceCallback ServiceCallback
	exitChan           chan struct{}
}

func ServiceSetup(config ServiceConfig, callback ServiceCallback) {
	logFile, err := setupLogging()
	if err != nil {
		log.Fatalf("Fatal Error: Could not set up logging: %v", err)
	}
	defer logFile.Close()

	svcConfig := &service.Config{
		Name:        config.Name,
		DisplayName: config.DisplayName,
		Description: config.Description,
	}

	prg := &YekongaService{
		runServiceCallback: callback,
	}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		statusLog.Fatal(err)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			err = s.Install()
			if err != nil {
				statusLog.Fatalf("Failed to install service: %v", err)
			}
			statusLog.Println("Service installed successfully")
			return
		case "uninstall":
			err = s.Uninstall()
			if err != nil {
				statusLog.Fatalf("Failed to uninstall service: %v", err)
			}

			statusLog.Println("Service uninstalled successfully")
			return
		case "status":
			applicationStatus(fmt.Sprintf("%v", svcConfig.Name))
			return
		case "log":
			applicationLiveLogs(fmt.Sprintf("%v.service", svcConfig.Name))
			return
		case "start":
			err = s.Start()
			if err != nil {
				statusLog.Fatalf("Failed to start service: %v", err)
			}
			statusLog.Println("Service started successfully")
			return
		case "stop":
			err = s.Stop()
			if err != nil {
				statusLog.Fatalf("Failed to stop service: %v", err)
			}
			statusLog.Println("Service stopped successfully")
			return
		}
	}

	err = s.Run()
	if err != nil {
		statusLog.Fatal(err)
	}
}

func (p *YekongaService) Start(s service.Service) error {
	statusLog.Println("Service starting...")
	p.exitChan = make(chan struct{})

	go p.run()
	return nil
}

func (p *YekongaService) Stop(s service.Service) error {
	statusLog.Println("Service stopping...")
	// Signal the run goroutine to exit
	close(p.exitChan)
	statusLog.Println("Service stopped.")
	return nil
}

func (p *YekongaService) run() {

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	p.runServiceCallback()

	for {
		select {
		case <-ticker.C:
			// This is our recurring task (Program Log)
			programLog.Printf("Heartbeat: The service is currently operational. Current time: %s", time.Now().Format(time.RFC3339))

		case <-p.exitChan:
			statusLog.Println("Received exit signal. Shutting down main loop.")
			return
		}
	}

}

// setupLogging initializes the log file and the custom loggers.
func setupLogging() (*os.File, error) {
	// Create a log file that captures all output.
	f, err := os.OpenFile("service_log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening log file: %v", err)
	}

	// Set the global log output to the file for compatibility with 'service.Run()' and fatal errors.
	log.SetOutput(f)

	// Initialize custom loggers with specific prefixes for status and program events.
	// We use the file as the output writer for both.
	statusLog = log.New(f, "[STATUS] ", log.Ldate|log.Ltime|log.Lshortfile)
	programLog = log.New(f, "[PROGRAM] ", log.Ldate|log.Ltime)

	return f, nil
}

func applicationLiveLogs(name string) {
	// The command and arguments are passed separately to exec.Command
	// Note: The program running this Go code must have sudo rights
	// (e.g., run the Go binary with `sudo`) or be configured for passwordless sudo
	cmd := exec.Command("clear", "&&", "sudo", "journalctl", "-u", name, "-f")

	// 1. Create a pipe to read the standard output of the command
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error creating StdoutPipe: %v", err)
	}

	// 2. Start the command
	if err := cmd.Start(); err != nil {
		log.Fatalf("Error starting command: %v", err)
	}

	fmt.Printf("--- Starting live log stream for %v --- \n", name)
	fmt.Println("   (Press Ctrl+C to stop the Go program and the command)\n")

	// 3. Use a bufio.Scanner in a goroutine to read the output line by line
	// This is crucial for handling the continuous stream from 'journalctl -f'
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		// Display the line as soon as it is read
		fmt.Println(scanner.Text())
	}

	// Check for scanner errors (e.g., if the pipe closed unexpectedly)
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading stdout: %v", err)
	}

	// 4. Wait for the command to finish (it won't normally for journalctl -f
	// unless the Go program is killed with Ctrl+C, which also kills the child process).
	if err := cmd.Wait(); err != nil {
		// When killed by Ctrl+C, the error is usually "signal: interrupt"
		// or "exit status 1", which is expected behavior.
		log.Printf("Command finished with error: %v", err)
	}
}

func applicationStatus(name string) {
	// The command and arguments are passed separately to exec.Command
	// Note: The program running this Go code must have sudo rights
	// (e.g., run the Go binary with `sudo`) or be configured for passwordless sudo
	cmd := exec.Command("systemctl", "status", name)

	// 1. Create a pipe to read the standard output of the command
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error creating StdoutPipe: %v", err)
	}

	// 2. Start the command
	if err := cmd.Start(); err != nil {
		log.Fatalf("Error starting command: %v", err)
	}

	// 3. Use a bufio.Scanner in a goroutine to read the output line by line
	// This is crucial for handling the continuous stream from 'journalctl -f'
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		// Display the line as soon as it is read
		fmt.Println(scanner.Text())
	}

	// Check for scanner errors (e.g., if the pipe closed unexpectedly)
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading stdout: %v", err)
	}

	// 4. Wait for the command to finish (it won't normally for journalctl -f
	// unless the Go program is killed with Ctrl+C, which also kills the child process).
	if err := cmd.Wait(); err != nil {
		// When killed by Ctrl+C, the error is usually "signal: interrupt"
		// or "exit status 1", which is expected behavior.
		log.Printf("Command finished with error: %v", err)
	}
}
