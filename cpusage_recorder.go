package cpumassager

// cpusageRecorder cpu使用率记录器
type cpusageRecorder struct {
	recordCounters [10]int
}

// CounterType 计数器类型，用来记录满足不同条件的CPU使用率情况
type CounterType int

const (
	// CounterTypeZero 用来记录CPU使用率>=0情况的计数器
	CounterTypeZero CounterType = 0

	// CounterTypeTen 用来记录CPU使用率>=10情况的计数器
	CounterTypeTen CounterType = 1

	// CounterTypeTwenty 用来记录CPU使用率>=20情况的计数器
	CounterTypeTwenty CounterType = 2

	// CounterTypeThirty 用来记录CPU使用率>=30情况的计数器
	CounterTypeThirty CounterType = 3

	// CounterTypeForty 用来记录CPU使用率>=40情况的计数器
	CounterTypeForty CounterType = 4

	// CounterTypeFifty 用来记录CPU使用率>=50情况的计数器
	CounterTypeFifty CounterType = 5

	// CounterTypeSixty 用来记录CPU使用率>=60情况的计数器
	CounterTypeSixty CounterType = 6

	// CounterTypeSeventy 用来记录CPU使用率>=70情况的计数器
	CounterTypeSeventy CounterType = 7

	// CounterTypeEighty 用来记录CPU使用率>=80情况的计数器
	CounterTypeEighty CounterType = 8

	// CounterTypeNinety 用来记录CPU使用率>=90情况的计数器
	CounterTypeNinety CounterType = 9
)

// allCounterTypes 返回所有的计数器类型
func allCounterTypes() []CounterType {
	return []CounterType{
		CounterTypeZero,
		CounterTypeTen,
		CounterTypeTwenty,
		CounterTypeThirty,
		CounterTypeForty,
		CounterTypeFifty,
		CounterTypeSixty,
		CounterTypeSeventy,
		CounterTypeEighty,
		CounterTypeNinety,
	}
}

// addRecord 添加一条cpu使用率的记录
func (r *cpusageRecorder) addRecord(cpusage float64) {
	if cpusage < 0 || cpusage > 100 {
		return
	}

	for _, counterType := range allCounterTypes() {
		if cpusage >= float64(counterType)*10 {
			if r.recordCounters[counterType] < 100 {
				r.recordCounters[counterType]++
			}
		} else {
			if r.recordCounters[counterType] > 0 {
				r.recordCounters[counterType]--
			}
		}
	}
}

// getRecordNumOfCounterType 获取制定计数器的记录数
func (r *cpusageRecorder) getRecordNumOfCounterType(ct CounterType) int {
	return r.recordCounters[ct]
}
