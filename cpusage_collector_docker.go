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

var cpuNum = 0

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
		cpuPercent = (dockerDelta / systemDelta) * float64(cpuNum) * 100.0
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
	cpuNum, err = getCPUNum()
	if err != nil {
		return nil, fmt.Errorf("getCPUNum error:%s", err.Error())
	}
	c.lastCPUData = curDockerCPUData
	return c, nil
}
