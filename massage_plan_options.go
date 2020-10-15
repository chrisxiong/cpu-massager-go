package cpumassager

import "fmt"

type options struct {
	cpusageCollector CPUsageCollector

	// highLoadLevel 高负荷等级，和CPU使用率计数器对等，用来判断CPU的
	// 负载是否过高，当CPU的使用率高于tirenewwLevel，则认为当前CPU负载
	// 过高，例如，如果highLoadLevel为CounterTypeSeventy，那么CPU使用
	// 率>=70就认为当前CPU高负荷了
	// highLoadLevel需要和loadStatusJudgeRatio配合使用，单次的CPU使用率
	// 超过所配置的highLoadLevel有可能是毛刺，cpusageRecorder会每隔一
	// 秒钟记录一次CPU使用率，最多纪录100次，如果100次记录中超过所配置
	// 的highLoadLevel的数量>=100*loadStatusJudgeRatio，则认为CPU当前在
	// 疲累状态需要根据按摩力度算法按一定比例拒绝请求（做下马杀鸡）
	highLoadLevel CounterType
	// loadStatusJudgeRatio 负荷状态判别比例
	loadStatusJudgeRatio float64

	// initialIntensity 和stepIntensity、currentIntensity、checkPeriodInSeconds
	// 配合使用，intensity 表示按摩力度，是一个[0, 100]的数值，代表以多大比例拒
	// 绝服务，initialIntensity是初始按摩力度，表示CPU刚进入疲累状态时候拒绝服务
	// 的概率，每隔checkPeriodInSeconds检查周期，会根据情况调整currentIntensity
	// 如果CPU在检查周期内依然疲累，则以stepIntensity步进调高按摩力度
	// 如果CPU在检查周期内疲累降低，则以stepIntensity步进降低按摩力度
	initialIntensity     uint // 推荐50，发生过载就以50%的概率拒绝服务，快降
	stepIntensity        uint // 推荐5，以5%的幅度升降拒绝服务的概率，慢调
	checkPeriodInSeconds uint
}

// isValid 用来判断options中的各个选项参数是否合法
func (o *options) isValid() (bool, error) {
	if o.cpusageCollector == nil {
		return false, fmt.Errorf("cpusageCollector should not be nil")
	}
	if o.loadStatusJudgeRatio > 1.0 || o.loadStatusJudgeRatio < 0.1 {
		return false, fmt.Errorf("loadStatusJudgeRatio should in [0.1, 1.0], 0.2 is recommended(means cpu can enter tired in 20 seconds)")
	}
	if o.initialIntensity > fullIntensity {
		return false, fmt.Errorf("initialIntensity should not greater than:%d, 50 is recommended(means 50%% tasks will be ignored)",
			fullIntensity)
	}
	if o.stepIntensity > maxStepIntensity {
		return false, fmt.Errorf("stepIntensity should not greater than:%d, 1 is recommended", maxStepIntensity)
	}
	if o.checkPeriodInSeconds > maxCheckPeriodInSeconds {
		return false, fmt.Errorf("checkPeriodInSeconds should not greater than:%d, 3 is recommended", maxCheckPeriodInSeconds)
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

// WithHighLoadLevel 用来设定massagePlan的高负荷等级
func WithHighLoadLevel(highLoadLevel CounterType) Option {
	return func(o *options) {
		o.highLoadLevel = highLoadLevel
	}
}

// WithLoadStatusJudgeRatio 用来设定massagePlan的高负荷判别比例
func WithLoadStatusJudgeRatio(loadStatusJudgeRatio float64) Option {
	return func(o *options) {
		o.loadStatusJudgeRatio = loadStatusJudgeRatio
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
