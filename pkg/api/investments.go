package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/investments"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"strconv"
)

type InvestmentsApi struct{}

var Investments = &InvestmentsApi{}

func (a *InvestmentsApi) List(c *core.WebContext) (any, *errs.Error) {
	rows, err := services.Investments.List(c, c.GetCurrentUid())
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return rows, nil
}
func (a *InvestmentsApi) Detail(c *core.WebContext) (any, *errs.Error) {
	result, err := services.Investments.Detail(c, c.GetCurrentUid(), c.Query("id"))
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return result, nil
}
func (a *InvestmentsApi) Create(c *core.WebContext) (any, *errs.Error) {
	var req services.InvestmentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	result, err := services.Investments.Create(c, c.GetCurrentUid(), req)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return result, nil
}
func (a *InvestmentsApi) Preview(c *core.WebContext) (any, *errs.Error) { return a.apply(c, true) }
func (a *InvestmentsApi) Save(c *core.WebContext) (any, *errs.Error)    { return a.apply(c, false) }
func (a *InvestmentsApi) apply(c *core.WebContext, preview bool) (any, *errs.Error) {
	var req services.InvestmentOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	// Preserve upstream edit-period and reconciled-account restrictions.
	user, err := services.Users.GetUserById(c, c.GetCurrentUid())
	if err != nil {
		return nil, errs.Or(err, errs.ErrUserNotFound)
	}
	zone, err := c.GetClientTimezone()
	if err != nil {
		return nil, errs.ErrClientTimezoneOffsetInvalid
	}
	existing, err := services.Investments.Detail(c, c.GetCurrentUid(), req.PositionId)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	ids := []int64{existing.Position.CostAccountId, req.CashAccountId}
	for _, op := range existing.Operations {
		if op.CashAccountId > 0 {
			ids = append(ids, op.CashAccountId)
		}
	}
	accounts, err := services.Accounts.GetAccountsByAccountIds(c, c.GetCurrentUid(), ids)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	if !user.CanEditTransactionByTransactionTime(utils.GetMinTransactionTimeFromUnixTime(req.OccurredAt), zone, accounts[existing.Position.CostAccountId], accounts[req.CashAccountId]) {
		return nil, errs.ErrCannotModifyTransactionWithThisTransactionTime
	}
	for _, op := range existing.Operations {
		if req.OperationId != "" || op.OccurredAt >= req.OccurredAt {
			if !op.Cancelled && !user.CanEditTransactionByTransactionTime(utils.GetMinTransactionTimeFromUnixTime(op.OccurredAt), zone, accounts[existing.Position.CostAccountId], accounts[op.CashAccountId]) {
				return nil, errs.ErrCannotModifyTransactionWithThisTransactionTime
			}
		}
	}
	result, err := services.Investments.Apply(c, c.GetCurrentUid(), req, preview)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return result, nil
}

func (a *InvestmentsApi) Attach(c *core.WebContext) (any, *errs.Error) {
	var req struct {
		PositionId  string `json:"positionId"`
		OperationId string `json:"operationId"`
		PictureId   int64  `json:"pictureId,string"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	if err := services.Investments.Attach(c, c.GetCurrentUid(), req.PositionId, req.OperationId, req.PictureId); err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return true, nil
}

// Valuation reads the configured snapshot provider; it never mutates the ledger.
func (a *InvestmentsApi) Valuation(c *core.WebContext) (any, *errs.Error) {
	detail, err := services.Investments.Detail(c, c.GetCurrentUid(), c.Query("id"))
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	config := settings.Container.GetCurrentConfig()
	if config.InvestmentQuoteURL == "" {
		return map[string]any{"status": "unconfigured", "marketValue": nil}, nil
	}
	quote, err := (investments.CMBPlatformProvider{Endpoint: config.InvestmentQuoteURL, Token: config.InvestmentQuoteToken}).Fetch(c)
	if err != nil {
		return map[string]any{"status": "unavailable", "reason": err.Error(), "marketValue": nil}, nil
	}
	value, err := investments.Value(detail.Position.Quantity, quote.CustomerSell)
	if err != nil {
		return nil, errs.ErrInvestmentInvalid
	}
	return map[string]any{"status": "reference", "marketValue": strconv.FormatInt(value, 10), "unrealized": strconv.FormatInt(value-detail.Position.Cost, 10), "positionVersion": detail.Position.Version, "quote": quote, "note": "参考估值；NowTime 不证明价格最后更新时间，最终成交以招行交易界面为准"}, nil
}
