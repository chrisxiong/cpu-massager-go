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
	opts options

	// isStarted 判断马杀鸡计划是否已经启动的标识字段，避免重复调用
	isStarted       bool
	cpusageRecorder cpusageRecorder
	currentState    massagePlanState

	currentIntensity uint

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

func (p *massagePlan) Start(opts options) error {
	if valid, err := opts.isValid(); !valid {
		return fmt.Errorf("options invalid:%s", err.Error())
	}
	if p.isStarted == true {
		return fmt.Errorf("massage plan has been started")
	}
	p.opts = opts
	p.currentIntensity = opts.initialIntensity
	p.isStarted = true
	go func() {
		for {
			p.AddACPUsageRecord()
			time.Sleep(time.Duration(time.Second * 1))
		}
	}()
	return nil
}

func (p *massagePlan) IsHighLoad() bool {
	tiredCount := p.cpusageRecorder.GetRecordNumOfCounterType(p.opts.tirenessLevel)
	const recordSum = 100
	if tiredCount > int(recordSum*p.opts.tiredRatio) {
		return true
	}
	return false
}

func (p *massagePlan) IsHighLoadDurationExceedCheckPeriod() bool {
	period := p.currentCPUsageRecordTime.Sub(p.oldestTiredTime)
	return period > time.Second*time.Duration(p.opts.checkPeriodInSeconds)
}

func (p *massagePlan) IsLowLoadDurationExceedCheckPeriod() bool {
	period := p.currentCPUsageRecordTime.Sub(p.latestTiredTime)
	return period > time.Second*time.Duration(p.opts.checkPeriodInSeconds)
}

func (p *massagePlan) SetRelaxed() {
	p.currentState = stateRelaxed{}
	p.currentIntensity = p.opts.initialIntensity
	zeroTime := time.Time{}
	p.oldestTiredTime = zeroTime
	p.latestTiredTime = zeroTime
	p.clearWorkspace()
}

func (p *massagePlan) SetTired() {
	p.currentState = stateTired{}
	p.currentIntensity = p.opts.initialIntensity
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
	p.cpusageRecorder.AddRecord(p.opts.cpusageCollector.GetCPUsage())
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
		if p.currentIntensity > p.opts.stepIntensity {
			p.currentIntensity -= p.opts.stepIntensity
		} else {
			p.currentIntensity = emptyIntensity
		}
		p.UpdateOldestTiredTime()
		p.UpdateLatestTiredTime()
		p.clearWorkspace()
	}
}

func (p *massagePlan) IncreaseIntensity() {
	if p.currentIntensity+p.opts.stepIntensity < fullIntensity {
		p.currentIntensity += p.opts.stepIntensity
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

// StartMassagePlan 启动马杀鸡计划，在启动程序后立即调用
// func main() {
//     err := cpumassage.StartMassagePlan()
//     if err != nil {
//         handleError() //  处理出错的情况，一般打印一下出错信息
//         os.Exit(1) //  然后退出就好了
//     }
//     serve() //  进入服务程序正常处理流程
// }
func StartMassagePlan(opts ...Option) error {
	const defaultTirenesLevel = CounterTypeEighty // CPU使用率>=80%算是高负荷
	const defaultTiredRatio = 0.6                 // 高负荷占比超过60%
	const defaultInitialIntensity = 50            // 初始化的按摩力度，50表示50%的概率拒绝服务，快速降温
	const defaultStepIntensity = 10               // 以10为粒度上下调整按摩力度
	const defaultCheckPeriodInSeconds = 10        // 每隔10秒钟审视当前按摩力度是否合适
	options := &options{
		tirenessLevel:        defaultTirenesLevel,
		tiredRatio:           defaultTiredRatio,
		initialIntensity:     defaultInitialIntensity,
		stepIntensity:        defaultStepIntensity,
		checkPeriodInSeconds: defaultCheckPeriodInSeconds,
	}
	for _, o := range opts {
		o(options)
	}
	if options.cpusageCollector == nil {
		// 没有指定CPU使用率收集器，那么采用Linux采集器
		linuxCPUsageCollector, err := NewLinuxCPUsageCollector()
		if err != nil {
			return fmt.Errorf("NewLinuxCPUsageCollector error:%s", err.Error())
		}
		options.cpusageCollector = linuxCPUsageCollector
	}
	if valid, err := options.isValid(); !valid {
		return fmt.Errorf("options invalid:%s", err.Error())
	}
	return planInst.Start(*options)
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
