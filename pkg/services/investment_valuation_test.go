package services

import (
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"testing"
	"time"
)

func TestValuationSettingsIsolateAndPreserveLedger(t *testing.T) {
	c := setupInvestmentTest(t)
	if err := datastore.Container.UserDataStore.SyncStructs(new(models.InvestmentValuationSetting)); err != nil {
		t.Fatal(err)
	}
	p, err := Investments.Create(c, 1, InvestmentCreateRequest{RequestKey: "valuation-create-01", Name: "test", CostAccountId: 2})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := Investments.Detail(c, 1, p.Id)
	initial, err := Investments.ValuationSettings(c, 1, p.Id)
	if err != nil || initial.Mode != "cost" || initial.Version != 0 {
		t.Fatal("legacy default", err)
	}
	req := InvestmentValuationRequest{PositionId: p.Id, Mode: "manual", ManualPrice: "1000.1234", ManualAsOf: time.Now().Unix(), MaxAgeMinutes: 1440}
	if _, err = Investments.SaveValuationSettings(c, 2, req); err == nil {
		t.Fatal("foreign owner saved settings")
	}
	if _, err = Investments.ValuationSettings(c, 2, p.Id); err != errs.ErrInvestmentNotFound {
		t.Fatal("foreign owner read settings", err)
	}
	result, err := Investments.SaveValuationSettings(c, 1, req)
	if err != nil || result.Version != 1 {
		t.Fatal("save", err)
	}
	if _, err = Investments.SaveValuationSettings(c, 1, req); err != errs.ErrInvestmentConflict {
		t.Fatal("stale update", err)
	}
	req.Version = 1
	for _, price := range []string{"0", "NaN", "-1", "1e5"} {
		bad := req
		bad.ManualPrice = price
		if _, err = Investments.SaveValuationSettings(c, 1, bad); err == nil {
			t.Fatal("invalid price", price)
		}
	}
	for _, age := range []int64{0, 10081} {
		bad := req
		bad.MaxAgeMinutes = age
		if _, err = Investments.SaveValuationSettings(c, 1, bad); err == nil {
			t.Fatal("invalid age")
		}
	}
	req.Mode = "cost"
	if _, err = Investments.SaveValuationSettings(c, 1, req); err != nil {
		t.Fatal(err)
	}
	after, _ := Investments.Detail(c, 1, p.Id)
	if after.Position.Version != before.Position.Version || after.Position.Cost != before.Position.Cost || len(after.Operations) != 0 || investmentBalance(t, c, 1) != 2000000 || investmentBalance(t, c, 2) != 0 {
		t.Fatal("valuation changed ledger")
	}
}
