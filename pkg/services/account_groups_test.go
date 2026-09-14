package services

import (
	"encoding/json"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"sync"
	"testing"
)

func TestAccountGroupsIsolationAndLedgerPreservation(t *testing.T) {
	c := setupInvestmentTest(t)
	if err := datastore.Container.UserDataStore.SyncStructs(new(models.AccountGroup), new(models.AccountGroupMember)); err != nil {
		t.Fatal(err)
	}
	sess := AccountGroups.UserDataDB(1).NewSession(c)
	_, err := sess.Insert(&models.User{Uid: 2, Username: "other", Email: "other@example.invalid"}, &models.Account{AccountId: 3, Uid: 1, Name: "parent", Type: models.ACCOUNT_TYPE_MULTI_SUB_ACCOUNTS, Category: models.ACCOUNT_CATEGORY_CHECKING_ACCOUNT}, &models.Account{AccountId: 4, Uid: 1, ParentAccountId: 3, Name: "child", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT, Category: models.ACCOUNT_CATEGORY_CHECKING_ACCOUNT, Currency: "USD", Balance: 123}, &models.Account{AccountId: 5, Uid: 2, Name: "foreign", Type: models.ACCOUNT_TYPE_SINGLE_ACCOUNT})
	sess.Close()
	if err != nil {
		t.Fatal(err)
	}
	snapshot := func() string {
		sess := AccountGroups.UserDataDB(1).NewSession(c)
		defer sess.Close()
		rows := []*models.Account{}
		if err := sess.OrderBy("account_id").Find(&rows); err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(rows)
		return string(b)
	}
	before := snapshot()
	g, err := AccountGroups.Save(c, 1, AccountGroupRequest{Name: " 招商银行 "})
	if err != nil {
		t.Fatal(err)
	}
	if g.Name != "招商银行" {
		t.Fatal("name not normalized")
	}
	if _, err := AccountGroups.Save(c, 1, AccountGroupRequest{Name: "招商银行"}); err != errs.ErrAccountGroupConflict {
		t.Fatalf("duplicate: %v", err)
	}
	foreign, err := AccountGroups.Save(c, 2, AccountGroupRequest{Name: "招商银行"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AccountGroups.Save(c, 1, AccountGroupRequest{Id: foreign.Id, Name: "bad", Version: 1}); err != errs.ErrAccountGroupNotFound {
		t.Fatal("cross-user rename", err)
	}
	if err := AccountGroups.Assign(c, 1, AccountGroupAssignRequest{AccountId: 1, GroupId: foreign.Id}); err != errs.ErrAccountGroupNotFound {
		t.Fatal("cross-user group", err)
	}
	for _, id := range []int64{4, 5, 999} {
		if err := AccountGroups.Assign(c, 1, AccountGroupAssignRequest{AccountId: id, GroupId: g.Id}); err != errs.ErrAccountGroupInvalid {
			t.Fatalf("invalid/foreign/child assignment %d: %v", id, err)
		}
	}
	for _, id := range []int64{1, 2, 3} {
		if err := AccountGroups.Assign(c, 1, AccountGroupAssignRequest{AccountId: id, GroupId: g.Id}); err != nil {
			t.Fatal(err)
		}
	}
	if err := AccountGroups.Assign(c, 1, AccountGroupAssignRequest{AccountId: 1, GroupId: "", ExpectedGroupId: ""}); err != errs.ErrAccountGroupConflict {
		t.Fatal("stale move", err)
	}
	renamed, err := AccountGroups.Save(c, 1, AccountGroupRequest{Id: g.Id, Name: "招行", Version: g.Version})
	if err != nil {
		t.Fatal(err)
	}
	if err := AccountGroups.Delete(c, 1, AccountGroupRequest{Id: g.Id, Version: g.Version}); err != errs.ErrAccountGroupConflict {
		t.Fatal("stale delete", err)
	}
	if err := AccountGroups.Delete(c, 2, AccountGroupRequest{Id: g.Id, Version: renamed.Version}); err != errs.ErrAccountGroupNotFound {
		t.Fatal("cross-user delete", err)
	}
	list, err := AccountGroups.List(c, 1)
	if err != nil || len(list.Groups) != 1 || len(list.Members) != 3 {
		t.Fatalf("list: %+v %v", list, err)
	}
	if err := AccountGroups.Delete(c, 1, AccountGroupRequest{Id: g.Id, Version: renamed.Version}); err != nil {
		t.Fatal(err)
	}
	list, err = AccountGroups.List(c, 1)
	if err != nil || len(list.Groups) != 0 || len(list.Members) != 0 {
		t.Fatal("delete leaves membership", err)
	}
	if snapshot() != before {
		t.Fatal("group operation changed ledger accounts")
	}
	list, err = AccountGroups.List(c, 2)
	if err != nil || len(list.Groups) != 1 {
		t.Fatal("other user affected", err)
	}
	// Two devices moving an ungrouped account: only one conditional assignment wins.
	a, _ := AccountGroups.Save(c, 1, AccountGroupRequest{Name: "A"})
	b, _ := AccountGroups.Save(c, 1, AccountGroupRequest{Name: "B"})
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, id := range []string{a.Id, b.Id} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			results <- AccountGroups.Assign(c, 1, AccountGroupAssignRequest{AccountId: 1, GroupId: id})
		}(id)
	}
	wg.Wait()
	close(results)
	ok, conflicts := 0, 0
	for err := range results {
		if err == nil {
			ok++
		} else if err == errs.ErrAccountGroupConflict {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || conflicts != 1 {
		t.Fatalf("lost update: %d %d", ok, conflicts)
	}
	if snapshot() != before {
		t.Fatal("concurrent grouping changed accounts")
	}
}
