package services

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func setupInvestmentTest(t *testing.T) core.Context {
	t.Helper()
	config := &settings.Config{DatabaseConfig: &settings.DatabaseConfig{DatabaseType: settings.Sqlite3DbType, DatabasePath: filepath.Join(t.TempDir(), "ledger.db"), MaxOpenConnection: 1, MaxIdleConnection: 1}, UuidGeneratorType: settings.InternalUuidGeneratorType}
	if os.Getenv("EBK_TEST_POSTGRES") == "1" {
		config.DatabaseConfig = &settings.DatabaseConfig{DatabaseType: settings.PostgresDbType, DatabaseHost: "ebk-investment-test-pg:5432", DatabaseName: "ebk_investment_test", DatabaseUser: "postgres", DatabaseSSLMode: "disable", MaxOpenConnection: 5, MaxIdleConnection: 2}
	}
	if os.Getenv("EBK_TEST_MYSQL") == "1" {
		config.DatabaseConfig = &settings.DatabaseConfig{DatabaseType: settings.MySqlDbType, DatabaseHost: "ebk-investment-test-mysql:3306", DatabaseName: "ebk_investment_test", DatabaseUser: "root", MaxOpenConnection: 5, MaxIdleConnection: 2}
	}
	if err := datastore.InitializeDataStore(config); err != nil {
		t.Fatal(err)
	}
	if err := uuid.InitializeUuidGenerator(config); err != nil {
		t.Fatal(err)
	}
	if err := datastore.Container.UserDataStore.SyncStructs(new(models.User), new(models.Account), new(models.Transaction), new(models.TransactionCategory), new(models.TransactionTagIndex), new(models.TransactionTag), new(models.TransactionPictureInfo), new(models.InvestmentPosition), new(models.InvestmentRevision), new(models.InvestmentPosting), new(models.InvestmentCommand), new(models.InvestmentAttachment)); err != nil {
		t.Fatal(err)
	}
	c := core.NewNullContext()
	sess := Investments.UserDataDB(1).NewSession(c)
	defer sess.Close()
	if _, err := sess.Insert(&models.User{Uid: 1, Username: "test", Email: "test@example.invalid"}); err != nil {
		t.Fatal(err)
	}
	_, err := sess.Insert(&models.Account{AccountId: 1, Uid: 1, Name: "cash", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Category: models.ACCOUNT_CATEGORY_CASH, Currency: "CNY", Balance: 2000000}, &models.Account{AccountId: 2, Uid: 1, Name: "gold", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Category: models.ACCOUNT_CATEGORY_INVESTMENT, Currency: "CNY"})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []struct {
		id, parent int64
		kind       models.TransactionCategoryType
	}{{10, 0, models.CATEGORY_TYPE_TRANSFER}, {11, 10, models.CATEGORY_TYPE_TRANSFER}, {20, 0, models.CATEGORY_TYPE_INCOME}, {21, 20, models.CATEGORY_TYPE_INCOME}, {30, 0, models.CATEGORY_TYPE_EXPENSE}, {31, 30, models.CATEGORY_TYPE_EXPENSE}} {
		if _, err = sess.Insert(&models.TransactionCategory{CategoryId: r.id, Uid: 1, ParentCategoryId: r.parent, Type: r.kind, Name: "test"}); err != nil {
			t.Fatal(err)
		}
	}
	return c
}
func investmentBalance(t *testing.T, c core.Context, id int64) int64 {
	t.Helper()
	sess := Investments.UserDataDB(1).NewSession(c)
	defer sess.Close()
	var a models.Account
	has, err := sess.ID(id).Get(&a)
	if err != nil || !has {
		t.Fatalf("balance: %v", err)
	}
	return a.Balance
}
func TestInvestmentAtomicLifecycle(t *testing.T) {
	c := setupInvestmentTest(t)
	p, err := Investments.Create(c, 1, InvestmentCreateRequest{RequestKey: "create-0001", Name: "test gold", CostAccountId: 2})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix() - 1000
	req := InvestmentOperationRequest{RequestKey: "opening-0001", PositionId: p.Id, ExpectedVersion: 1, Kind: "opening", OccurredAt: now, Quantity: "10", Gross: 900000}
	detail, err := Investments.Apply(c, 1, req, false)
	if err != nil {
		t.Fatal(err)
	}
	if investmentBalance(t, c, 1) != 2000000 || investmentBalance(t, c, 2) != 900000 {
		t.Fatal("opening debited cash")
	}
	req = InvestmentOperationRequest{RequestKey: "buy-00000001", PositionId: p.Id, ExpectedVersion: detail.Position.Version, Kind: "buy", OccurredAt: now, Quantity: "5", Gross: 480000, Fee: 500, CashAccountId: 1, TransferCategoryId: 11, IncomeCategoryId: 21, ExpenseCategoryId: 31}
	preview, err := Investments.Apply(c, 1, req, true)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Calculation.Cost != 1380500 || investmentBalance(t, c, 2) != 900000 {
		t.Fatal("preview changed ledger")
	}
	bought, err := Investments.Apply(c, 1, req, false)
	if err != nil {
		t.Fatal(err)
	}
	repeat, err := Investments.Apply(c, 1, req, false)
	if err != nil || repeat.Position.Version != bought.Position.Version {
		t.Fatalf("idempotency %v", err)
	}
	req.Gross++
	if _, err = Investments.Apply(c, 1, req, false); err != errs.ErrInvestmentConflict {
		t.Fatalf("key conflict: %v", err)
	}
	req.Gross--
	if investmentBalance(t, c, 1) != 1519500 || investmentBalance(t, c, 2) != 1380500 {
		t.Fatal("buy balances")
	}
	buyID := ""
	for _, r := range bought.Operations {
		if r.Kind == "buy" {
			buyID = r.OperationId
		}
	}
	req.RequestKey = "sell-0000001"
	req.ExpectedVersion = bought.Position.Version
	req.Kind = "sell"
	req.OccurredAt = now + 20
	req.Quantity = "3"
	req.Gross = 300000
	sold, err := Investments.Apply(c, 1, req, false)
	if err != nil {
		t.Fatal(err)
	}
	if sold.Position.Quantity != "12" || sold.Position.Cost != 1104400 || sold.Position.Realized != 23400 || investmentBalance(t, c, 1) != 1819000 {
		t.Fatalf("sell: %+v", sold.Position)
	}
	// Correction replays all subsequent allocations while preserving audit revisions.
	req.RequestKey = "correct-0001"
	req.OperationId = buyID
	req.ExpectedVersion = sold.Position.Version
	req.Kind = "buy"
	req.OccurredAt = now + 10
	req.Quantity = "5"
	req.Gross = 510000
	corrected, err := Investments.Apply(c, 1, req, false)
	if err != nil {
		t.Fatal(err)
	}
	if corrected.Position.Cost != 1128400 || corrected.Position.Realized != 17400 || investmentBalance(t, c, 1) != 1789000 {
		t.Fatalf("correction: %+v", corrected.Position)
	}
	// Failure after deleting old postings must roll back both old and new ledger changes.
	req.RequestKey = "failure-0001"
	req.OperationId = buyID
	req.ExpectedVersion = corrected.Position.Version
	req.Kind = "buy"
	req.OccurredAt = now + 30
	req.TransferCategoryId = 999
	if _, err = Investments.Apply(c, 1, req, false); err == nil {
		t.Fatal("invalid category accepted")
	}
	after, err := Investments.Detail(c, 1, p.Id)
	if err != nil || after.Position.Version != corrected.Position.Version || investmentBalance(t, c, 1) != 1789000 || investmentBalance(t, c, 2) != 1128400 {
		t.Fatalf("rollback failed: %v", err)
	}
	// Legacy create/modify/delete must not bypass the quantity ledger.
	tx := &models.Transaction{Uid: 1, AccountId: 1, RelatedAccountId: 2, Type: models.TRANSACTION_DB_TYPE_TRANSFER_OUT, Amount: 100, RelatedAccountAmount: 100, CategoryId: 11, TransactionTime: utils.GetMinTransactionTimeFromUnixTime(now + 40)}
	if err = Transactions.CreateTransaction(c, tx, nil, nil); err != errs.ErrInvestmentProtected {
		t.Fatalf("legacy create: %v", err)
	}
	sess := Investments.UserDataDB(1).NewSession(c)
	var link models.InvestmentPosting
	_, err = sess.Where("uid=? AND active=?", 1, true).Get(&link)
	sess.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err = Transactions.DeleteTransaction(c, 1, link.TransactionId); err != errs.ErrInvestmentProtected {
		t.Fatalf("legacy delete: %v", err)
	}
	if err = Transactions.DeleteAllTransactions(c, 1, false); err != errs.ErrInvestmentProtected {
		t.Fatalf("clear: %v", err)
	}

	// Receipt ownership survives correction/cancellation; unused-upload cleanup cannot remove it.
	sess = Investments.UserDataDB(1).NewSession(c)
	_, err = sess.Insert(&models.TransactionPictureInfo{Uid: 1, PictureId: 9001, PictureExtension: "png"}, &models.TransactionPictureInfo{Uid: 2, PictureId: 9002, PictureExtension: "png"})
	sess.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err = Investments.Attach(c, 1, p.Id, buyID, 9001); err != nil {
		t.Fatal(err)
	}
	if err = Investments.Attach(c, 1, p.Id, buyID, 9001); err != nil {
		t.Fatal("receipt retry", err)
	}
	if err = Investments.Attach(c, 1, p.Id, buyID, 9002); err == nil {
		t.Fatal("foreign receipt accepted")
	}
	if err = TransactionPictures.RemoveUnusedTransactionPicture(c, 1, 9001); err == nil {
		t.Fatal("owned receipt removed as unused")
	}
	// Two concurrent identical requests recover the same persisted result.
	concurrent := InvestmentOperationRequest{RequestKey: "concurrent-0001", PositionId: p.Id, ExpectedVersion: corrected.Position.Version, Kind: "sell", OccurredAt: now + 100, Quantity: "10", Gross: 1000000, CashAccountId: 1, TransferCategoryId: 11, IncomeCategoryId: 21, ExpenseCategoryId: 31}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := Investments.Apply(c, 1, concurrent, false); results <- e }()
	}
	wg.Wait()
	close(results)
	for e := range results {
		if e != nil {
			t.Fatalf("concurrent retry: %v", e)
		}
	}
	current, e := Investments.Detail(c, 1, p.Id)
	if e != nil || current.Position.Quantity != "2" {
		t.Fatalf("concurrent oversell: %+v %v", current, e)
	}
	if len(current.Attachments) != 1 {
		t.Fatal("receipt lost")
	}
	concurrent.RequestKey = "oversell-0001"
	concurrent.ExpectedVersion = current.Position.Version
	if _, e = Investments.Apply(c, 1, concurrent, false); e == nil {
		t.Fatal("oversell accepted")
	}
	// Different commands based on the same position version cannot both sell.
	results = make(chan error, 2)
	for _, key := range []string{"different-sale-a", "different-sale-b"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			r := concurrent
			r.RequestKey = key
			r.Quantity = "1.5"
			r.Gross = 150000
			_, e := Investments.Apply(c, 1, r, false)
			results <- e
		}(key)
	}
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for e := range results {
		if e == nil {
			successes++
		} else if e == errs.ErrInvestmentConflict {
			conflicts++
		} else {
			t.Fatal(e)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent distinct commands: %d success, %d conflict", successes, conflicts)
	}
	current, e = Investments.Detail(c, 1, p.Id)
	if e != nil || current.Position.Quantity != "0.5" {
		t.Fatalf("distinct sale quantity: %v %v", current, e)
	}
	beforeVersion := current.Position.Version
	cancel := concurrent
	cancel.RequestKey = "invalid-cancel-buy"
	cancel.ExpectedVersion = beforeVersion
	cancel.OperationId = buyID
	cancel.Cancelled = true
	if _, e = Investments.Apply(c, 1, cancel, false); e == nil {
		t.Fatal("cancellation causing historical oversell accepted")
	}
	current, e = Investments.Detail(c, 1, p.Id)
	if e != nil || current.Position.Version != beforeVersion || len(current.Attachments) != 1 {
		t.Fatal("failed cancellation altered ledger or receipt")
	}
	if _, err = Investments.Detail(c, 2, p.Id); err != errs.ErrInvestmentNotFound {
		t.Fatalf("cross-user: %v", err)
	}
}
