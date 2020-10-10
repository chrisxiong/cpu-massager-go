package cpumassager

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptions(t *testing.T) {
	require := require.New(t)
	options := &options{}
	require.Nil(options.cpusageCollector)
	require.Equal(CounterTypeZero, options.highLoadLevel)
	require.Equal(0.0, options.highLoadRatio)
	require.Equal(uint(0), options.initialIntensity)
	require.Equal(uint(0), options.stepIntensity)
	require.Equal(uint(0), options.checkPeriodInSeconds)

	linuxCPUsageCollector, _ := NewLinuxCPUsageCollector()
	require.NotNil(linuxCPUsageCollector)
	const defaultHighLoadLevel = CounterTypeEighty
	const defaultHighLoadRatio = 0.6
	const defaultInitialIntensity = 50
	const defaultStepIntensity = 10
	const defaultCheckPeriodInSeconds = 10
	WithCPUSageCollector(linuxCPUsageCollector)(options)
	WithHighLoadLevel(defaultHighLoadLevel)(options)
	WithHighLoadRatio(defaultHighLoadRatio)(options)
	WithInitialIntensity(defaultInitialIntensity)(options)
	WithStepIntensity(defaultStepIntensity)(options)
	WithCheckPeriodInseconds(defaultCheckPeriodInSeconds)(options)

	require.Equal(linuxCPUsageCollector, options.cpusageCollector)
	require.Equal(defaultHighLoadLevel, options.highLoadLevel)
	require.Equal(defaultHighLoadRatio, options.highLoadRatio)
	require.Equal(uint(defaultInitialIntensity), options.initialIntensity)
	require.Equal(uint(defaultStepIntensity), options.stepIntensity)
	require.Equal(uint(defaultCheckPeriodInSeconds), options.checkPeriodInSeconds)
}
