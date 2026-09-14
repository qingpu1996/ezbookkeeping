package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/services"
)

type AccountGroupsApi struct{}

var AccountGroups = &AccountGroupsApi{}

func (a *AccountGroupsApi) List(c *core.WebContext) (any, *errs.Error) {
	r, e := services.AccountGroups.List(c, c.GetCurrentUid())
	if e != nil {
		return nil, errs.Or(e, errs.ErrOperationFailed)
	}
	return r, nil
}
func (a *AccountGroupsApi) Save(c *core.WebContext) (any, *errs.Error) {
	var req services.AccountGroupRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(e)
	}
	r, e := services.AccountGroups.Save(c, c.GetCurrentUid(), req)
	if e != nil {
		return nil, errs.Or(e, errs.ErrOperationFailed)
	}
	return r, nil
}
func (a *AccountGroupsApi) Delete(c *core.WebContext) (any, *errs.Error) {
	var req services.AccountGroupRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(e)
	}
	if e := services.AccountGroups.Delete(c, c.GetCurrentUid(), req); e != nil {
		return nil, errs.Or(e, errs.ErrOperationFailed)
	}
	return true, nil
}
func (a *AccountGroupsApi) Assign(c *core.WebContext) (any, *errs.Error) {
	var req services.AccountGroupAssignRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(e)
	}
	if e := services.AccountGroups.Assign(c, c.GetCurrentUid(), req); e != nil {
		return nil, errs.Or(e, errs.ErrOperationFailed)
	}
	return true, nil
}
