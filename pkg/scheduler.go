package pkg

import (
	"op-status/config"
	"op-status/models"
	"op-status/monitor"
	"time"
)

// Scheduler manages metrics collection
type Scheduler struct {
}

// NewScheduler creates a new metrics collection scheduler
func NewScheduler() *Scheduler {
	// Pre-initialize CPU percent sampling to avoid first request blocking
	monitor.GetCPUMetrics()
	return &Scheduler{}
}

// Start is no longer needed as metrics are collected on demand
func (s *Scheduler) Start() {
	// No background collection needed anymore
}

// Stop is no longer needed
func (s *Scheduler) Stop() {
	// Nothing to stop
}

// GetMetrics collects and returns real-time metrics
// CPU, Memory, Network are collected on every request
// Public IP uses its own caching logic
func (s *Scheduler) GetMetrics() models.SystemMetrics {
	var metrics models.SystemMetrics
	var errors []string
	metrics.Timestamp = time.Now().In(config.LocalLocation)

	// Collect CPU metrics in real-time
	cpuMetrics, err := monitor.GetCPUMetrics()
	if err == nil {
		metrics.CPU = cpuMetrics
	} else {
		errors = append(errors, "cpu: "+err.Error())
	}

	// Collect memory metrics in real-time
	memMetrics, err := monitor.GetMemoryMetrics()
	if err == nil {
		metrics.Memory = memMetrics
	} else {
		errors = append(errors, "memory: "+err.Error())
	}

	// Collect network metrics in real-time (network module has internal fine-grained lock)
	netMetrics, err := monitor.GetNetworkMetrics()
	if err == nil {
		metrics.Network = netMetrics
	} else {
		errors = append(errors, "network: "+err.Error())
	}

	// Public IP uses its own built-in caching logic
	publicIP := monitor.GetPublicIPMetrics()
	if publicIP.Connected {
		// Only include public_ip field when connected (has valid IPv4)
		metrics.PublicIP = &publicIP
	} else {
		// No valid IPv4, omit the field in response
		metrics.PublicIP = nil
	}

	// Collect DHCP lease metrics
	dhcpMetrics, err := monitor.GetDHCPLeases()
	if err == nil {
		metrics.DHCP = dhcpMetrics
	} else {
		errors = append(errors, "dhcp: "+err.Error())
	}

	if len(errors) > 0 {
		metrics.CollectionErrors = errors
	}

	return metrics
}

// GetDHCPMetrics returns only DHCP lease information
func (s *Scheduler) GetDHCPMetrics() *models.DHCPMetrics {
	dhcpMetrics, err := monitor.GetDHCPLeases()
	if err != nil {
		return nil
	}
	return dhcpMetrics
}

// RefreshPublicIP forces an immediate refresh of public IP
func (s *Scheduler) RefreshPublicIP() models.PublicIPMetrics {
	return monitor.RefreshPublicIP()
}
