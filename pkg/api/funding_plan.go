package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/services"
)

type FundingPlanApi struct{}

var FundingPlans = &FundingPlanApi{}

func (a *FundingPlanApi) Get(c *core.WebContext) (any, *errs.Error) {
	r, e := services.FundingPlans.Get(c, c.GetCurrentUid())
	if e != nil {
		return nil, errs.Or(e, errs.ErrOperationFailed)
	}
	return r, nil
}
func (a *FundingPlanApi) Save(c *core.WebContext) (any, *errs.Error) {
	var req services.FundingPlanRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(e)
	}
	r, e := services.FundingPlans.Save(c, c.GetCurrentUid(), req)
	if e != nil {
		return nil, errs.Or(e, errs.ErrOperationFailed)
	}
	return r, nil
}
