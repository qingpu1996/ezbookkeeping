package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/investments"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/utils"
	"xorm.io/xorm"
)

type InvestmentService struct {
	ServiceUsingDB
	transactions *TransactionService
}

var Investments = &InvestmentService{ServiceUsingDB: ServiceUsingDB{container: datastore.Container}, transactions: Transactions}

// The capability is private and cannot be supplied through any HTTP request.
type investmentWriteContext struct{ core.Context }

func investmentID() string {
	b := randomInvestmentBytes()
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
func randomInvestmentBytes() []byte {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return b
}

type InvestmentCreateRequest struct {
	DefinitionId  string `json:"definitionId,omitempty"`
	RequestKey    string `json:"requestKey"`
	Name          string `json:"name"`
	CostAccountId int64  `json:"costAccountId,string"`
}
type InvestmentOperationRequest struct {
	EntryPurpose       string `json:"entryPurpose,omitempty"`
	AssetDefinitionId  string `json:"assetDefinitionId,omitempty"`
	RequestKey         string `json:"requestKey"`
	PositionId         string `json:"positionId"`
	ExpectedVersion    int64  `json:"expectedVersion"`
	OperationId        string `json:"operationId"` // empty: new; populated: correction or cancellation
	Kind               string `json:"kind"`
	OccurredAt         int64  `json:"occurredAt"`
	TimezoneUtcOffset  int16  `json:"utcOffset"`
	Quantity           string `json:"quantity"`
	Gross              int64  `json:"gross,string"`
	Fee                int64  `json:"fee,string"`
	CashAccountId      int64  `json:"cashAccountId,string"`
	TransferCategoryId int64  `json:"transferCategoryId,string"`
	IncomeCategoryId   int64  `json:"incomeCategoryId,string"`
	ExpenseCategoryId  int64  `json:"expenseCategoryId,string"`
	Cancelled          bool   `json:"cancelled"`
	Comment            string `json:"comment"`
}
type InvestmentDetail struct {
	Attachments []*models.InvestmentAttachment `json:"attachments"`
	History     []*models.InvestmentRevision   `json:"history,omitempty"`
	Position    *models.InvestmentPosition     `json:"position"`
	Operations  []*models.InvestmentRevision   `json:"operations"`
	Calculation investments.Result             `json:"calculation"`
}

func commandHash(kind string, req any) string {
	b, _ := json.Marshal(req)
	sum := sha256.Sum256(append([]byte(kind), b...))
	return hex.EncodeToString(sum[:])
}
func loadInvestmentCommand(sess *xorm.Session, uid int64, key, hash string, out any) (bool, error) {
	if uid <= 0 || len(key) < 8 || len(key) > 64 || strings.TrimSpace(key) != key {
		return false, errs.ErrInvestmentInvalid
	}
	var row models.InvestmentCommand
	has, err := sess.Where("uid=? AND request_key=?", uid, key).Get(&row)
	if err != nil || !has {
		return false, err
	}
	if row.PayloadHash != hash {
		return false, errs.ErrInvestmentConflict
	}
	return true, json.Unmarshal([]byte(row.Result), out)
}
func saveInvestmentCommand(sess *xorm.Session, uid int64, key, hash string, out any) error {
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	_, err = sess.Insert(&models.InvestmentCommand{Id: investmentID(), Uid: uid, RequestKey: key, PayloadHash: hash, Result: string(b)})
	return err
}
func (s *InvestmentService) List(c core.Context, uid int64) ([]*models.InvestmentPosition, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}
	rows := []*models.InvestmentPosition{}
	sess := s.UserDataDB(uid).NewSession(c)
	defer sess.Close()
	err := sess.Where("uid=?", uid).OrderBy("created_unix_time, id").Find(&rows)
	if err == nil {
		for _, p := range rows {
			if err = decorateInvestment(sess, p); err != nil {
				break
			}
		}
	}
	return rows, err
}
func (s *InvestmentService) Create(c core.Context, uid int64, req InvestmentCreateRequest) (*models.InvestmentPosition, error) {
	if uid <= 0 || strings.TrimSpace(req.Name) == "" || len(req.Name) > 128 || req.CostAccountId <= 0 {
		return nil, errs.ErrInvestmentInvalid
	}
	var result models.InvestmentPosition
	hash := commandHash("create", req)
	err := s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		if err := lockInvestmentOwner(sess, uid); err != nil {
			return err
		}
		found, err := loadInvestmentCommand(sess, uid, req.RequestKey, hash, &result)
		if err != nil || found {
			return err
		}
		definitionId := req.DefinitionId
		if definitionId == "" {
			definitionId = legacyGoldDefinition(uid).Id
		}
		definition, err := definitionInSession(sess, uid, definitionId)
		if err != nil {
			return err
		}
		// No-op balance update takes a database write lock without changing the account.
		_, err = sess.ID(req.CostAccountId).Where("uid=? AND deleted=?", uid, false).SetExpr("balance", "balance").Cols("balance").Update(&models.Account{})
		if err != nil {
			return err
		}
		var account models.Account
		has, err := sess.ID(req.CostAccountId).Where("uid=? AND deleted=?", uid, false).Get(&account)
		if err != nil {
			return err
		}
		if !has || account.Hidden || account.Type != models.ACCOUNT_TYPE_SINGLE_ACCOUNT || account.Category != models.ACCOUNT_CATEGORY_INVESTMENT || account.Currency != "CNY" || account.Balance != 0 {
			return errs.ErrInvestmentInvalid
		}
		exists, err := sess.Where("uid=? AND deleted=? AND (account_id=? OR related_account_id=?)", uid, false, account.AccountId, account.AccountId).Exist(&models.Transaction{})
		if err != nil {
			return err
		}
		if exists {
			return errs.ErrInvestmentInvalid
		}
		result = models.InvestmentPosition{Id: investmentID(), Uid: uid, Name: strings.TrimSpace(req.Name), AssetType: definition.Kind, Unit: definition.Unit, Currency: account.Currency, CostAccountId: account.AccountId, Quantity: "0", Algorithm: investments.Algorithm, Version: 1, CreatedUnixTime: time.Now().Unix()}
		if _, err = sess.Insert(&result); err != nil {
			return err
		}
		if _, err = sess.Insert(&models.InvestmentDefinitionBinding{PositionId: result.Id, Uid: uid, DefinitionId: definition.Id}); err != nil {
			return err
		}
		if err = decorateInvestment(sess, &result); err != nil {
			return err
		}
		return saveInvestmentCommand(sess, uid, req.RequestKey, hash, &result)
	})
	return &result, err
}
func currentInvestmentRevisions(sess *xorm.Session, uid int64, position string) ([]*models.InvestmentRevision, error) {
	rows := []*models.InvestmentRevision{}
	if err := sess.Where("uid=? AND position_id=?", uid, position).OrderBy("revision desc").Find(&rows); err != nil {
		return nil, err
	}
	current := []*models.InvestmentRevision{}
	seen := map[string]bool{}
	for _, row := range rows {
		if !seen[row.OperationId] {
			seen[row.OperationId] = true
			current = append(current, row)
		}
	}
	sort.Slice(current, func(i, j int) bool {
		if current[i].OccurredAt == current[j].OccurredAt {
			return current[i].Sequence < current[j].Sequence
		}
		return current[i].OccurredAt < current[j].OccurredAt
	})
	return current, nil
}
func replayInvestment(rows []*models.InvestmentRevision) (investments.Result, error) {
	es := make([]investments.Event, 0, len(rows))
	for _, r := range rows {
		es = append(es, investments.Event{ID: r.OperationId, Kind: r.Kind, OccurredAt: r.OccurredAt, Sequence: r.Sequence, Quantity: r.Quantity, Gross: r.Gross, Fee: r.Fee, Cancelled: r.Cancelled})
	}
	return investments.Replay(es)
}
func (s *InvestmentService) Detail(c core.Context, uid int64, id string) (*InvestmentDetail, error) {
	var result InvestmentDetail
	err := s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		p := new(models.InvestmentPosition)
		has, err := sess.ID(id).Where("uid=?", uid).Get(p)
		if err != nil {
			return err
		}
		if !has {
			return errs.ErrInvestmentNotFound
		}
		if err := decorateInvestment(sess, p); err != nil {
			return err
		}
		rows, err := currentInvestmentRevisions(sess, uid, id)
		if err != nil {
			return err
		}
		calc, err := replayInvestment(rows)
		result = InvestmentDetail{Position: p, Operations: rows, Calculation: calc}
		if err != nil {
			return err
		}
		if err = sess.Where("uid=? AND position_id=?", uid, id).Find(&result.Attachments); err != nil {
			return err
		}
		if err = sess.Where("uid=? AND position_id=?", uid, id).OrderBy("recorded_at, revision, id").Find(&result.History); err != nil {
			return err
		}
		var latest models.InvestmentPosition
		has, err = sess.ID(id).Where("uid=?", uid).Get(&latest)
		if err != nil {
			return err
		}
		if !has || latest.Version != p.Version || calc.Quantity != p.Quantity || calc.Cost != p.Cost || calc.Realized != p.Realized {
			return errs.ErrInvestmentConflict
		}
		return nil
	})
	return &result, err
}
func prepareInvestmentRevision(uid int64, p *models.InvestmentPosition, rows []*models.InvestmentRevision, req InvestmentOperationRequest) ([]*models.InvestmentRevision, *models.InvestmentRevision, error) {
	if req.ExpectedVersion != p.Version {
		return nil, nil, errs.ErrInvestmentConflict
	}
	if req.OccurredAt <= 0 || req.OccurredAt > time.Now().Unix() || req.TimezoneUtcOffset < -720 || req.TimezoneUtcOffset > 840 || len(req.Comment) > 255 {
		return nil, nil, errs.ErrInvestmentInvalid
	}
	q, err := investments.Quantity(req.Quantity)
	if err != nil {
		return nil, nil, errs.ErrInvestmentInvalid
	}
	next := &models.InvestmentRevision{Id: investmentID(), Uid: uid, PositionId: p.Id, OperationId: investmentID(), Revision: 1, Sequence: p.Version, Kind: req.Kind, OccurredAt: req.OccurredAt, TimezoneUtcOffset: req.TimezoneUtcOffset, Quantity: investments.FormatQuantity(q), Gross: req.Gross, Fee: req.Fee, CashAccountId: req.CashAccountId, TransferCategoryId: req.TransferCategoryId, IncomeCategoryId: req.IncomeCategoryId, ExpenseCategoryId: req.ExpenseCategoryId, Cancelled: req.Cancelled, Comment: req.Comment, RecordedAt: time.Now().Unix()}
	if req.OperationId != "" {
		found := false
		for i, old := range rows {
			if old.OperationId == req.OperationId {
				if old.Cancelled {
					return nil, nil, errs.ErrInvestmentInvalid
				}
				next.OperationId = old.OperationId
				next.Sequence = old.Sequence
				next.Revision = old.Revision + 1
				if req.Cancelled {
					*next = *old
					next.Id = investmentID()
					next.Revision++
					next.Cancelled = true
					next.RecordedAt = time.Now().Unix()
				}
				rows[i] = next
				found = true
				break
			}
		}
		if !found {
			return nil, nil, errs.ErrInvestmentNotFound
		}
	} else {
		if req.Cancelled {
			return nil, nil, errs.ErrInvestmentInvalid
		}
		rows = append(rows, next)
	}
	if next.Kind != "opening" && (next.CashAccountId <= 0 || next.CashAccountId == p.CostAccountId) {
		return nil, nil, errs.ErrInvestmentInvalid
	}
	if next.Kind == "opening" && next.CashAccountId != 0 {
		return nil, nil, errs.ErrInvestmentInvalid
	}
	return rows, next, nil
}

var errInvestmentPreviewRollback = errors.New("investment preview rollback")

// Apply preview and save share validation and calculation. Preview always rolls back locks.
func (s *InvestmentService) Apply(c core.Context, uid int64, req InvestmentOperationRequest, preview bool) (*InvestmentDetail, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}
	var result InvestmentDetail
	if req.EntryPurpose != "" && req.EntryPurpose != InvestmentPurchasePurpose {
		return nil, errs.ErrInvestmentInvalid
	}
	if req.EntryPurpose == InvestmentPurchasePurpose && (req.Kind != "buy" || req.OperationId != "" || req.Cancelled || req.TransferCategoryId != 0 || req.AssetDefinitionId == "") {
		return nil, errs.ErrInvestmentInvalid
	}
	hash := commandHash("apply", req)
	db := s.UserDataDB(uid)
	err := db.DoTransaction(c, func(sess *xorm.Session) error {
		if err := lockInvestmentOwner(sess, uid); err != nil {
			return err
		}
		if !preview {
			found, err := loadInvestmentCommand(sess, uid, req.RequestKey, hash, &result)
			if err != nil || found {
				return err
			}
		}
		// A compare-and-swap write serializes all operations on this position across processes.
		n, err := sess.ID(req.PositionId).Where("uid=? AND version=?", uid, req.ExpectedVersion).SetExpr("version", "version+1").Cols("version").Update(&models.InvestmentPosition{})
		if err != nil {
			return err
		}
		if n != 1 {
			return errs.ErrInvestmentConflict
		}
		p := new(models.InvestmentPosition)
		has, err := sess.ID(req.PositionId).Where("uid=?", uid).Get(p)
		if err != nil {
			return err
		}
		if !has {
			return errs.ErrInvestmentNotFound
		}
		if err := decorateInvestment(sess, p); err != nil {
			return err
		}
		if !req.Cancelled {
			if err := validateDefinitionQuantity(req.Quantity, p.QuantityPrecision); err != nil {
				return err
			}
		}
		if req.EntryPurpose == InvestmentPurchasePurpose {
			if req.AssetDefinitionId != p.DefinitionId {
				return errs.ErrInvestmentInvalid
			}
			categoryId, e := s.purchaseFundingCategory(sess, uid)
			if e != nil {
				return e
			}
			req.TransferCategoryId = categoryId
		}
		p.Version-- // provisional CAS increment remains invisible until commit
		rows, err := currentInvestmentRevisions(sess, uid, p.Id)
		if err != nil {
			return err
		}
		oldCalculation, err := replayInvestment(rows)
		if err != nil {
			return err
		}
		previousRows := append([]*models.InvestmentRevision(nil), rows...)
		rows, next, err := prepareInvestmentRevision(uid, p, rows, req)
		if err != nil {
			return err
		}
		calc, err := replayInvestment(rows)
		if err != nil {
			return errs.ErrInvestmentInvalid
		}
		// Lock monetary accounts in stable order; validate ownership, currency and balance bounds.
		accountIDs := map[int64]bool{p.CostAccountId: true}
		for _, r := range append(previousRows, rows...) {
			if r.CashAccountId > 0 {
				accountIDs[r.CashAccountId] = true
			}
		}
		ids := make([]int64, 0, len(accountIDs))
		for id := range accountIDs {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids {
			if _, err = sess.ID(id).Where("uid=? AND deleted=?", uid, false).SetExpr("balance", "balance").Cols("balance").Update(&models.Account{}); err != nil {
				return err
			}
			var a models.Account
			has, err := sess.ID(id).Where("uid=? AND deleted=?", uid, false).Get(&a)
			if err != nil {
				return err
			}
			if !has || a.Hidden || a.Currency != p.Currency || a.Type != models.ACCOUNT_TYPE_SINGLE_ACCOUNT {
				return errs.ErrInvestmentInvalid
			}
			if id == p.CostAccountId && a.Balance != p.Cost {
				return errs.ErrInvestmentConflict
			}
			if id != p.CostAccountId {
				if err = guardInvestmentAccounts(sess, uid, []int64{id}); err != nil {
					return err
				}
			}
		}
		result = InvestmentDetail{Position: p, Operations: rows, Calculation: calc}

		changed := map[string]bool{next.OperationId: true}
		oldEffects := map[string]investments.Effect{}
		for _, effect := range oldCalculation.Effects {
			oldEffects[effect.EventID] = effect
		}
		for _, effect := range calc.Effects {
			old, has := oldEffects[effect.EventID]
			if !has || old.CashDelta != effect.CashDelta || old.AllocatedCost != effect.AllocatedCost || old.Realized != effect.Realized {
				changed[effect.EventID] = true
			}
		}
		internal := investmentWriteContext{c}
		oldPostings := []*models.InvestmentPosting{}
		if err = sess.Where("uid=? AND position_id=? AND active=?", uid, p.Id, true).OrderBy("position_version desc, transaction_id desc").Find(&oldPostings); err != nil {
			return err
		}
		for _, link := range oldPostings {
			if !changed[link.OperationId] {
				continue
			}
			if err = s.transactions.deleteTransactionInSession(internal, sess, uid, link.TransactionId); err != nil {
				return err
			}
		}

		changedIDs := []string{}
		for id := range changed {
			changedIDs = append(changedIDs, id)
		}
		if _, err = sess.Where("uid=? AND position_id=? AND active=?", uid, p.Id, true).In("operation_id", changedIDs).Cols("active").Update(&models.InvestmentPosting{Active: false}); err != nil {
			return err
		}
		if _, err = sess.Insert(next); err != nil {
			return err
		}
		byID := map[string]*models.InvestmentRevision{}
		for _, r := range rows {
			byID[r.OperationId] = r
		}
		for _, effect := range calc.Effects {
			if !changed[effect.EventID] {
				continue
			}
			r := byID[effect.EventID]
			post := func(kind models.TransactionDbType, from, to, amount, category int64) error {
				if amount == 0 {
					return nil
				}
				t := &models.Transaction{Uid: uid, Type: kind, AccountId: from, RelatedAccountId: to, Amount: amount, CategoryId: category, TransactionTime: utils.GetMinTransactionTimeFromUnixTime(r.OccurredAt), TimezoneUtcOffset: r.TimezoneUtcOffset, Comment: "[investment] " + p.Name}

				// Reserve a later internal slot before balance-opening validation.
				// Multiple business operations may share the same displayed second.
				var latest models.Transaction
				has, err := sess.Where("uid=? AND transaction_time>=? AND transaction_time<=?", uid, utils.GetMinTransactionTimeFromUnixTime(r.OccurredAt), utils.GetMaxTransactionTimeFromUnixTime(r.OccurredAt)).OrderBy("transaction_time desc").Get(&latest)
				if err != nil {
					return err
				}
				if has {
					t.TransactionTime = latest.TransactionTime + 1
				}
				if t.TransactionTime > utils.GetMaxTransactionTimeFromUnixTime(r.OccurredAt)-1 {
					return errs.ErrCannotCreateTransactionWithThisTransactionTime
				}
				if kind == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
					t.RelatedAccountAmount = amount
				}
				if err := s.transactions.createTransactionInSession(internal, sess, t, nil, nil); err != nil {
					return err
				}
				_, err = sess.Insert(&models.InvestmentPosting{Id: investmentID(), Uid: uid, PositionId: p.Id, OperationId: r.OperationId, RevisionId: r.Id, TransactionId: t.TransactionId, RelatedTransactionId: t.RelatedId, PositionVersion: p.Version + 1, Active: true})
				return err
			}
			switch r.Kind {
			case "opening":
				err = post(models.TRANSACTION_DB_TYPE_MODIFY_BALANCE, p.CostAccountId, 0, r.Gross, 0)
			case "buy":
				err = post(models.TRANSACTION_DB_TYPE_TRANSFER_OUT, r.CashAccountId, p.CostAccountId, -effect.CashDelta, r.TransferCategoryId)
			case "sell":
				err = post(models.TRANSACTION_DB_TYPE_TRANSFER_OUT, p.CostAccountId, r.CashAccountId, effect.AllocatedCost, r.TransferCategoryId)
				if err == nil && effect.Realized > 0 {
					err = post(models.TRANSACTION_DB_TYPE_INCOME, r.CashAccountId, 0, effect.Realized, r.IncomeCategoryId)
				}
				if err == nil && effect.Realized < 0 {
					err = post(models.TRANSACTION_DB_TYPE_EXPENSE, r.CashAccountId, 0, -effect.Realized, r.ExpenseCategoryId)
				}
			}
			if err != nil {
				return err
			}
		}
		for _, id := range ids {
			var a models.Account
			_, err = sess.ID(id).Where("uid=?", uid).Get(&a)
			if err != nil {
				return err
			}
			if a.Balance > investments.MaxMoney || a.Balance < -investments.MaxMoney {
				return errs.ErrInvestmentInvalid
			}
			if id == p.CostAccountId && a.Balance != calc.Cost {
				return errs.ErrInvestmentConflict
			}
		}
		p.Version++
		p.Quantity = calc.Quantity
		p.Cost = calc.Cost
		p.Realized = calc.Realized
		if _, err = sess.ID(p.Id).Where("uid=?", uid).Cols("version", "quantity", "cost", "realized").Update(p); err != nil {
			return err
		}
		if preview {
			return errInvestmentPreviewRollback
		}
		return saveInvestmentCommand(sess, uid, req.RequestKey, hash, &result)
	})
	if errors.Is(err, errInvestmentPreviewRollback) {
		err = nil
	}

	if err != nil && !preview {
		session := s.UserDataDB(uid).NewSession(c)
		found, readErr := loadInvestmentCommand(session, uid, req.RequestKey, hash, &result)
		session.Close()
		if readErr == nil && found {
			return &result, nil
		}
		if readErr == errs.ErrInvestmentConflict {
			return nil, readErr
		}
	}
	return &result, err
}

// Attach binds an already uploaded, user-owned receipt. Retrying the same binding is safe.
func (s *InvestmentService) Attach(c core.Context, uid int64, positionID, operationID string, pictureID int64) error {
	if uid <= 0 || pictureID <= 0 {
		return errs.ErrInvestmentInvalid
	}
	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		if err := lockInvestmentOwner(sess, uid); err != nil {
			return err
		}
		var existing models.InvestmentAttachment
		has, err := sess.ID(pictureID).Get(&existing)
		if err != nil {
			return err
		}
		if has {
			if existing.Uid == uid && existing.PositionId == positionID && existing.OperationId == operationID {
				return nil
			}
			return errs.ErrInvestmentInvalid
		}
		var p models.InvestmentPosition
		has, err = sess.ID(positionID).Where("uid=?", uid).Get(&p)
		if err != nil {
			return err
		}
		if !has {
			return errs.ErrInvestmentNotFound
		}
		if _, err = sess.ID(positionID).Where("uid=?", uid).SetExpr("version", "version").Cols("version").Update(&models.InvestmentPosition{}); err != nil {
			return err
		}
		var revision models.InvestmentRevision
		has, err = sess.Where("uid=? AND position_id=? AND operation_id=?", uid, positionID, operationID).OrderBy("revision desc").Get(&revision)
		if err != nil {
			return err
		}
		if !has || revision.Cancelled {
			return errs.ErrInvestmentInvalid
		}
		count, err := sess.Where("uid=? AND operation_id=?", uid, operationID).Count(&models.InvestmentAttachment{})
		if err != nil {
			return err
		}
		if count >= 5 {
			return errs.ErrTransactionHasTooManyPictures
		}
		var picture models.TransactionPictureInfo
		has, err = sess.ID(pictureID).Where("uid=? AND deleted=? AND transaction_id=?", uid, false, models.TransactionPictureNewPictureTransactionId).Get(&picture)
		if err != nil {
			return err
		}
		if !has {
			return errs.ErrTransactionPictureNotFound
		}
		// -1 means owned by an investment receipt, never an unused upload or a cash posting.
		n, err := sess.ID(pictureID).Where("uid=? AND deleted=? AND transaction_id=?", uid, false, 0).Cols("transaction_id").Update(&models.TransactionPictureInfo{TransactionId: -1})
		if err != nil {
			return err
		}
		if n != 1 {
			return errs.ErrInvestmentConflict
		}
		_, err = sess.Insert(&models.InvestmentAttachment{PictureId: pictureID, Uid: uid, PositionId: positionID, OperationId: operationID, RevisionId: revision.Id, Extension: picture.PictureExtension, RecordedAt: time.Now().Unix()})
		return err
	})
}
