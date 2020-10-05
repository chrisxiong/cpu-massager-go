package cpumassager

import (
	"fmt"
	"sync/atomic"
	"time"
)

var planInst *massagePlan

func init() {
	planInst = &massagePlan{
		isStarted:       false,
		cpusageRecorder: cpusageRecorder{},
		currentState:    stateRelaxed{},
	}
}

const (
	emptyIntensity = 0
	fullIntensity  = 100
)

// massagePlan 马杀鸡计划
type massagePlan struct {
	// isStarted 判断马杀鸡计划是否已经启动的标识字段，避免重复调用
	isStarted bool

	cpusageCollector CPUsageCollector
	cpusageRecorder  cpusageRecorder
	currentState     massagePlanState

	// tirenessLevel 疲劳等级，和CPU使用率计数器对等，用来判断CPU的
	// 负载是否过高，当CPU的使用率高于tirenewwLevel，则认为当前CPU
	// 负载过高，例如，如果tirenessLevel为CounterTypeSeventy，那么
	// CPU使用率>=70就认为CPU高负载了
	// tirenessLevel需要和tiredRatio配合使用，单次的CPU使用率超过所
	// 配置的tirenessLevel有可能是毛刺，cpusageRecorder会每隔一秒钟
	// 记录一次CPU使用率，最多纪录100次，如果100次记录中超过所配置的
	// tirenessLevel的数量>=100*tiredRatio，则认为CPU当前在疲劳状态
	// 需要根据按摩力度算法按一定比例拒绝请求（做下马杀鸡）
	tirenessLevel CounterType
	tiredRatio    float64

	// initialIntensity 和stepIntensity、currentIntensity、checkPeriodInSeconds
	// 配合使用，intensity 表示按摩力度，是一个[0, 100]的数值，代表以多大比例拒
	// 绝服务，initialIntensity是初始按摩力度，表示CPU刚进入疲累状态时候拒绝服务
	// 的概率，每隔checkPeriodInSeconds检查周期，会根据情况调整currentIntensity
	// 如果CPU在检查周期内依然疲累，则以stepIntensity步进调高按摩力度
	// 如果CPU在检查周期内疲累降低，则以stepIntensity步进降低按摩力度
	initialIntensity     uint
	stepIntensity        uint
	currentIntensity     uint
	checkPeriodInSeconds uint

	// oldestTiredTime 检查周期内最早进入疲累状态的时间, 如果当前是疲累状态且
	// oldestTiredTime到当前的时间差超过了checkPeriodInSeconds，则需要将当前的
	// 按摩力度调高
	oldestTiredTime time.Time
	// latestTiredTime 检查周期内最近进入疲累状态的时间，如果当前是疲累状态且
	// latestTiredTime到当前的时间差超过了checkPeriodInSeconds，则需要将当前的
	// 按摩力度调低
	latestTiredTime time.Time
	// currentCPUsageRecordTime 当前的CPU使用率记录登记时间，初次进入疲累状态或
	// 调整按摩力度的时候，会需要用该时间重置oldestTiredTime和latestTiredTime
	currentCPUsageRecordTime time.Time

	// todoTasks 待处理任务，和doneTasks配合使用在疲累状态时候，依据按摩力度
	// 算法决定是否要做马杀鸡来拒绝服务，每次接受到请求都需要调用NeedMassage
	// 就会增加一个todoTask，当判断不需要做马杀鸡来拒绝服务，doneTask需要加一
	// 由于这两个变量可能会被多个业务routine读写，所以采用了原子操作
	todoTasks uint64
	// doneTasks 已处理任务
	doneTasks uint64
}

func (p *massagePlan) Start(cpusageCollector CPUsageCollector,
	tirenessLevel CounterType,
	initialIntensity uint,
	stepIntensity uint,
	tiredRatio float64,
	checkPeriodInSeconds uint) error {
	if p.isStarted == true {
		return fmt.Errorf("massage plan has been started")
	}

	if cpusageCollector == nil {
		return fmt.Errorf("cpusageCollector should not be nil")
	}
	p.cpusageCollector = cpusageCollector

	p.tirenessLevel = tirenessLevel

	if initialIntensity > fullIntensity {
		return fmt.Errorf("initialIntensity should less than:%d, 50 is recommended(means 50%% tasks will be ignored)", fullIntensity)
	}
	p.initialIntensity = initialIntensity
	p.currentIntensity = initialIntensity

	if stepIntensity > fullIntensity {
		return fmt.Errorf("stepIntensity should less than:%d, 10 is recommended", fullIntensity)
	}
	p.stepIntensity = stepIntensity

	if tiredRatio > 1.0 || tiredRatio < 0.0 {
		return fmt.Errorf("tiredRatio should in (0.0, 1.0), 0.6 isrecommended")
	}
	p.tiredRatio = tiredRatio

	p.checkPeriodInSeconds = checkPeriodInSeconds
	p.isStarted = true
	go func() {
		for {
			p.AddACPUsageRecord()
			time.Sleep(time.Duration(time.Second * 1))
		}
	}()
	return nil
}

func (p *massagePlan) StartLinux() error {
	linuxCPUsageCollector, err := NewLinuxCPUsageCollector()
	if err != nil {
		return fmt.Errorf("NewLinuxCPUsageCollector error:%s", err.Error())
	}
	const defaultTirenesLevel = CounterTypeFifty
	const defaultInitialIntensity = 50
	const defaultStepIntensity = 10
	const defaultTiredRatio = 0.6
	const defaultCheckPeriodInSeconds = 10
	return p.Start(linuxCPUsageCollector,
		defaultTirenesLevel,
		defaultInitialIntensity,
		defaultStepIntensity,
		defaultTiredRatio,
		defaultCheckPeriodInSeconds)
}

func (p *massagePlan) IsHighLoad() bool {
	tiredCount := p.cpusageRecorder.GetRecordNumOfCounterType(p.tirenessLevel)
	const recordSum = 100
	if tiredCount > int(recordSum*p.tiredRatio) {
		return true
	}
	return false
}

func (p *massagePlan) IsHighLoadDurationExceedCheckPeriod() bool {
	period := p.currentCPUsageRecordTime.Sub(p.oldestTiredTime)
	return period > time.Second*time.Duration(p.checkPeriodInSeconds)
}

func (p *massagePlan) IsLowLoadDurationExceedCheckPeriod() bool {
	period := p.currentCPUsageRecordTime.Sub(p.latestTiredTime)
	return period > time.Second*time.Duration(p.checkPeriodInSeconds)
}

func (p *massagePlan) SetRelaxed() {
	p.currentState = stateRelaxed{}
	p.currentIntensity = p.initialIntensity
	zeroTime := time.Time{}
	p.oldestTiredTime = zeroTime
	p.latestTiredTime = zeroTime
	p.clearWorkspace()
}

func (p *massagePlan) SetTired() {
	p.currentState = stateTired{}
	p.currentIntensity = p.initialIntensity
	p.UpdateLatestTiredTime()
	p.UpdateOldestTiredTime()
	p.clearWorkspace()
}

func (p *massagePlan) isRelaxed() bool {
	return p.currentState == stateRelaxed{}
}

func (p *massagePlan) isTired() bool {
	return p.currentState == stateTired{}
}

func (p *massagePlan) AddACPUsageRecord() {
	p.cpusageRecorder.AddRecord(p.cpusageCollector.GetCPUsage())
	p.updateCurTime()
	p.currentState.AddACPUsageRecord(p)
}

func (p *massagePlan) UpdateOldestTiredTime() {
	p.oldestTiredTime = p.currentCPUsageRecordTime
}

func (p *massagePlan) UpdateLatestTiredTime() {
	p.latestTiredTime = p.currentCPUsageRecordTime
}

func (p *massagePlan) updateCurTime() {
	p.currentCPUsageRecordTime = time.Now()
}

func (p *massagePlan) DecreaseIntensity() {
	if p.currentIntensity == emptyIntensity {
		p.SetRelaxed()
	} else {
		if p.currentIntensity > p.stepIntensity {
			p.currentIntensity -= p.stepIntensity
		} else {
			p.currentIntensity = emptyIntensity
		}
		p.UpdateOldestTiredTime()
		p.UpdateLatestTiredTime()
		p.clearWorkspace()
	}
}

func (p *massagePlan) IncreaseIntensity() {
	if p.currentIntensity+p.stepIntensity < fullIntensity {
		p.currentIntensity += p.stepIntensity
	} else {
		p.currentIntensity = fullIntensity
	}
	p.UpdateOldestTiredTime()
	p.UpdateLatestTiredTime()
	p.clearWorkspace()
}

func (p *massagePlan) clearWorkspace() {
	atomic.StoreUint64(&p.todoTasks, 0)
	atomic.StoreUint64(&p.doneTasks, 0)
}

func (p *massagePlan) addANewTask() {
	atomic.AddUint64(&p.todoTasks, 1)
}

func (p *massagePlan) finishATask() {
	atomic.AddUint64(&p.doneTasks, 1)
}

func (p *massagePlan) todoTaskNum() uint64 {
	return atomic.LoadUint64(&p.todoTasks)
}

func (p *massagePlan) doneTaskNum() uint64 {
	return atomic.LoadUint64(&p.doneTasks)
}

func (p *massagePlan) canDoWorkInTired() bool {
	p.addANewTask()
	requireTasks := p.todoTaskNum() * (fullIntensity - uint64(p.currentIntensity)) / fullIntensity
	if p.doneTaskNum() < requireTasks {
		p.finishATask()
		return true
	}
	return false
}

func (p *massagePlan) NeedMassage() bool {
	if p.isRelaxed() {
		return false
	}
	if p.canDoWorkInTired() {
		return false
	}
	return true
}

// StartMassagePlan 启动马杀鸡计划，参数的说明参见massagePlan的注释
// 使用方法和StartMassagePlanLinux一样
func StartMassagePlan(cpusageCollector CPUsageCollector,
	tirenessLevel CounterType,
	initialIntensity uint,
	stepIntensity uint,
	tiredRatio float64,
	checkPeriodInSeconds uint) error {
	return planInst.Start(cpusageCollector,
		tirenessLevel,
		initialIntensity,
		stepIntensity,
		tiredRatio,
		checkPeriodInSeconds)
}

// StartMassagePlanLinux 以默认参数启动linux环境的马杀鸡计划
// 启动程序后立即调用，启动马杀鸡计划
// func main() {
//     err := cpumassage.StartMassagePlanLinux()
//     if err != nil {
//         handleError() //  处理出错的情况，一般打印一下出错信息
//         os.Exit(1) //  然后退出就好了
//     }
//     serve() //  进入服务程序正常处理流程
// }
func StartMassagePlanLinux() error {
	return planInst.StartLinux()
}

// NeedMassage 非是否需要做下马杀鸡放松一下
// 每次收到请求都调用一下，若返回false，继续做后续处理，否则直接返回
// func handleARequest() {
//     if cpumassage.NeedMassage() {
//         refuse() //  拒绝服务该请求，做一些简单的处理，例如设定回包的错误码，上报过载告警等
//         return  //  然后直接返回
//     }
//     process() //  正常处理该请求
// }
func NeedMassage() bool {
	return planInst.NeedMassage()
}
