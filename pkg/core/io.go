package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// RunMode controls how command output is handled
type RunMode int

const (
	// RunModeSilent captures output, logs to file only, no console output
	RunModeSilent RunMode = iota
	// RunModeLog logs command to logger, captures output to file only
	RunModeLog
	// RunModeStream streams output to console (via logger) and logs to file
	RunModeStream
)

// RunCommand executes a command with the specified output handling mode.
// mode: RunModeSilent (no console), RunModeLog (log command, silent output), RunModeStream (stream to console)
// args: command and its arguments (first element is the command)
// logger: logger for command logging and output formatting
// logPath: path to the log file for raw output capture (if empty, logging is skipped)
func RunCommand(mode RunMode, args []string, logger *Logger, logPath string) error {
	if len(args) == 0 {
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)

	// If no log path is provided, use os.DevNull for file operations
	var logFile *os.File
	if logPath != "" {
		var err error
		logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		defer logFile.Close()
	}

	switch mode {
	case RunModeSilent:
		// Silent: no console output, capture to log file (or discard if no log path)
		if logFile != nil {
			fmt.Fprintln(logFile, "--- Command Output Start ---")
			cmd.Stdout = logFile
			cmd.Stderr = logFile
		} else {
			devNull, _ := os.Open(os.DevNull)
			cmd.Stdout = devNull
			cmd.Stderr = devNull
			defer devNull.Close()
		}
		err := cmd.Run()
		if logFile != nil {
			fmt.Fprintln(logFile, "--- Command Output End ---")
			fmt.Fprintln(logFile)
		}
		if err != nil {
			return err
		}
		return nil

	case RunModeLog:
		// Log: log command to logger, capture to log file (or discard if no log path)
		if logger != nil {
			logger.Command(fmt.Sprintf("%s", args))
		}
		if logFile != nil {
			fmt.Fprintln(logFile, "--- Command Output Start ---")
			cmd.Stdout = logFile
			cmd.Stderr = logFile
		} else {
			devNull, _ := os.Open(os.DevNull)
			cmd.Stdout = devNull
			cmd.Stderr = devNull
			defer devNull.Close()
		}
		err := cmd.Run()
		if logFile != nil {
			fmt.Fprintln(logFile, "--- Command Output End ---")
			fmt.Fprintln(logFile)
		}
		if err != nil {
			return err
		}
		return nil

	case RunModeStream:
		// Stream: log output lines to console via logger, capture to log file (or discard if no log path)
		if logger != nil {
			logger.Command(fmt.Sprintf("%s", args))
		}
		if logFile != nil {
			fmt.Fprintln(logFile, "--- Command Output Start ---")
		}

		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf

		err := cmd.Run()

		output := buf.String()
		if output != "" {
			lines := splitLines(output, 125)
			for _, line := range lines {
				if logFile != nil {
					fmt.Fprintln(logFile, line)
				}
				// Also log to console
				fmt.Print(line + "\n")
			}
		}

		if logFile != nil {
			fmt.Fprintln(logFile, "--- Command Output End ---")
			fmt.Fprintln(logFile)
		}
		if err != nil {
			return err
		}
		return nil
	}

	return nil
}

// RunCommandWithOutput executes a command and returns its output.
// Useful for commands where output needs to be captured and processed.
func RunCommandWithOutput(args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// LookPath checks if a command is available in PATH
func LookPath(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

// cmdStreamLine formats a command output line with ANSI styling
func cmdStreamLine(line string, width int, pad int, bg string, fg string) string {
	if width <= 0 {
		return ""
	}

	if len(line) <= width {
		return fmt.Sprintf("%*s%-*.*s%*s", pad, "", width-pad, width-pad, line, pad, "")
	}

	// Truncate line
	max := width - pad - pad
	if max <= 3 {
		return fmt.Sprintf("%*s%s%*s", pad, "", line[:max], pad, "")
	}
	return fmt.Sprintf("%*s%-*.*s%*s", pad, "", max, max, line[:max-3]+"...", pad, "")
}

// runCommandStream executes a command and streams output line by line to both console and log file
// This mirrors the bash cmd_streamer function behavior
func runCommandStream(args []string, logger *Logger, logPath string) error {
	if len(args) == 0 {
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	fmt.Fprintln(logFile, "--- Command Output Start ---")

	// Channel to merge stdout and stderr
	outputChan := make(chan string)

	// Helper to read from a pipe and send to channel
	readPipe := func(pipe io.Reader, ch chan<- string) {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			ch <- scanner.Text()
		}
	}

	// Start goroutines to read stdout and stderr
	go readPipe(stdout, outputChan)
	go readPipe(stderr, outputChan)

	// Read and process output
	var cmdErr error
	for {
		line, ok := <-outputChan
		if !ok {
			// Both channels closed
			break
		}
		// Log to file
		fmt.Fprintln(logFile, line)
		// Log to console
		fmt.Println(line)
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		cmdErr = err
	}

	return cmdErr
}
