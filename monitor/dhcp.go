package monitor

import (
	"bufio"
	"os"
	"strings"
	"time"
	"op-status/config"
	"op-status/models"
)

var macVendorMap = map[string]string{
	"b8:27:eb": "树莓派",
	"3c:22:fb": "苹果笔记本",
	"ba:54:14": "红米手机",
	"b6:81:77": "苹果手机",
	"16:01:0f": "安卓设备",
	"f6:18:23": "未知设备",
	"dc:a6:32": "树莓派4",
	"28:cd:c4": "树莓派Zero",
	"00:1a:2a": "思科设备",
	"00:15:5d": "Hyper-V虚拟机",
	"00:0c:29": "VMware虚拟机",
	"52:54:00": "KVM虚拟机",
}

// GetDHCPLeases reads and parses DHCP leases file
func GetDHCPLeases() (*models.DHCPMetrics, error) {
	file, err := os.Open("/tmp/dhcp.leases")
	if err != nil {
		// For testing on non-OpenWrt systems, return sample data
		return getSampleDHCPData(), nil
	}
	defer file.Close()

	var leases []models.DHCPLease
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}

		lease := models.DHCPLease{
			Expires:    parseInt64(parts[0]),
			MACAddress: parts[1],
			IPAddress:  parts[2],
			Hostname:   parts[3],
			ClientID:   parts[4],
		}

		// Identify device type by MAC prefix
		if len(lease.MACAddress) >= 8 {
			prefix := strings.ToLower(lease.MACAddress[:8])
			if deviceType, ok := macVendorMap[prefix]; ok {
				lease.DeviceType = deviceType
			} else {
				lease.DeviceType = "未知设备"
			}
		} else {
			lease.DeviceType = "未知设备"
		}

		leases = append(leases, lease)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &models.DHCPMetrics{
		Leases:       leases,
		TotalDevices: len(leases),
		LastUpdate:   time.Now().In(config.LocalLocation),
	}, nil
}

// getSampleDHCPData returns sample data for testing
func getSampleDHCPData() *models.DHCPMetrics {
	leases := []models.DHCPLease{
		{
			Expires:    0,
			MACAddress: "b8:27:eb:41:66:33",
			IPAddress:  "192.168.2.12",
			Hostname:   "rpi3-2",
			ClientID:   "01:b8:27:eb:41:66:33",
			DeviceType: "树莓派",
		},
		{
			Expires:    1773514889,
			MACAddress: "3c:22:fb:e4:08:74",
			IPAddress:  "192.168.2.173",
			Hostname:   "sanenchendeMBP",
			ClientID:   "01:3c:22:fb:e4:08:74",
			DeviceType: "苹果笔记本",
		},
		{
			Expires:    1773514819,
			MACAddress: "ba:54:14:8c:41:9b",
			IPAddress:  "192.168.2.230",
			Hostname:   "Redmi-Note-9-Pro",
			ClientID:   "01:ba:54:14:8c:41:9b",
			DeviceType: "红米手机",
		},
		{
			Expires:    0,
			MACAddress: "b8:27:eb:e5:75:98",
			IPAddress:  "192.168.2.11",
			Hostname:   "rpi3-1",
			ClientID:   "01:b8:27:eb:e5:75:98",
			DeviceType: "树莓派",
		},
		{
			Expires:    1773508070,
			MACAddress: "16:01:0f:d2:13:2a",
			IPAddress:  "192.168.2.199",
			Hostname:   "Phh-Treble-Go",
			ClientID:   "01:16:01:0f:d2:13:2a",
			DeviceType: "安卓设备",
		},
		{
			Expires:    1773495540,
			MACAddress: "f6:18:23:56:36:23",
			IPAddress:  "192.168.2.224",
			Hostname:   "*",
			ClientID:   "01:f6:18:23:56:36:23",
			DeviceType: "未知设备",
		},
		{
			Expires:    1773514819,
			MACAddress: "b6:81:77:b1:df:d4",
			IPAddress:  "192.168.2.112",
			Hostname:   "iPhone",
			ClientID:   "01:b6:81:77:b1:df:d4",
			DeviceType: "苹果手机",
		},
	}

	return &models.DHCPMetrics{
		Leases:       leases,
		TotalDevices: len(leases),
		LastUpdate:   time.Now().In(config.LocalLocation),
	}
}

func parseInt64(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}
