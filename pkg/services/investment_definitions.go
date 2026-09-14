package services

import (
	"fmt"
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"regexp"
	"strings"
	"xorm.io/xorm"
)

// Legacy accounts resolve to a stable, per-owner definition without rewriting their ledger.
func legacyGoldDefinition(uid int64) *models.InvestmentDefinition {
	return &models.InvestmentDefinition{Id: fmt.Sprintf("gold-%d", uid), Uid: uid, Name: "黄金", Kind: "gold", Unit: "g", UnitName: "克", Precision: 12}
}
func definitionInSession(s *xorm.Session, uid int64, id string) (*models.InvestmentDefinition, error) {
	var d models.InvestmentDefinition
	has, err := s.Where("uid=? AND id=?", uid, id).Get(&d)
	if err != nil {
		return nil, err
	}
	if has {
		return &d, nil
	}
	legacy := legacyGoldDefinition(uid)
	if id == legacy.Id {
		return legacy, nil
	}
	return nil, errs.ErrInvestmentNotFound
}
func decorateInvestment(s *xorm.Session, p *models.InvestmentPosition) error {
	var b models.InvestmentDefinitionBinding
	has, err := s.Where("uid=? AND position_id=?", p.Uid, p.Id).Get(&b)
	if err != nil {
		return err
	}
	id := legacyGoldDefinition(p.Uid).Id
	if has {
		id = b.DefinitionId
	}
	d, err := definitionInSession(s, p.Uid, id)
	if err != nil {
		return err
	}
	if d.Unit != p.Unit || d.Kind != p.AssetType {
		return errs.ErrInvestmentInvalid
	}
	p.DefinitionId = d.Id
	p.UnitName = d.UnitName
	p.QuantityPrecision = d.Precision
	return nil
}
func definitionUsed(s *xorm.Session, uid int64, id string) (bool, error) {
	if id == legacyGoldDefinition(uid).Id {
		return s.Where("uid=?", uid).Exist(&models.InvestmentPosition{})
	}
	return s.Where("uid=? AND definition_id=?", uid, id).Exist(&models.InvestmentDefinitionBinding{})
}
func (s *InvestmentService) Definitions(c core.Context, uid int64) ([]*models.InvestmentDefinition, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}
	sess := s.UserDataDB(uid).NewSession(c)
	defer sess.Close()
	rows := []*models.InvestmentDefinition{}
	if err := sess.Where("uid=?", uid).OrderBy("name,id").Find(&rows); err != nil {
		return nil, err
	}
	legacy := legacyGoldDefinition(uid)
	found := false
	for _, d := range rows {
		if d.Id == legacy.Id {
			found = true
		}
	}
	if !found {
		rows = append([]*models.InvestmentDefinition{legacy}, rows...)
	}
	for _, d := range rows {
		used, err := definitionUsed(sess, uid, d.Id)
		if err != nil {
			return nil, err
		}
		d.InUse = used
	}
	return rows, nil
}
func (s *InvestmentService) SaveDefinition(c core.Context, uid int64, req models.InvestmentDefinition) (*models.InvestmentDefinition, error) {
	if uid <= 0 || req.Id == "" || len(req.Id) > 36 || strings.TrimSpace(req.Name) == "" || len(req.Name) > 128 || req.Kind != "gold" || !regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,15}$`).MatchString(req.Unit) || strings.TrimSpace(req.UnitName) == "" || len(req.UnitName) > 32 || req.Precision < 0 || req.Precision > 12 {
		return nil, errs.ErrInvestmentInvalid
	}
	var result *models.InvestmentDefinition
	err := s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		if err := lockInvestmentOwner(sess, uid); err != nil {
			return err
		}
		var old models.InvestmentDefinition
		has, err := sess.Where("id=?", req.Id).Get(&old)
		if err != nil {
			return err
		}
		if has && old.Uid != uid {
			return errs.ErrInvestmentNotFound
		}
		if !has && strings.HasPrefix(req.Id, "gold-") && req.Id != legacyGoldDefinition(uid).Id {
			return errs.ErrInvestmentNotFound
		}
		if !has && req.Id == legacyGoldDefinition(uid).Id {
			old = *legacyGoldDefinition(uid)
		}
		if req.Version != old.Version {
			return errs.ErrInvestmentConflict
		}
		used, err := definitionUsed(sess, uid, req.Id)
		if err != nil {
			return err
		}
		if used && (old.Kind != req.Kind || old.Unit != req.Unit || old.UnitName != req.UnitName || old.Precision != req.Precision) {
			return errs.ErrInvestmentConflict
		}
		req.Uid = uid
		req.Name = strings.TrimSpace(req.Name)
		req.UnitName = strings.TrimSpace(req.UnitName)
		req.Version++
		req.InUse = used
		if has {
			_, err = sess.Where("uid=? AND id=?", uid, req.Id).AllCols().Update(&req)
		} else {
			_, err = sess.Insert(&req)
		}
		result = &req
		return err
	})
	return result, err
}
func validateDefinitionQuantity(quantity string, precision int) error {
	parts := strings.Split(quantity, ".")
	if len(parts) > 2 {
		return errs.ErrInvestmentInvalid
	}
	if len(parts) == 2 && len(strings.TrimRight(parts[1], "0")) > precision {
		return errs.ErrInvestmentInvalid
	}
	return nil
}
