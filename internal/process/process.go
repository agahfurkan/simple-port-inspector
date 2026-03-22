package process

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// Kill sends a SIGTERM to the given PID.
func Kill(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}

	err = proc.Signal(syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("failed to kill process %d: %w", pid, err)
	}
	return nil
}

// ForceKill sends a SIGKILL to the given PID.
func ForceKill(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}

	err = proc.Signal(syscall.SIGKILL)
	if err != nil {
		return fmt.Errorf("failed to force kill process %d: %w", pid, err)
	}
	return nil
}

// IsRunning checks if a process is still alive.
func IsRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Sending signal 0 checks if the process exists
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// GetProcessPath returns the full path of the executable for a PID.
func GetProcessPath(pid int) string {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// GetProcessCPU returns the CPU usage percentage for a PID.
func GetProcessCPU(pid int) string {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "%cpu=")
	output, err := cmd.Output()
	if err != nil {
		return "?"
	}
	return strings.TrimSpace(string(output))
}

// GetProcessMem returns the memory usage percentage for a PID.
func GetProcessMem(pid int) string {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "%mem=")
	output, err := cmd.Output()
	if err != nil {
		return "?"
	}
	return strings.TrimSpace(string(output))
}

// GetProcessStartTime returns the start time of a PID.
func GetProcessStartTime(pid int) string {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "lstart=")
	output, err := cmd.Output()
	if err != nil {
		return "?"
	}
	return strings.TrimSpace(string(output))
}
