package cpumassager

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCurDockerCPUData(t *testing.T) {
	assert := assert.New(t)
	curCPUData, err := getCurDockerCPUData()
	require.Nil(t, err, "getCurDockerCPUData return err")
	require.NotNil(t, curCPUData, "getCurDockerCPUData return nil dockerCPUData")
	assert.Less(uint64(0), curCPUData.dockerUsage)
	assert.Less(uint64(0), curCPUData.systemUsage)
}

func TestGetCPUsageDocker(t *testing.T) {
	assert := assert.New(t)
	c, err := NewDockerCPUsageCollector()
	if !assert.NotNil(c) {
		assert.FailNow("NewDockerCPUsageCollector return nil")
	}
	if !assert.Nil(err) {
		assert.FailNow("NewDockerCPUsageCollector return error")
	}
	time.Sleep(time.Duration(time.Millisecond * 10))
	assert.NotEqual(0, c.GetCPUsage())
}
