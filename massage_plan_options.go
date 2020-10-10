package cpumassager

import "fmt"

type options struct {
	cpusageCollector CPUsageCollector

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
	checkPeriodInSeconds uint
}

// isValid 用来判断options中的各个选项参数是否合法
func (o *options) isValid() (bool, error) {
	if o.cpusageCollector == nil {
		return false, fmt.Errorf("cpusageCollector should not be nil")
	}
	if o.initialIntensity > fullIntensity {
		return false, fmt.Errorf("initialIntensity should less than:%d, 50 is recommended(means 50%% tasks will be ignored)",
			fullIntensity)
	}
	if o.stepIntensity > fullIntensity {
		return false, fmt.Errorf("stepIntensity should less than:%d, 10 is recommended", fullIntensity)
	}
	if o.tiredRatio > 1.0 || o.tiredRatio < 0.0 {
		return false, fmt.Errorf("tiredRatio should in (0.0, 1.0), 0.6 isrecommended")
	}
	if o.checkPeriodInSeconds > 100 {
		return false, fmt.Errorf("checkPeriodInSeconds:%d, too long, <30 is recommended", o.checkPeriodInSeconds)
	}
	return true, nil
}

// Option 用来设定massagePlan的启动参数的函数
type Option func(*options)

// WithCPUSageCollector 用来设定massagePlan的CPU使用率收集器
func WithCPUSageCollector(cpusageCollector CPUsageCollector) Option {
	return func(o *options) {
		o.cpusageCollector = cpusageCollector
	}
}

// WithTirenessLevel 用来设定massagePlan的疲累等级
func WithTirenessLevel(tirenessLevel CounterType) Option {
	return func(o *options) {
		o.tirenessLevel = tirenessLevel
	}
}

// WithTiredRatio 用来设定massagePlan的疲累判别比例
func WithTiredRatio(tiredRatio float64) Option {
	return func(o *options) {
		o.tiredRatio = tiredRatio
	}
}

// WithInitialIntensity 用来设定massagePlan的初始按摩力度
func WithInitialIntensity(initialIntensity uint) Option {
	return func(o *options) {
		o.initialIntensity = initialIntensity
	}
}

// WithStepIntensity 用来设定massagePlan的步进按摩力度
func WithStepIntensity(stepIntensity uint) Option {
	return func(o *options) {
		o.stepIntensity = stepIntensity
	}
}

// WithCheckPeriodInseconds 用来设定massagePlan检查CPU状态的周期
func WithCheckPeriodInseconds(checkPeriodInseconds uint) Option {
	return func(o *options) {
		o.checkPeriodInSeconds = checkPeriodInseconds
	}
}
