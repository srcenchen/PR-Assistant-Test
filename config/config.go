package config

import (
	"log"
	"time"
)

// LocalLocation is the timezone used for all date/time calculations.
// On OpenWrt (musl libc), Go may not find the system timezone database,
// so we load it explicitly on startup.
var LocalLocation *time.Location

func init() {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Printf("Warning: failed to load timezone Asia/Shanghai, using fixed UTC+8: %v", err)
		LocalLocation = time.FixedZone("CST", 8*3600)
	} else {
		LocalLocation = loc
	}
}

const (
	// APIPort is the port on which the API server listens
	APIPort = ":8080"

	// PublicIPCheckURL is the URL used to check public IP address
	PublicIPCheckURL = "https://ip.me"
	// Alternative public IP check URLs if the primary fails
	// PublicIPCheckURL = "https://ifconfig.me"
	// PublicIPCheckURL = "https://icanhazip.com"

	// PublicIPTimeout is the timeout for public IP check requests
	PublicIPTimeout = 5 * time.Second

	// PublicIPCheckMaxWait is the max time GetPublicIPMetrics will wait for
	// an async probe to finish before falling back to the last cached result.
	PublicIPCheckMaxWait = 2 * time.Second

	// MaxNetworkThroughputHistory is the number of samples to keep for throughput calculation
	MaxNetworkThroughputHistory = 2

	// DailyBaselineFile is the file path for persisting daily traffic baselines
	DailyBaselineFile = "/tmp/op-status-daily-traffic.json"
)

// MonitorInterfaces is the list of network interfaces to monitor
var MonitorInterfaces = []string{"wan", "br-lan", "eth0"}
