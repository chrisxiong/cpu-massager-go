package cpumassager

import (
	"testing"
)

func TestStateRelaxedToStateTired(t *testing.T) {
	/*
		var p *massagePlan
		monkey.PatchInstanceMethod(reflect.TypeOf(p), "IsTiredCountExceedLimit", func(_ *massagePlan) bool { return true })
		planInst.SetRelaxed()
		assert.False(t, planInst.isTired())
		assert.True(t, planInst.isRelaxed())

		relax := stateRelaxed{}
		relax.AddACPUsageRecord(planInst)
		assert.True(t, planInst.isTired())
		assert.False(t, planInst.isRelaxed())
	*/
}

func TestStateRelaxedToStateTiredV2(t *testing.T) {
	/*
		plan := &massagePlan{}
		var p *massagePlan
		patches := gomonkey.ApplyMethod(reflect.TypeOf(p), "IsTiredCountExceedLimit",
			func(_ *massagePlan) bool {
				return true
			})
		defer patches.Reset()
		plan.SetRelaxed()
		assert.True(t, plan.IsTiredCountExceedLimit())
		assert.False(t, plan.isTired())
		assert.True(t, plan.isRelaxed())

		relax := stateRelaxed{}
		relax.AddACPUsageRecord(plan)
		assert.True(t, planInst.isTired())
		assert.False(t, planInst.isRelaxed())
	*/
}
