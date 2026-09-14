package services

import (
	"encoding/json"
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"regexp"
	"strconv"
	"xorm.io/xorm"
)

type FundingPlanService struct{ ServiceUsingDB }

var FundingPlans = &FundingPlanService{ServiceUsingDB{container: datastore.Container}}

type FundingPlanRequest struct {
	Currency   string   `json:"currency"`
	Reserve    string   `json:"reserve"`
	AccountIDs []string `json:"accountIds"`
	Version    int64    `json:"version"`
}
type FundingPlanResponse struct {
	Configured bool `json:"configured"`
	FundingPlanRequest
}

func readFundingPlan(sess *xorm.Session, uid int64) (*FundingPlanResponse, error) {
	row := &models.FundingPlan{}
	found, err := sess.ID(uid).Get(row)
	if err != nil {
		return nil, err
	}
	r := &FundingPlanResponse{Configured: found, FundingPlanRequest: FundingPlanRequest{AccountIDs: []string{}}}
	if found {
		r.Currency = row.Currency
		r.Reserve = strconv.FormatInt(row.Reserve, 10)
		r.Version = row.Version
		if err = json.Unmarshal([]byte(row.AccountIDsJSON), &r.AccountIDs); err != nil {
			return nil, err
		}
	}
	return r, nil
}
func (s *FundingPlanService) Get(c core.Context, uid int64) (*FundingPlanResponse, error) {
	sess := s.UserDataDB(uid).NewSession(c)
	defer sess.Close()
	return readFundingPlan(sess, uid)
}
func (s *FundingPlanService) Save(c core.Context, uid int64, req FundingPlanRequest) (*FundingPlanResponse, error) {
	if !regexp.MustCompile(`^[A-Z]{3}$`).MatchString(req.Currency) || !regexp.MustCompile(`^(0|[1-9][0-9]{0,12})$`).MatchString(req.Reserve) || len(req.AccountIDs) > 1000 || req.Version < 0 {
		return nil, errs.ErrFundingPlanInvalid
	}
	reserve, err := strconv.ParseInt(req.Reserve, 10, 64)
	if err != nil {
		return nil, errs.ErrFundingPlanInvalid
	}
	var result *FundingPlanResponse
	err = s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		if err := lockInvestmentOwner(sess, uid); err != nil {
			return err
		}
		old, err := readFundingPlan(sess, uid)
		if err != nil {
			return err
		}
		if old.Version != req.Version {
			return errs.ErrFundingPlanConflict
		}
		seen := map[int64]bool{}
		for _, id := range req.AccountIDs {
			aid, err := strconv.ParseInt(id, 10, 64)
			if err != nil || aid <= 0 || seen[aid] || strconv.FormatInt(aid, 10) != id {
				return errs.ErrFundingPlanInvalid
			}
			seen[aid] = true
			var a models.Account
			found, err := sess.ID(aid).Where("uid=? AND deleted=?", uid, false).Get(&a)
			if err != nil {
				return err
			}
			if !found || a.Type != models.ACCOUNT_TYPE_SINGLE_ACCOUNT || !a.Category.IsAsset() || a.Currency != req.Currency {
				return errs.ErrFundingPlanInvalid
			}
			has, err := sess.Where("uid=? AND cost_account_id=?", uid, aid).Exist(&models.InvestmentPosition{})
			if err != nil {
				return err
			}
			if has {
				return errs.ErrFundingPlanInvalid
			}
		}
		if req.AccountIDs == nil {
			req.AccountIDs = []string{}
		}
		encoded, err := json.Marshal(req.AccountIDs)
		if err != nil {
			return err
		}
		row := &models.FundingPlan{Uid: uid, Currency: req.Currency, Reserve: reserve, AccountIDsJSON: string(encoded), Version: old.Version + 1}
		if old.Configured {
			_, err = sess.ID(uid).AllCols().Update(row)
		} else {
			_, err = sess.Insert(row)
		}
		if err != nil {
			return err
		}
		result, err = readFundingPlan(sess, uid)
		return err
	})
	return result, err
}
