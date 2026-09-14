package services

import (
	"encoding/json"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"sync"
	"testing"
)

func TestFundingPlanIsolationAndLedgerPreservation(t *testing.T) {
	c := setupInvestmentTest(t)
	if err := datastore.Container.UserDataStore.SyncStructs(new(models.FundingPlan)); err != nil {
		t.Fatal(err)
	}
	sess := FundingPlans.UserDataDB(1).NewSession(c)
	_, err := sess.Insert(&models.User{Uid: 2, Username: "other", Email: "other@example.invalid"}, &models.Account{AccountId: 3, Uid: 2, Type: 1, Category: 1, Currency: "CNY"}, &models.Account{AccountId: 4, Uid: 1, Type: 1, Category: 1, Currency: "USD"}, &models.Account{AccountId: 5, Uid: 1, Type: 1, Category: 3, Currency: "CNY"}, &models.Account{AccountId: 6, Uid: 1, Type: 2, Category: 1, Currency: "CNY"}, &models.InvestmentPosition{Id: "gold", Uid: 1, CostAccountId: 2})
	sess.Close()
	if err != nil {
		t.Fatal(err)
	}
	snapshot := func() string {
		sess := FundingPlans.UserDataDB(1).NewSession(c)
		defer sess.Close()
		a := []*models.Account{}
		tr := []*models.Transaction{}
		p := []*models.InvestmentPosition{}
		if err := sess.OrderBy("account_id").Find(&a); err != nil {
			t.Fatal(err)
		}
		if err := sess.Find(&tr); err != nil {
			t.Fatal(err)
		}
		if err := sess.Find(&p); err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal([]any{a, tr, p})
		return string(b)
	}
	before := snapshot()
	r, err := FundingPlans.Get(c, 1)
	if err != nil || r.Configured || r.Version != 0 {
		t.Fatal(r, err)
	}
	req := FundingPlanRequest{Currency: "CNY", Reserve: "1000000", AccountIDs: []string{"1"}}
	for _, id := range []string{"2", "3", "4", "5", "6", "999", "01"} {
		bad := req
		bad.AccountIDs = []string{id}
		if _, err := FundingPlans.Save(c, 1, bad); err != errs.ErrFundingPlanInvalid {
			t.Fatalf("invalid account %s: %v", id, err)
		}
	}
	for _, v := range []string{"", "-1", "1.2", "10000000000000", "01"} {
		bad := req
		bad.Reserve = v
		if _, err := FundingPlans.Save(c, 1, bad); err != errs.ErrFundingPlanInvalid {
			t.Fatal("reserve", v, err)
		}
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := FundingPlans.Save(c, 1, req); results <- e }()
	}
	wg.Wait()
	close(results)
	ok, conflict := 0, 0
	for e := range results {
		if e == nil {
			ok++
		} else if e == errs.ErrFundingPlanConflict {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatal(ok, conflict)
	}
	r, err = FundingPlans.Get(c, 1)
	if err != nil || !r.Configured || r.Reserve != req.Reserve || len(r.AccountIDs) != 1 {
		t.Fatal(r, err)
	}
	other, err := FundingPlans.Get(c, 2)
	if err != nil || other.Configured {
		t.Fatal("cross user", err)
	}
	req.Version = 1
	req.Reserve = "0"
	req.AccountIDs = []string{}
	r, err = FundingPlans.Save(c, 1, req)
	if err != nil || r.Reserve != "0" || len(r.AccountIDs) != 0 || r.Version != 2 {
		t.Fatal(r, err)
	}
	if snapshot() != before {
		t.Fatal("planning modified the ledger")
	}
}
