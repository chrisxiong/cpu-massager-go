package cpumassager

// massagePlanState 马杀鸡计划的状态，用于进行添加请求记录之后判别状态扭转
type massagePlanState interface {
	// AddACPUsageRecord 添加一条CPU使用记录，并尝试扭转massagePlan的状态
	AddACPUsageRecord(p *massagePlan)
}

// stateRelaxed 放松状态，在满足条件时候扭转到疲累状态
type stateRelaxed struct{}

func (s stateRelaxed) AddACPUsageRecord(p *massagePlan) {
	if p.IsTiredCountExceedLimit() {
		p.UpdateLatestTiredTime()
		p.UpdateOldestTiredTime()
		p.SetTired()
	}
}

// stateTired 疲累状态，在满足条件时候扭转到放松状态
type stateTired struct{}

func (s stateTired) AddACPUsageRecord(p *massagePlan) {
	if p.IsTiredCountExceedLimit() {
		p.UpdateLatestTiredTime()
		if p.IsOldestTiredTimeExceedCheckPeriod() {
			p.IncreaseIntensity()
		}
	} else {
		if p.IsLatestTiredTimeExceedCheckPeriod() {
			p.DecreaseIntensity()
		}
	}
}
