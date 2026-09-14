package services

import (
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"testing"
	"time"
)

func TestMinuteScheduledTransfer(t *testing.T) {
	c := setupInvestmentTest(t)
	if err := datastore.Container.UserDataStore.SyncStructs(new(models.TransactionTemplate)); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 9, 15, 0, 0, 0, 0, time.FixedZone("CST", 480*60)).Unix()
	end := start + 86399
	tmpl := &models.TransactionTemplate{TemplateId: 99, Uid: 1, TemplateType: models.TRANSACTION_TEMPLATE_TYPE_SCHEDULE, Name: "test", Type: models.TRANSACTION_TYPE_TRANSFER, CategoryId: 11, AccountId: 1, RelatedAccountId: 2, Amount: 30000, RelatedAccountAmount: 30000, ScheduledFrequencyType: models.TRANSACTION_SCHEDULE_FREQUENCY_TYPE_DAILY, ScheduledFrequency: "1", ScheduledAt: 77, ScheduledTimezoneUtcOffset: 480, ScheduledStartTime: &start, ScheduledEndTime: &end}
	sess := Transactions.UserDataDB(1).NewSession(c)
	defer sess.Close()
	if _, err := sess.Insert(tmpl); err != nil {
		t.Fatal(err)
	}
	run := func(at int64) {
		if err := Transactions.CreateScheduledTransactions(c, at, time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	due := start + 557*60
	run(due - 86400)
	run(due - 60)
	if investmentBalance(t, c, 1) != 2000000 {
		t.Fatal("early transfer")
	}
	run(due)
	if investmentBalance(t, c, 1) != 1970000 || investmentBalance(t, c, 2) != 30000 {
		t.Fatal("transfer not balanced")
	}
	var rows []*models.Transaction
	if err := sess.Find(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatal("expected transfer pair", len(rows))
	}
	for _, r := range rows {
		if utils.GetUnixTimeFromTransactionTime(r.TransactionTime) != due {
			t.Fatal("wrong recorded time")
		}
	}
	run(due + 60)
	run(due + 86400)
	if investmentBalance(t, c, 1) != 1970000 {
		t.Fatal("outside window transfer")
	}
}
