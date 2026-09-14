package api

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/investments"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/services"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"strconv"
	"time"
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

// Valuation reads configured prices without posting any ledger transaction.
func (a *InvestmentsApi) Valuation(c *core.WebContext) (any, *errs.Error) {
	detail, err := services.Investments.Detail(c, c.GetCurrentUid(), c.Query("id"))
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	setting, err := services.Investments.ValuationSettings(c, c.GetCurrentUid(), detail.Position.Id)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return valuePosition(c, detail.Position, setting, nil, nil), nil
}
func (a *InvestmentsApi) ValuationSettings(c *core.WebContext) (any, *errs.Error) {
	row, err := services.Investments.ValuationSettings(c, c.GetCurrentUid(), c.Query("id"))
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	config := settings.Container.GetCurrentConfig()
	return map[string]any{"settings": row, "automaticAvailable": config.InvestmentQuoteURL != "", "automaticName": config.InvestmentQuoteName, "automaticMaxAgeMinutes": config.InvestmentQuoteMaxAgeMinutes}, nil
}
func (a *InvestmentsApi) SaveValuationSettings(c *core.WebContext) (any, *errs.Error) {
	var req services.InvestmentValuationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}
	row, err := services.Investments.SaveValuationSettings(c, c.GetCurrentUid(), req)
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	return row, nil
}
func (a *InvestmentsApi) Valuations(c *core.WebContext) (any, *errs.Error) {
	rows, err := services.Investments.List(c, c.GetCurrentUid())
	if err != nil {
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}
	result := make([]map[string]any, 0, len(rows))
	var quote *investments.GoldQuote
	var quoteErr error
	fetched := false
	for _, p := range rows {
		setting, err := services.Investments.ValuationSettings(c, c.GetCurrentUid(), p.Id)
		if err != nil {
			return nil, errs.Or(err, errs.ErrOperationFailed)
		}
		if setting.Mode == "automatic" && !fetched {
			quote, quoteErr = fetchGoldQuote(c)
			fetched = true
		}
		result = append(result, valuePosition(c, p, setting, quote, quoteErr))
	}
	return result, nil
}
func fetchGoldQuote(c *core.WebContext) (*investments.GoldQuote, error) {
	config := settings.Container.GetCurrentConfig()
	return (investments.CMBPlatformProvider{Endpoint: config.InvestmentQuoteURL, Token: config.InvestmentQuoteToken, Provider: config.InvestmentQuoteProvider, MaxAge: time.Duration(config.InvestmentQuoteMaxAgeMinutes) * time.Minute}).Fetch(c)
}
func valuePosition(c *core.WebContext, p *models.InvestmentPosition, s *models.InvestmentValuationSetting, q *investments.GoldQuote, quoteErr error) map[string]any {
	result := map[string]any{"positionId": p.Id, "accountId": strconv.FormatInt(p.CostAccountId, 10), "positionVersion": p.Version, "cost": strconv.FormatInt(p.Cost, 10), "quantity": p.Quantity, "mode": s.Mode, "status": "cost", "marketValue": nil, "sourceName": "", "validUntil": int64(0)}
	if s.Mode == "cost" {
		return result
	}
	var price string
	var at time.Time
	maxAge := time.Duration(s.MaxAgeMinutes) * time.Minute
	result["status"] = "unavailable"
	if s.Mode == "manual" {
		price = s.ManualPrice
		at = time.Unix(s.ManualAsOf, 0)
		result["sourceName"] = "手动参考价"
	} else {
		config := settings.Container.GetCurrentConfig()
		result["sourceName"] = config.InvestmentQuoteName
		if config.InvestmentQuoteURL == "" {
			result["status"] = "unconfigured"
			return result
		}
		if q == nil && quoteErr == nil {
			q, quoteErr = fetchGoldQuote(c)
		}
		if quoteErr != nil {
			result["reason"] = quoteErr.Error()
			return result
		}
		price = q.CustomerSell
		at, _ = time.Parse(time.RFC3339Nano, q.FetchedAt)
		maxAge = time.Duration(config.InvestmentQuoteMaxAgeMinutes) * time.Minute
		result["quote"] = q
	}
	result["fetchedAt"] = at.Format(time.RFC3339Nano)
	result["unitPrice"] = price
	if time.Since(at) > maxAge || at.After(time.Now().Add(time.Minute)) {
		result["status"] = "stale"
		return result
	}
	value, err := investments.Value(p.Quantity, price)
	if err != nil {
		result["reason"] = "invalid valuation"
		return result
	}
	result["status"] = "reference"
	result["marketValue"] = strconv.FormatInt(value, 10)
	result["unrealized"] = strconv.FormatInt(value-p.Cost, 10)
	result["validUntil"] = at.Add(maxAge).Unix()
	return result
}
