package cpumassager

import (
	"testing"

	"github.com/bmizerany/assert"
)

func TestCPUsageRecorder(t *testing.T) {
	recorder := cpusageRecorder{}
	for _, counter := range allCounterTypes() {
		assert.Equal(t, 0, recorder.getRecordNumOfCounterType(counter))
	}

	// 先灌满100条记录
	for i := 0; i < 100; i++ {
		recorder.addRecord(75)
	}
	// 超过100条记录的会涉及到相关计数器的增减
	recorder.addRecord(69) // CounterTypeSeventy减1
	recorder.addRecord(51) // CounterTypeSixty和CounterTypeSeventy分别减1
	// 非法的记录数不影响计数器的记录
	recorder.addRecord(-1)
	recorder.addRecord(101)
	var testCases = []struct {
		in       CounterType
		expected int
	}{
		{CounterTypeZero, 100},
		{CounterTypeTen, 100},
		{CounterTypeTwenty, 100},
		{CounterTypeThirty, 100},
		{CounterTypeForty, 100},
		{CounterTypeFifty, 100},
		{CounterTypeSixty, 99},
		{CounterTypeSeventy, 98},
		{CounterTypeEighty, 0},
		{CounterTypeNinety, 0},
	}
	for _, testCase := range testCases {
		assert.Equal(t, testCase.expected, recorder.getRecordNumOfCounterType(testCase.in))
	}
}
