package scanner

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// PortEntry represents a single port binding by a process.
type PortEntry struct {
	PID       int
	Command   string
	User      string
	Port      int
	Protocol  string // TCP or UDP
	State     string // LISTEN, ESTABLISHED, etc.
	LocalAddr string
	FD        string
	Type      string // IPv4, IPv6
}

// ProcessGroup groups all port entries belonging to the same process.
type ProcessGroup struct {
	PID     int
	Command string
	User    string
	Ports   []PortEntry
}

// ScanResult holds the full scan output.
type ScanResult struct {
	Entries  []PortEntry
	Groups   []ProcessGroup
	ScannedAt time.Time
}

// Scan runs lsof to discover all listening and connected network ports.
func Scan() (*ScanResult, error) {
	return ScanWithFilter("")
}

// ScanWithFilter runs lsof with an optional port filter.
func ScanWithFilter(portFilter string) (*ScanResult, error) {
	// lsof flags:
	// -i       = list internet connections
	// -n       = no DNS resolution (faster)
	// -P       = no port name resolution (show numbers)
	args := []string{"-i", "-n", "-P"}
	if portFilter != "" {
		args = []string{"-i", ":" + portFilter, "-n", "-P"}
	}

	cmd := exec.Command("lsof", args...)
	output, err := cmd.Output()
	if err != nil {
		// lsof returns exit code 1 when no results found
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return &ScanResult{ScannedAt: time.Now()}, nil
			}
		}
		return nil, fmt.Errorf("failed to run lsof: %w", err)
	}

	entries := parseLsofOutput(string(output))

	// Deduplicate entries that differ only by IPv4/IPv6 (same PID+Port+Protocol+State)
	entries = deduplicateEntries(entries)

	// Sort by port number
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Port != entries[j].Port {
			return entries[i].Port < entries[j].Port
		}
		return entries[i].PID < entries[j].PID
	})

	groups := groupByProcess(entries)

	return &ScanResult{
		Entries:   entries,
		Groups:    groups,
		ScannedAt: time.Now(),
	}, nil
}

var portRegex = regexp.MustCompile(`[:\*](\d+)$`)

func parseLsofOutput(output string) []PortEntry {
	var entries []PortEntry
	scanner := bufio.NewScanner(strings.NewReader(output))

	// Skip header line
	if scanner.Scan() {
		// header consumed
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		command := fields[0]
		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		user := fields[2]
		fd := fields[3]
		connType := fields[4] // IPv4, IPv6

		// fields[7] is the NAME field for network connections
		// but field positions can shift; the protocol is usually in field 7
		// and name is the last field(s)
		proto := ""
		state := ""
		name := ""

		// Find the protocol field (TCP/UDP)
		protoIdx := -1
		for i := 4; i < len(fields); i++ {
			upper := strings.ToUpper(fields[i])
			if upper == "TCP" || upper == "UDP" {
				proto = upper
				protoIdx = i
				break
			}
		}

		if protoIdx == -1 {
			continue
		}

		// Name is the field after protocol
		if protoIdx+1 < len(fields) {
			name = fields[protoIdx+1]
		}

		// State is the last field if it looks like a state (in parens)
		lastField := fields[len(fields)-1]
		if strings.HasPrefix(lastField, "(") && strings.HasSuffix(lastField, ")") {
			state = strings.Trim(lastField, "()")
		}

		// Extract port from the local address part
		// Format: host:port or *:port or [::1]:port->remote:port
		localPart := name
		if idx := strings.Index(name, "->"); idx != -1 {
			localPart = name[:idx]
		}

		matches := portRegex.FindStringSubmatch(localPart)
		if matches == nil {
			continue
		}

		port, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		entry := PortEntry{
			PID:       pid,
			Command:   command,
			User:      user,
			Port:      port,
			Protocol:  proto,
			State:     state,
			LocalAddr: localPart,
			FD:        fd,
			Type:      connType,
		}

		entries = append(entries, entry)
	}

	return entries
}

// deduplicateEntries collapses entries that share the same PID, Port, Protocol,
// and State but differ only in address family (IPv4 vs IPv6). When both exist
// we keep the IPv4 entry and annotate the Type field as "IPv4+IPv6".
func deduplicateEntries(entries []PortEntry) []PortEntry {
	type dedupKey struct {
		PID      int
		Port     int
		Protocol string
		State    string
	}

	seen := make(map[dedupKey]int) // key -> index in result slice
	var result []PortEntry

	for _, e := range entries {
		k := dedupKey{PID: e.PID, Port: e.Port, Protocol: e.Protocol, State: e.State}
		if idx, exists := seen[k]; exists {
			// Already have an entry for this key — merge type info
			existing := &result[idx]
			if existing.Type != e.Type {
				existing.Type = "IPv4+IPv6"
			}
			// Prefer the more specific local address (not wildcard)
			if strings.HasPrefix(existing.LocalAddr, "*") && !strings.HasPrefix(e.LocalAddr, "*") {
				existing.LocalAddr = e.LocalAddr
			}
		} else {
			seen[k] = len(result)
			result = append(result, e)
		}
	}

	return result
}

// systemCommands is a set of known macOS system process names that use network
// ports but are generally not interesting to application developers.
var systemCommands = map[string]bool{
	"ControlCe":  true, // Control Center
	"rapportd":   true, // Rapport daemon (AirPlay/Handoff)
	"sharingd":   true, // Sharing daemon (AirDrop, Handoff)
	"identitys":  true, // Identity Services
	"remotepai":  true, // Remote Pairing
	"WiFiAgent":  true, // WiFi agent
	"airportd":   true, // Airport daemon
	"configd":    true, // System Configuration
	"mDNSRespo":  true, // mDNS Responder
	"netbiosd":   true, // NetBIOS daemon
	"SystemUIS":  true, // SystemUIServer
	"UserEvent":  true, // UserEventAgent
	"symptomsd":  true, // Symptoms daemon
	"apsd":       true, // Apple Push Service
	"CommCenter":  true, // CommCenter
	"cloudd":     true, // iCloud daemon
	"nsurlsessi": true, // NSURLSession
	"trustd":     true, // Certificate trust daemon
	"locationd":  true, // Location daemon
	"bluetoothd": true, // Bluetooth daemon
	"WirelessPr": true, // Wireless Proximity
	"timed":      true, // Time daemon
	"parsecd":    true, // Parsec daemon
	"IMDPersist":  true, // IMD Persistence
	"AMPDeviceD": true, // AMP Device Discovery
	"loginwindo": true, // loginwindow
	"dasd":       true, // Duet Activity Scheduler
	"coreautha":  true, // CoreAuth agent
	"biometrick": true, // Biometric Kit
	"WindowServ": true, // WindowServer
	"EEventMan":  true, // Enterprise Event Manager
}

// IsSystemProcess returns true if the process is a known macOS system process.
func IsSystemProcess(command string) bool {
	if systemCommands[command] {
		return true
	}
	return false
}

func groupByProcess(entries []PortEntry) []ProcessGroup {
	groupMap := make(map[int]*ProcessGroup)

	for _, e := range entries {
		g, exists := groupMap[e.PID]
		if !exists {
			g = &ProcessGroup{
				PID:     e.PID,
				Command: e.Command,
				User:    e.User,
			}
			groupMap[e.PID] = g
		}
		g.Ports = append(g.Ports, e)
	}

	var groups []ProcessGroup
	for _, g := range groupMap {
		groups = append(groups, *g)
	}

	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].Ports) > 0 && len(groups[j].Ports) > 0 {
			return groups[i].Ports[0].Port < groups[j].Ports[0].Port
		}
		return groups[i].PID < groups[j].PID
	})

	return groups
}
