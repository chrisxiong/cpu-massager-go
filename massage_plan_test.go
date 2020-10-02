package cpumassager

import (
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMassagePlanState(t *testing.T) {
	assert := assert.New(t)
	mp := massagePlan{}
	mp.SetTired()
	assert.False(mp.isRelaxed())
	assert.True(mp.isTired())

	mp.SetRelaxed()
	assert.True(mp.isRelaxed())
	assert.False(mp.isTired())
}

func TestAddACPUsageRecordAndNeedMassage(t *testing.T) {
	require := require.New(t)
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	mockCollector := NewMockCPUsageCollector(mockCtl)
	mockCollector.EXPECT().GetCPUsage().Return(51.0).AnyTimes()

	mp := massagePlan{
		isStarted:        false,
		cpusageCollector: mockCollector,
		cpusageRecorder:  cpusageRecorder{},
		currentState:     stateRelaxed{},
		tirenessLevel:    CounterTypeFifty,
		initialIntensity: 50,
		stepIntensity:    10,
		currentIntensity: 50,
		tiredRatio:       0.6,
	}
	require.True(mp.isRelaxed())
	require.False(mp.isTired())
	require.False(mp.IsTiredCountExceedLimit())
	require.True(mp.oldestTiredTime.IsZero())
	require.True(mp.latestTiredTime.IsZero())
	require.True(mp.currentCPUsageRecordTime.IsZero())
	require.Equal(uint64(0), mp.todoTaskNum())
	require.Equal(uint64(0), mp.doneTaskNum())
	require.False(mp.NeedMassage())

	// tiredRatio 0.59
	collectCounts := 59
	for i := 0; i < collectCounts; i++ {
		mp.AddACPUsageRecord()
		require.False(mp.NeedMassage())
	}
	require.True(mp.isRelaxed())
	require.False(mp.isTired())
	require.False(mp.IsTiredCountExceedLimit())
	require.True(mp.oldestTiredTime.IsZero())
	require.True(mp.latestTiredTime.IsZero())
	require.False(mp.currentCPUsageRecordTime.IsZero())
	require.Equal(uint64(collectCounts)+1, mp.todoTaskNum())
	require.Equal(uint64(collectCounts)+1, mp.doneTaskNum())

	// tiredRatio 0.60
	mp.AddACPUsageRecord()
	require.True(mp.isRelaxed())
	require.False(mp.isTired())
	require.False(mp.IsTiredCountExceedLimit())
	require.True(mp.oldestTiredTime.IsZero())
	require.True(mp.latestTiredTime.IsZero())
	require.False(mp.currentCPUsageRecordTime.IsZero())
	require.False(mp.NeedMassage())
	require.Equal(uint64(collectCounts)+2, mp.todoTaskNum())
	require.Equal(uint64(collectCounts)+2, mp.doneTaskNum())

	// tiredRatio 0.61
	mp.AddACPUsageRecord()
	require.False(mp.isRelaxed())
	require.True(mp.isTired())
	require.True(mp.IsTiredCountExceedLimit())
	require.False(mp.oldestTiredTime.IsZero())
	require.False(mp.latestTiredTime.IsZero())
	require.False(mp.currentCPUsageRecordTime.IsZero())
	require.Equal(uint64(0), mp.todoTaskNum())
	require.Equal(uint64(0), mp.doneTaskNum())
	// 进入疲累状态后，按照50的按摩力度以50%的概率享受马杀鸡服务
	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			require.True(mp.NeedMassage())
		} else {
			require.False(mp.NeedMassage())
		}
	}
}

func TestDecreaseIntensity(t *testing.T) {
	require := require.New(t)
	mp := massagePlan{
		initialIntensity: 50,
		stepIntensity:    10,
		currentIntensity: 50,
	}
	mp.currentCPUsageRecordTime = time.Now()
	require.True(mp.oldestTiredTime.IsZero())
	require.True(mp.latestTiredTime.IsZero())
	require.False(mp.currentCPUsageRecordTime.IsZero())

	for i := 0; i < 5; i++ {
		mp.DecreaseIntensity()
		require.Equalf(mp.initialIntensity-(uint(i)+1)*mp.stepIntensity,
			mp.currentIntensity,
			"currentIntensity:%d not equal in ite:%d", mp.currentIntensity, i)
		require.Equal(mp.currentCPUsageRecordTime, mp.oldestTiredTime)
		require.Equal(mp.currentCPUsageRecordTime, mp.latestTiredTime)
		require.False(mp.currentCPUsageRecordTime.IsZero())
	}
	require.Equalf(uint(emptyIntensity),
		mp.currentIntensity,
		"currentIntensity:%d not equal", mp.currentIntensity)

	mp.DecreaseIntensity()
	require.Equalf(mp.initialIntensity,
		mp.currentIntensity,
		"currentIntensity:%d not equal", mp.currentIntensity)
	require.True(mp.oldestTiredTime.IsZero())
	require.True(mp.latestTiredTime.IsZero())
	require.False(mp.currentCPUsageRecordTime.IsZero())
}

func TestIncreaseIntensity(t *testing.T) {
	require := require.New(t)
	mp := massagePlan{
		initialIntensity: 50,
		stepIntensity:    10,
		currentIntensity: 50,
	}
	mp.currentCPUsageRecordTime = time.Now()
	require.True(mp.oldestTiredTime.IsZero())
	require.True(mp.latestTiredTime.IsZero())
	require.False(mp.currentCPUsageRecordTime.IsZero())

	for i := 0; i < 5; i++ {
		mp.IncreaseIntensity()
		if i < 5 {
			require.Equalf(mp.initialIntensity+(uint(i)+1)*mp.stepIntensity,
				mp.currentIntensity,
				"currentIntensity:%d not equal in ite:%d", mp.currentIntensity, i)
			require.Equal(mp.currentCPUsageRecordTime, mp.oldestTiredTime)
			require.Equal(mp.currentCPUsageRecordTime, mp.latestTiredTime)
			require.False(mp.currentCPUsageRecordTime.IsZero())
		}
	}
	mp.IncreaseIntensity()
	require.Equalf(uint(fullIntensity), mp.currentIntensity,
		"currentIntensity:%d not equal", mp.currentIntensity)
}

func TestIsOldestTiredTimeExceedCheckPeriod(t *testing.T) {
	periodInSecond := 10
	curTime := time.Now()
	mp := massagePlan{
		currentCPUsageRecordTime: curTime,
		oldestTiredTime:          time.Unix(0, curTime.UnixNano()-int64(periodInSecond)*1e9-1),
		checkPeriodInSeconds:     uint(periodInSecond),
	}
	assert.Truef(t, mp.IsOldestTiredTimeExceedCheckPeriod(),
		"curTime:%v, oldestTiredTime:%v, period:%v", mp.currentCPUsageRecordTime, mp.oldestTiredTime, mp.currentCPUsageRecordTime.Sub(mp.oldestTiredTime))
}

func TestIsLatestTiredTimeExceedCheckPeriod(t *testing.T) {
	periodInSecond := 10
	curTime := time.Now()
	mp := massagePlan{
		currentCPUsageRecordTime: curTime,
		latestTiredTime:          time.Unix(0, curTime.UnixNano()-int64(periodInSecond)*1e9-1),
		checkPeriodInSeconds:     uint(periodInSecond),
	}
	assert.Truef(t, mp.IsLatestTiredTimeExceedCheckPeriod(),
		"curTime:%v, oldestTiredTime:%v, period:%v", mp.currentCPUsageRecordTime, mp.oldestTiredTime, mp.currentCPUsageRecordTime.Sub(mp.oldestTiredTime))
}

func TestCanDoWork(t *testing.T) {
	require := assert.New(t)
	mp := massagePlan{
		initialIntensity: 50,
		stepIntensity:    10,
		currentIntensity: 50,
	}
	mp.SetRelaxed()
	require.Equal(uint64(0), mp.todoTaskNum())
	require.Equal(uint64(0), mp.doneTaskNum())
	require.False(mp.NeedMassage())
	require.Equal(uint64(1), mp.todoTaskNum())
	require.Equal(uint64(1), mp.doneTaskNum())

	// 疲劳状态下50%的概率丢包
	mp.SetTired()
	require.Equal(uint64(0), mp.todoTaskNum())
	require.Equal(uint64(0), mp.doneTaskNum())
	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			require.True(mp.NeedMassage())
		} else {
			require.False(mp.NeedMassage())
		}
	}

	// 疲劳状态下60%的概率丢包
	mp.IncreaseIntensity()
	require.Equal(uint(60), mp.currentIntensity)
	require.Equal(uint64(0), mp.todoTaskNum())
	require.Equal(uint64(0), mp.doneTaskNum())
	for i := 0; i < 100; i++ {
		// 60%的概率丢包，请求号尾数为2、4、7、9的请求需要干活，其他则马杀鸡对待
		strIte := strconv.Itoa(i)
		unit := strIte[len(strIte)-1 : len(strIte)]
		if unit == "2" || unit == "4" || unit == "7" || unit == "9" {
			assert.Falsef(t, mp.NeedMassage(), "error in ite:%d, unit:%s", i, unit)
		} else {
			assert.Truef(t, mp.NeedMassage(), "error in ite:%d, unit:%s", i, unit)
		}
	}
}
