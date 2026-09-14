package services

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"sort"
	"xorm.io/xorm"
)

func guardInvestmentAccounts(sess *xorm.Session, uid int64, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	// Include parents: deleting or changing a parent must not orphan a managed child.
	positions := []*models.InvestmentPosition{}
	if err := sess.Where("uid=?", uid).Find(&positions); err != nil {
		return err
	}
	wanted := map[int64]bool{}
	for _, id := range ids {
		if id > 0 {
			wanted[id] = true
		}
	}
	for _, p := range positions {
		if wanted[p.CostAccountId] {
			return errs.ErrInvestmentProtected
		}
		var a models.Account
		has, err := sess.ID(p.CostAccountId).Where("uid=?", uid).Get(&a)
		if err != nil {
			return err
		}
		if has && wanted[a.ParentAccountId] {
			return errs.ErrInvestmentProtected
		}
	}
	return nil
}
func guardInvestmentTransaction(sess *xorm.Session, uid int64, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	rows := []*models.Transaction{}
	if err := sess.Where("uid=?", uid).In("transaction_id", ids).Find(&rows); err != nil {
		return err
	}
	accounts := []int64{}
	for _, r := range rows {
		accounts = append(accounts, r.AccountId, r.RelatedAccountId)
	}
	if err := guardInvestmentAccounts(sess, uid, accounts); err != nil {
		return err
	}
	// Realized P/L legs touch only the cash account; they are protected by their links.
	has, err := sess.Where("uid=?", uid).In("transaction_id", ids).Exist(&models.InvestmentPosting{})
	if err != nil {
		return err
	}
	if has {
		return errs.ErrInvestmentProtected
	}
	has, err = sess.Where("uid=?", uid).In("related_transaction_id", ids).Exist(&models.InvestmentPosting{})
	if err != nil {
		return err
	}
	if has {
		return errs.ErrInvestmentProtected
	}
	return nil
}
func (s *InvestmentService) GuardAccounts(c core.Context, uid int64, ids []int64) error {
	sess := s.UserDataDB(uid).NewSession(c)
	defer sess.Close()
	return guardInvestmentAccounts(sess, uid, ids)
}
func (s *InvestmentService) GuardTransactions(c core.Context, uid int64, ids []int64) error {
	sess := s.UserDataDB(uid).NewSession(c)
	defer sess.Close()
	return guardInvestmentTransaction(sess, uid, ids)
}
func (s *InvestmentService) GuardClear(c core.Context, uid int64) error {
	sess := s.UserDataDB(uid).NewSession(c)
	defer sess.Close()
	has, err := sess.Where("uid=?", uid).Exist(&models.InvestmentPosition{})
	if err != nil {
		return err
	}
	if has {
		return errs.ErrInvestmentProtected
	}
	return nil
}

// Locks synchronize ordinary posting and position binding. Call inside the writing transaction.
func guardInvestmentPosting(c core.Context, sess *xorm.Session, t *models.Transaction) error {
	if err := lockInvestmentOwner(sess, t.Uid); err != nil {
		return err
	}
	if _, internal := c.(investmentWriteContext); internal {
		return nil
	}
	ids := []int64{t.AccountId}
	if t.RelatedAccountId > 0 && t.RelatedAccountId != t.AccountId {
		ids = append(ids, t.RelatedAccountId)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		if _, err := sess.ID(id).Where("uid=? AND deleted=?", t.Uid, false).SetExpr("balance", "balance").Cols("balance").Update(&models.Account{}); err != nil {
			return err
		}
	}
	if err := guardInvestmentAccounts(sess, t.Uid, ids); err != nil {
		return err
	}
	return guardInvestmentTransaction(sess, t.Uid, []int64{t.TransactionId, t.RelatedId})
}

// Bulk moves/deletes of a cash account also include investment P/L postings.
func (s *InvestmentService) GuardAccountTransactions(c core.Context, uid int64, ids []int64) error {
	sess := s.UserDataDB(uid).NewSession(c)
	defer sess.Close()
	rows := []*models.Transaction{}
	if err := sess.Where("uid=? AND deleted=?", uid, false).In("account_id", ids).Find(&rows); err != nil {
		return err
	}
	txIDs := []int64{}
	for _, row := range rows {
		txIDs = append(txIDs, row.TransactionId)
	}
	// Chunk only the preflight query; no mutation occurs until every chunk passes.
	for start := 0; start < len(txIDs); start += 200 {
		end := start + 200
		if end > len(txIDs) {
			end = len(txIDs)
		}
		if err := guardInvestmentTransaction(sess, uid, txIDs[start:end]); err != nil {
			return err
		}
	}
	return guardInvestmentAccounts(sess, uid, ids)
}

// The current datastore initializes user and ledger tables in the same database.
// A per-user row lock serializes binding/clearing and avoids a newly bound account
// being cleared between a preflight query and a bulk write.
func lockInvestmentOwner(sess *xorm.Session, uid int64) error {
	_, err := sess.ID(uid).SetExpr("updated_unix_time", "updated_unix_time").Cols("updated_unix_time").Update(&models.User{})
	return err
}
