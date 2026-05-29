package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"op-status/config"
	"op-status/models"
	"time"

	"github.com/shirou/gopsutil/v4/net"
)

// NetworkHistoryEntry stores previous network metrics with timestamp for throughput calculation
type NetworkHistoryEntry struct {
	BytesSent   uint64
	BytesRecv   uint64
	LastUpdate  time.Time
}

// dailyBaseline stores the counter values at midnight for daily traffic calculation
type dailyBaseline struct {
	Date        string `json:"date"`
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	// Last known counter values, used to detect interface reset (counters going backwards)
	LastBytesSent   uint64 `json:"last_bytes_sent,omitempty"`
	LastBytesRecv   uint64 `json:"last_bytes_recv,omitempty"`
	LastPacketsSent uint64 `json:"last_packets_sent,omitempty"`
	LastPacketsRecv uint64 `json:"last_packets_recv,omitempty"`
	// Accumulated traffic from before the last counter reset
	AccBytesSent   uint64 `json:"acc_bytes_sent,omitempty"`
	AccBytesRecv   uint64 `json:"acc_bytes_recv,omitempty"`
	AccPacketsSent uint64 `json:"acc_packets_sent,omitempty"`
	AccPacketsRecv uint64 `json:"acc_packets_recv,omitempty"`
}

// dailyBaselineFile is the JSON structure for the persistent file
type dailyBaselineFile struct {
	Interfaces map[string]dailyBaseline `json:"interfaces"`
}

var (
	networkHistory = make(map[string][]NetworkHistoryEntry)
	networkMutex   sync.Mutex
	dailyBaselines = make(map[string]dailyBaseline)
	dailyMutex     sync.Mutex
)

func init() {
	loadDailyBaselines()
}

// loadDailyBaselines reads persisted baselines from file
func loadDailyBaselines() {
	data, err := os.ReadFile(config.DailyBaselineFile)
	if err != nil {
		return
	}
	var f dailyBaselineFile
	if err := json.Unmarshal(data, &f); err != nil {
		return
	}
	dailyMutex.Lock()
	defer dailyMutex.Unlock()
	for name, baseline := range f.Interfaces {
		dailyBaselines[name] = baseline
	}
}

// saveDailyBaselines writes current baselines to file atomically
func saveDailyBaselines() error {
	dailyMutex.Lock()
	interfaces := make(map[string]dailyBaseline, len(dailyBaselines))
	for k, v := range dailyBaselines {
		interfaces[k] = v
	}
	dailyMutex.Unlock()

	f := dailyBaselineFile{Interfaces: interfaces}
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}

	// Write to temp file first, then rename for atomicity
	tmpFile, err := os.CreateTemp(filepath.Dir(config.DailyBaselineFile), ".op-status-baseline-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	if err := os.Rename(tmpName, config.DailyBaselineFile); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("atomic rename failed: %w", err)
	}
	return nil
}

// getDailyTraffic calculates today's traffic by subtracting the midnight baseline from current counters.
// Returns the daily traffic values and whether the baseline was updated (needs persist).
func getDailyTraffic(ifaceName string, currentBytesSent, currentBytesRecv, currentPacketsSent, currentPacketsRecv uint64) (bytesSent, bytesRecv, packetsSent, packetsRecv uint64, needsSave bool) {
	today := time.Now().In(config.LocalLocation).Format("2006-01-02")

	dailyMutex.Lock()
	baseline, exists := dailyBaselines[ifaceName]

	// If no baseline for this interface, or the date changed (new day), reset baseline
	if !exists || baseline.Date != today {
		baseline = dailyBaseline{
			Date:            today,
			BytesSent:       currentBytesSent,
			BytesRecv:       currentBytesRecv,
			PacketsSent:     currentPacketsSent,
			PacketsRecv:     currentPacketsRecv,
			LastBytesSent:   currentBytesSent,
			LastBytesRecv:   currentBytesRecv,
			LastPacketsSent: currentPacketsSent,
			LastPacketsRecv: currentPacketsRecv,
		}
		dailyBaselines[ifaceName] = baseline
		dailyMutex.Unlock()
		needsSave = true
	} else {
		// Detect counter reset (interface went down/up, counters reset to 0)
		if currentBytesSent < baseline.LastBytesSent {
			baseline.AccBytesSent += baseline.LastBytesSent - baseline.BytesSent
			baseline.BytesSent = currentBytesSent
			needsSave = true
		}
		if currentBytesRecv < baseline.LastBytesRecv {
			baseline.AccBytesRecv += baseline.LastBytesRecv - baseline.BytesRecv
			baseline.BytesRecv = currentBytesRecv
			needsSave = true
		}
		if currentPacketsSent < baseline.LastPacketsSent {
			baseline.AccPacketsSent += baseline.LastPacketsSent - baseline.PacketsSent
			baseline.PacketsSent = currentPacketsSent
			needsSave = true
		}
		if currentPacketsRecv < baseline.LastPacketsRecv {
			baseline.AccPacketsRecv += baseline.LastPacketsRecv - baseline.PacketsRecv
			baseline.PacketsRecv = currentPacketsRecv
			needsSave = true
		}
		// Update last known counters
		baseline.LastBytesSent = currentBytesSent
		baseline.LastBytesRecv = currentBytesRecv
		baseline.LastPacketsSent = currentPacketsSent
		baseline.LastPacketsRecv = currentPacketsRecv
		dailyBaselines[ifaceName] = baseline
		dailyMutex.Unlock()
	}

	// Calculate daily traffic (accumulated + current - baseline)
	if currentBytesSent >= baseline.BytesSent {
		bytesSent = baseline.AccBytesSent + currentBytesSent - baseline.BytesSent
	} else {
		bytesSent = baseline.AccBytesSent
	}
	if currentBytesRecv >= baseline.BytesRecv {
		bytesRecv = baseline.AccBytesRecv + currentBytesRecv - baseline.BytesRecv
	} else {
		bytesRecv = baseline.AccBytesRecv
	}
	if currentPacketsSent >= baseline.PacketsSent {
		packetsSent = baseline.AccPacketsSent + currentPacketsSent - baseline.PacketsSent
	} else {
		packetsSent = baseline.AccPacketsSent
	}
	if currentPacketsRecv >= baseline.PacketsRecv {
		packetsRecv = baseline.AccPacketsRecv + currentPacketsRecv - baseline.PacketsRecv
	} else {
		packetsRecv = baseline.AccPacketsRecv
	}

	return
}

// GetNetworkMetrics returns metrics for all monitored network interfaces
func GetNetworkMetrics() ([]models.NetworkInterfaceMetrics, error) {
	var metrics []models.NetworkInterfaceMetrics
	var shouldSave bool

	// Get all network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// Get I/O counters for all interfaces
	ioCounters, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	// Create a map of I/O counters by interface name
	ioMap := make(map[string]net.IOCountersStat)
	for _, io := range ioCounters {
		ioMap[io.Name] = io
	}

	now := time.Now().In(config.LocalLocation)

	// Process each monitored interface
	for _, ifaceName := range config.MonitorInterfaces {
		// Find the interface
		var iface net.InterfaceStat
		found := false
		for _, i := range ifaces {
			if i.Name == ifaceName {
				iface = i
				found = true
				break
			}
		}
		if !found {
			continue
		}

		// Get I/O counters for this interface
		io, ok := ioMap[ifaceName]
		if !ok {
			continue
		}

		// Collect IP addresses
		var ips []string
		hasIPv4 := false
		for _, addr := range iface.Addrs {
			ipStr := addr.Addr
			ips = append(ips, ipStr)
			// Check if this is an IPv4 address
			if strings.Contains(ipStr, ".") {
				hasIPv4 = true
			}
		}

		// If interface has no IPv4 addresses (only IPv6), skip it (especially for wan interface)
		if !hasIPv4 {
			continue
		}

		// Calculate throughput
		var sendRate, receiveRate float64
		networkMutex.Lock()
		history, hasHistory := networkHistory[ifaceName]
		if hasHistory && len(history) > 0 {
			last := history[len(history)-1]
			timeDiff := now.Sub(last.LastUpdate).Seconds()
			if timeDiff > 0 {
				if io.BytesSent >= last.BytesSent {
					sendRate = float64(io.BytesSent-last.BytesSent) / timeDiff
				}
				if io.BytesRecv >= last.BytesRecv {
					receiveRate = float64(io.BytesRecv-last.BytesRecv) / timeDiff
				}
			}
		}

		// Update history
		historyEntry := NetworkHistoryEntry{
			BytesSent:  io.BytesSent,
			BytesRecv:  io.BytesRecv,
			LastUpdate: now,
		}
		networkHistory[ifaceName] = append(networkHistory[ifaceName], historyEntry)
		// Keep only the last N samples
		if len(networkHistory[ifaceName]) > config.MaxNetworkThroughputHistory {
			networkHistory[ifaceName] = networkHistory[ifaceName][1:]
		}
		networkMutex.Unlock()

		// Calculate daily traffic (since midnight)
		dailySent, dailyRecv, dailyPktSent, dailyPktRecv, needsSave := getDailyTraffic(ifaceName, io.BytesSent, io.BytesRecv, io.PacketsSent, io.PacketsRecv)
		if needsSave {
			shouldSave = true
		}

		// Create metrics entry (counters show today's traffic, not cumulative total)
		ifaceMetrics := models.NetworkInterfaceMetrics{
			Name:        ifaceName,
			IPAddresses: ips,
			BytesSent:   dailySent,
			BytesRecv:   dailyRecv,
			PacketsSent: dailyPktSent,
			PacketsRecv: dailyPktRecv,
			ErrorsIn:    io.Errin,
			ErrorsOut:   io.Errout,
			DropIn:      io.Dropin,
			DropOut:     io.Dropout,
			SendRate:    sendRate,
			ReceiveRate: receiveRate,
			LastUpdate:  now,
		}

		metrics = append(metrics, ifaceMetrics)
	}

	// Persist baselines to file once after processing all interfaces
	if shouldSave {
		saveDailyBaselines()
	}

	return metrics, nil
}
