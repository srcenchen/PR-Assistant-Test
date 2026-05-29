package monitor

import (
	"op-status/models"
	"os"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/cpu"
)

// GetCPUMetrics returns current CPU metrics
func GetCPUMetrics() (models.CPUMetrics, error) {
	var metrics models.CPUMetrics

	// Get CPU usage percentage
	usage, err := cpu.Percent(0, false)
	if err != nil {
		return metrics, err
	}
	if len(usage) > 0 {
		metrics.UsagePercent = usage[0]
	}

	// Get CPU core count
	coreCount, err := cpu.Counts(true)
	if err != nil {
		// Fallback to physical core count if logical count fails
		coreCount, _ = cpu.Counts(false)
	}
	metrics.CoreCount = coreCount

	// Read CPU temperature from thermal zone
	tempData, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err == nil {
		tempStr := strings.TrimSpace(string(tempData))
		tempMilli, err := strconv.ParseInt(tempStr, 10, 64)
		if err == nil {
			metrics.Temperature = float64(tempMilli) / 1000.0
		}
	}

	return metrics, nil
}
