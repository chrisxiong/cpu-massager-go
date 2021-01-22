package cpumassager

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

const (
	clockTicksPerSecond  = 100 // linux环境下是一个100的常量
	nanoSecondsPerSecond = 1e9
)

var hostCPUNum = 1
var containerCPUNum = 1.0

// dockerCPUData linux系统docker环境下的CPU使用率相关数据
type dockerCPUData struct {
	dockerUsage uint64 // 从/sys/fs/cgroup/cpuacct/cpuacct.usage获取，以纳秒为单位
	systemUsage uint64 // 从/proc/stat中获取的以clock_tick为单位的值，换算后得到纳秒
}

// getCurDockerCPUAcctUsage 获取docker的cpuacct.usage的值
func getCurDockerCPUAcctUsage() (uint64, error) {
	const usageFile = "/sys/fs/cgroup/cpuacct/cpuacct.usage"
	v, err := ioutil.ReadFile(usageFile)
	if err != nil {
		return 0, fmt.Errorf("ReadFile:%s, error:%s", usageFile, err.Error())
	}
	usage, err := strconv.ParseUint(strings.TrimSpace(string(v)), 10, 64)
	if err != nil {
		return 0, err
	}
	return usage, nil
}

// getCPUNum 获取CPU数量
func getCPUNum() (int, error) {
	const usageFile = "/sys/fs/cgroup/cpuacct/cpuacct.usage_percpu"
	v, err := ioutil.ReadFile(usageFile)
	if err != nil {
		return 0, fmt.Errorf("ReadFile:%s, error:%s", usageFile, err.Error())
	}
	return len(strings.Fields(string(v))), nil
}

// getContainerCPUNumWithCFS CFS作SHCED_NORMAL的CPU使用率控制机制，被docker推荐使用
// SCHED_RT未支持
func getContainerCPUNumWithCFS() float64 {
	const cfsQuotaFile = "/sys/fs/cgroup/cpu/cpu.cfs_quota_us"
	cfsQuota, err := ioutil.ReadFile(cfsQuotaFile)
	if err != nil {
		return float64(hostCPUNum)
	}
	cfsQuotaUS, err := strconv.ParseFloat(strings.TrimSpace(string(cfsQuota)), 64)
	if cfsQuotaUS < 0 {
		return float64(hostCPUNum)
	}

	const cfsPeriodFile = "/sys/fs/cgroup/cpu/cpu.cfs_period_us"
	cfsPeriod, err := ioutil.ReadFile(cfsPeriodFile)
	if err != nil {
		return float64(hostCPUNum)
	}
	cfsPeriodUS, err := strconv.ParseFloat(strings.TrimSpace(string(cfsPeriod)), 64)
	if cfsPeriodUS < 0 {
		return float64(hostCPUNum)
	}

	return cfsQuotaUS / cfsPeriodUS
}

// getCurDockerCPUData 获取当前的docker CPUData
func getCurDockerCPUData() (*dockerCPUData, error) {
	d := &dockerCPUData{}
	linuxCPUData, err := getCurLinuxCPUData()
	if err != nil {
		return nil, fmt.Errorf("getCurLinuxCPUData error:%s", err.Error())
	}
	d.systemUsage = linuxCPUData.total / clockTicksPerSecond * nanoSecondsPerSecond
	dockerUsage, err := getCurDockerCPUAcctUsage()
	if err != nil {
		return nil, fmt.Errorf("getCurDockerCPUAcctUsage error:%s", err.Error())
	}
	d.dockerUsage = dockerUsage
	return d, nil
}

// dockerCPUsageCollector docker系统的CPU使用率收集器
type dockerCPUsageCollector struct {
	lastCPUData *dockerCPUData
	curCPUData  *dockerCPUData
}

func (c *dockerCPUsageCollector) GetCPUsage() float64 {
	curDockerCPUData, err := getCurDockerCPUData()
	if err != nil {
		return 0.0
	}
	c.curCPUData = curDockerCPUData
	if c.lastCPUData == nil {
		c.lastCPUData = c.curCPUData
		return 0.0
	}

	var (
		cpuPercent  = 0.0
		dockerDelta = float64(c.curCPUData.dockerUsage) - float64(c.lastCPUData.dockerUsage)
		systemDelta = float64(c.curCPUData.systemUsage) - float64(c.lastCPUData.systemUsage)
	)

	if dockerDelta > 0.0 && systemDelta > 0.0 {
		cpuPercent = (dockerDelta / systemDelta) * float64(hostCPUNum) * 100.0 / containerCPUNum
	}
	c.lastCPUData = c.curCPUData

	return cpuPercent
}

// NewDockerCPUsageCollector 新建一个docker的CPU使用率收集器
func NewDockerCPUsageCollector() (CPUsageCollector, error) {
	c := &dockerCPUsageCollector{}
	curDockerCPUData, err := getCurDockerCPUData()
	if err != nil {
		return nil, fmt.Errorf("getCurDockerCPUData error:%s", err.Error())
	}
	hostCPUNum, err = getCPUNum()
	if err != nil {
		return nil, fmt.Errorf("getCPUNum error:%s", err.Error())
	}
	containerCPUNum = getContainerCPUNumWithCFS()
	c.lastCPUData = curDockerCPUData
	return c, nil
}
