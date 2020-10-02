package cpumassager

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCurLinuxCPUData(t *testing.T) {
	assert := assert.New(t)
	curCPUData, err := getCurLinuxCPUData()
	require.Nil(t, err, "getCurLinuxCPUData return err")
	require.NotNil(t, curCPUData, "getCurLinuxCPUData retun nil linuxCPUData")
	assert.Less(uint64(0), curCPUData.user)
	assert.Less(uint64(0), curCPUData.nice)
	assert.Less(uint64(0), curCPUData.system)
	assert.Less(uint64(0), curCPUData.idle)
	assert.Less(uint64(0), curCPUData.iowait)
	assert.Less(uint64(0), curCPUData.irq)
	assert.Less(uint64(0), curCPUData.softirq)
	assert.LessOrEqual(uint64(0), curCPUData.steal)
	assert.LessOrEqual(uint64(0), curCPUData.guest)
	assert.LessOrEqual(uint64(0), curCPUData.guestnice)
	assert.Less(uint64(0), curCPUData.total)
}

func TestGetCPUsageLinux(t *testing.T) {
	assert := assert.New(t)
	c, err := NewLinuxCPUsageCollector()
	if !assert.NotNil(c) {
		assert.FailNow("NewLinuxCPUsageCollector return nil")
	}
	if !assert.Nil(err) {
		assert.FailNow("NewLinuxCPUsageCollector return error")
	}
	time.Sleep(time.Duration(time.Millisecond * 10))
	assert.NotEqual(0, c.GetCPUsage())
}
