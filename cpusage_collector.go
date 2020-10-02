//go:generate mockgen -destination mock_cpusage_collector.go -source cpusage_collector.go -package cpumassager
package cpumassager

import (
	"fmt"
	"os"
)

// CPUsageCollector 收集CPU使用率的接口
type CPUsageCollector interface {
	GetCPUsage() float64
}

// linuxCPUData linux系统下的CPU使用率相关数据
type linuxCPUData struct {
	user      uint64
	nice      uint64
	system    uint64
	idle      uint64
	iowait    uint64
	irq       uint64
	softirq   uint64
	steal     uint64
	guest     uint64
	guestnice uint64
	total     uint64
}

// getCurCPUData 获取当前的linuxCPUData
func getCurLinuxCPUData() (*linuxCPUData, error) {
	d := &linuxCPUData{}
	statFile, err := os.Open("/proc/stat")
	if err != nil {
		return nil, fmt.Errorf("open /proc/stat error:%s", err.Error())
	}
	fmt.Fscanf(statFile, "cpu%d%d%d%d%d%d%d%d%d%d",
		&d.user, &d.nice, &d.system, &d.idle, &d.iowait,
		&d.irq, &d.softirq, &d.steal, &d.guest, &d.guestnice)
	d.total = d.user + d.nice + d.system + d.idle + d.iowait +
		d.irq + d.softirq + d.steal + d.guest + d.guestnice
	return d, nil
}

// linuxCPUsageCollector linux系统的CPU使用率收集器
type linuxCPUsageCollector struct {
	lastCPUData *linuxCPUData
	curCPUData  *linuxCPUData
}

// GetCPUsage 获取CPU使用率，如果获取失败，返回负数
func (c *linuxCPUsageCollector) GetCPUsage() float64 {
	curLinuxCPUData, err := getCurLinuxCPUData()
	if err != nil {
		return -1
	}
	c.curCPUData = curLinuxCPUData
	if c.lastCPUData == nil {
		c.lastCPUData = c.curCPUData
		return 0
	}

	userPeriod := (c.curCPUData.user - c.curCPUData.guest) -
		(c.lastCPUData.user - c.lastCPUData.guest)
	nicePeriod := (c.curCPUData.nice - c.curCPUData.guestnice) -
		(c.lastCPUData.nice - c.lastCPUData.guestnice)
	systemPeriod := (c.curCPUData.system + c.curCPUData.irq + c.curCPUData.softirq) -
		(c.lastCPUData.system + c.lastCPUData.irq + c.lastCPUData.softirq)
	stealPeriod := c.curCPUData.steal - c.lastCPUData.steal
	guestPeriod := (c.curCPUData.guest + c.curCPUData.guestnice) -
		(c.lastCPUData.guest + c.lastCPUData.guestnice)
	usedPeriod := userPeriod + nicePeriod + systemPeriod + stealPeriod + guestPeriod
	totalPeriod := c.curCPUData.total - c.lastCPUData.total
	c.lastCPUData = c.curCPUData

	if totalPeriod < 0 {
		return -2
	}
	if totalPeriod == 0 {
		return 0
	}
	return float64(usedPeriod * 100.0 / totalPeriod)
}

// NewLinuxCPUsageCollector 新建一个linux的CPU使用率收集器
func NewLinuxCPUsageCollector() (CPUsageCollector, error) {
	c := &linuxCPUsageCollector{}
	curLinuxCPUData, err := getCurLinuxCPUData()
	if err != nil {
		return nil, fmt.Errorf("getCurLinuxCPUData error:%s", err.Error())
	}
	c.lastCPUData = curLinuxCPUData
	return c, nil
}
