package monitor

import (
	"op-status/models"

	"github.com/shirou/gopsutil/v4/mem"
)

// GetMemoryMetrics returns current memory metrics
func GetMemoryMetrics() (models.MemoryMetrics, error) {
	var metrics models.MemoryMetrics

	vmem, err := mem.VirtualMemory()
	if err != nil {
		return metrics, err
	}

	metrics.Total = vmem.Total
	metrics.Used = vmem.Used
	metrics.Free = vmem.Free
	metrics.UsedPercent = vmem.UsedPercent

	return metrics, nil
}
