package services

import (
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/uuid"
	"time"
	"xorm.io/xorm"
)

// A semantic purchase purpose, never a match on a user-editable category name.
const InvestmentPurchasePurpose = "asset_purchase"

// Called inside the position transaction. Preview rolls back categories and mapping too.
func (s *InvestmentService) purchaseFundingCategory(sess *xorm.Session, uid int64) (int64, error) {
	var mapping models.InvestmentPurchaseCategory
	found, err := sess.Where("uid=?", uid).Get(&mapping)
	if err != nil {
		return 0, err
	}
	if found {
		var category models.TransactionCategory
		ok, e := sess.Where("uid=? AND category_id=? AND deleted=? AND hidden=?", uid, mapping.CategoryId, false, false).Get(&category)
		if e != nil {
			return 0, e
		}
		if !ok || category.Type != models.CATEGORY_TYPE_TRANSFER || category.ParentCategoryId == 0 {
			return 0, errs.ErrInvestmentInvalid
		}
		var parent models.TransactionCategory
		ok, e = sess.Where("uid=? AND category_id=? AND deleted=? AND hidden=?", uid, category.ParentCategoryId, false, false).Get(&parent)
		if e != nil {
			return 0, e
		}
		if !ok || parent.Type != models.CATEGORY_TYPE_TRANSFER || parent.ParentCategoryId != 0 {
			return 0, errs.ErrInvestmentInvalid
		}
		return category.CategoryId, nil
	}
	now := time.Now().Unix()
	parent := &models.TransactionCategory{CategoryId: uuid.Container.GenerateUuid(uuid.UUID_TYPE_CATEGORY), Uid: uid, Type: models.CATEGORY_TYPE_TRANSFER, Name: "投资本金", Icon: 1, Color: "987633", CreatedUnixTime: now, UpdatedUnixTime: now}
	child := &models.TransactionCategory{CategoryId: uuid.Container.GenerateUuid(uuid.UUID_TYPE_CATEGORY), Uid: uid, Type: models.CATEGORY_TYPE_TRANSFER, ParentCategoryId: parent.CategoryId, Name: "资产买入", Icon: 1, Color: "987633", Comment: "资产买入本金转移；不计入消费支出", CreatedUnixTime: now, UpdatedUnixTime: now}
	if _, err = sess.Insert(parent, child); err != nil {
		return 0, err
	}
	if _, err = sess.Insert(&models.InvestmentPurchaseCategory{Uid: uid, CategoryId: child.CategoryId}); err != nil {
		return 0, err
	}
	return child.CategoryId, nil
}
