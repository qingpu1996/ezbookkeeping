package services

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/investments"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
	"time"
	"xorm.io/xorm"
)

type InvestmentValuationRequest struct {
	PositionId    string `json:"positionId"`
	Version       int64  `json:"version"`
	Mode          string `json:"mode"`
	ManualPrice   string `json:"manualPrice"`
	ManualAsOf    int64  `json:"manualAsOf"`
	MaxAgeMinutes int64  `json:"maxAgeMinutes"`
}

func (s *InvestmentService) ValuationSettings(c core.Context, uid int64, id string) (*models.InvestmentValuationSetting, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}
	sess := s.UserDataDB(uid).NewSession(c)
	defer sess.Close()
	has, err := sess.Where("uid=? AND id=?", uid, id).Exist(&models.InvestmentPosition{})
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, errs.ErrInvestmentNotFound
	}
	row := &models.InvestmentValuationSetting{PositionId: id, Uid: uid, Mode: "cost", MaxAgeMinutes: 1440}
	_, err = sess.Where("uid=? AND position_id=?", uid, id).Get(row)
	return row, err
}
func (s *InvestmentService) SaveValuationSettings(c core.Context, uid int64, req InvestmentValuationRequest) (*models.InvestmentValuationSetting, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}
	if req.Mode != "cost" && req.Mode != "manual" && req.Mode != "automatic" {
		return nil, errs.ErrInvestmentInvalid
	}
	if req.Mode == "automatic" && settings.Container.GetCurrentConfig().InvestmentQuoteURL == "" {
		return nil, errs.ErrInvestmentInvalid
	}
	if req.Mode == "manual" {
		if _, err := investments.Value("1", req.ManualPrice); err != nil {
			return nil, errs.ErrInvestmentInvalid
		}
		if req.ManualAsOf <= 0 || req.ManualAsOf > time.Now().Unix()+60 || req.MaxAgeMinutes < 1 || req.MaxAgeMinutes > 10080 {
			return nil, errs.ErrInvestmentInvalid
		}
	} else {
		req.ManualPrice = ""
		req.ManualAsOf = 0
		req.MaxAgeMinutes = 1440
	}
	var result *models.InvestmentValuationSetting
	err := s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		if err := lockInvestmentOwner(sess, uid); err != nil {
			return err
		}
		has, err := sess.Where("uid=? AND id=?", uid, req.PositionId).Exist(&models.InvestmentPosition{})
		if err != nil {
			return err
		}
		if !has {
			return errs.ErrInvestmentNotFound
		}
		var old models.InvestmentValuationSetting
		has, err = sess.Where("uid=? AND position_id=?", uid, req.PositionId).Get(&old)
		if err != nil {
			return err
		}
		if old.Version != req.Version {
			return errs.ErrInvestmentConflict
		}
		result = &models.InvestmentValuationSetting{PositionId: req.PositionId, Uid: uid, Version: req.Version + 1, Mode: req.Mode, ManualPrice: req.ManualPrice, ManualAsOf: req.ManualAsOf, MaxAgeMinutes: req.MaxAgeMinutes, UpdatedAt: time.Now().Unix()}
		if !has {
			_, err = sess.Insert(result)
		} else {
			_, err = sess.Where("uid=? AND position_id=?", uid, req.PositionId).AllCols().Update(result)
		}
		return err
	})
	return result, err
}
