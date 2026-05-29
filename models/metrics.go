package models

import "time"

// CPUMetrics contains CPU usage and temperature information
type CPUMetrics struct {
	UsagePercent float64 `json:"usage_percent"`
	Temperature  float64 `json:"temperature_celsius"`
	CoreCount    int     `json:"core_count"`
}

// MemoryMetrics contains system memory statistics
type MemoryMetrics struct {
	Total       uint64  `json:"total_bytes"`
	Used        uint64  `json:"used_bytes"`
	Free        uint64  `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
}

// NetworkInterfaceMetrics contains network interface statistics
type NetworkInterfaceMetrics struct {
	Name        string            `json:"name"`
	IPAddresses []string          `json:"ip_addresses"`
	BytesSent   uint64            `json:"bytes_sent"`
	BytesRecv   uint64            `json:"bytes_received"`
	PacketsSent uint64            `json:"packets_sent"`
	PacketsRecv uint64            `json:"packets_received"`
	ErrorsIn    uint64            `json:"errors_in"`
	ErrorsOut   uint64            `json:"errors_out"`
	DropIn      uint64            `json:"drops_in"`
	DropOut     uint64            `json:"drops_out"`
	SendRate    float64           `json:"send_rate_bps"`    // Bytes per second
	ReceiveRate float64           `json:"receive_rate_bps"` // Bytes per second
	LastUpdate  time.Time         `json:"last_update"`
}

// PublicIPMetrics contains public IP and connectivity information
type PublicIPMetrics struct {
	IPAddress    string    `json:"ip_address"`
	Connected    bool      `json:"connected"`
	LastCheck    time.Time `json:"last_check"`
	ResponseTime int64     `json:"response_time_ms"`
	Error        string    `json:"error,omitempty"`
}

// DHCPLease represents a single DHCP lease entry
type DHCPLease struct {
	Expires    int64  `json:"expires"`
	MACAddress string `json:"mac_address"`
	IPAddress  string `json:"ip_address"`
	Hostname   string `json:"hostname"`
	ClientID   string `json:"client_id"`
	DeviceType string `json:"device_type"`
}

// DHCPMetrics contains all DHCP lease information
type DHCPMetrics struct {
	Leases      []DHCPLease `json:"leases"`
	TotalDevices int        `json:"total_devices"`
	LastUpdate  time.Time   `json:"last_update"`
}

// SystemMetrics combines all system metrics into a single response
type SystemMetrics struct {
	Timestamp        time.Time                 `json:"timestamp"`
	CPU              CPUMetrics                `json:"cpu"`
	Memory           MemoryMetrics             `json:"memory"`
	Network          []NetworkInterfaceMetrics `json:"network_interfaces"`
	PublicIP         *PublicIPMetrics          `json:"public_ip,omitempty"`
	DHCP             *DHCPMetrics              `json:"dhcp,omitempty"`
	CollectionErrors []string                  `json:"collection_errors,omitempty"`
}
