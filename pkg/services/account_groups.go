package services

import (
	"github.com/mayswind/ezbookkeeping/pkg/core"
	"github.com/mayswind/ezbookkeeping/pkg/datastore"
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"strings"
	"unicode/utf8"
	"xorm.io/xorm"
)

type AccountGroupService struct{ ServiceUsingDB }

var AccountGroups = &AccountGroupService{ServiceUsingDB{container: datastore.Container}}

type AccountGroupsResponse struct {
	Groups  []*models.AccountGroup       `json:"groups"`
	Members []*models.AccountGroupMember `json:"members"`
}
type AccountGroupRequest struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Version int64  `json:"version"`
}
type AccountGroupAssignRequest struct {
	AccountId       int64  `json:"accountId,string"`
	GroupId         string `json:"groupId"`
	ExpectedGroupId string `json:"expectedGroupId"`
}

func (s *AccountGroupService) List(c core.Context, uid int64) (*AccountGroupsResponse, error) {
	r := &AccountGroupsResponse{Groups: []*models.AccountGroup{}, Members: []*models.AccountGroupMember{}}
	err := s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		// Use the same per-user lock as writes so a list cannot straddle group deletion.
		if err := lockInvestmentOwner(sess, uid); err != nil {
			return err
		}
		if err := sess.Where("uid=?", uid).OrderBy("name, id").Find(&r.Groups); err != nil {
			return err
		}
		return sess.Where("uid=?", uid).OrderBy("account_id").Find(&r.Members)
	})
	return r, err
}
func ownedGroup(sess *xorm.Session, uid int64, id string) (*models.AccountGroup, error) {
	g := &models.AccountGroup{}
	found, err := sess.ID(id).Where("uid=?", uid).Get(g)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errs.ErrAccountGroupNotFound
	}
	return g, nil
}
func (s *AccountGroupService) Save(c core.Context, uid int64, req AccountGroupRequest) (*models.AccountGroup, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" || utf8.RuneCountInString(name) > 64 {
		return nil, errs.ErrAccountGroupInvalid
	}
	var result *models.AccountGroup
	err := s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		if err := lockInvestmentOwner(sess, uid); err != nil {
			return err
		}
		var current *models.AccountGroup
		if req.Id != "" {
			var err error
			current, err = ownedGroup(sess, uid, req.Id)
			if err != nil {
				return err
			}
			if current.Version != req.Version {
				return errs.ErrAccountGroupConflict
			}
		}
		rows := []*models.AccountGroup{}
		if err := sess.Where("uid=?", uid).Find(&rows); err != nil {
			return err
		}
		for _, g := range rows {
			if g.Id != req.Id && strings.EqualFold(g.Name, name) {
				return errs.ErrAccountGroupConflict
			}
		}
		if current == nil {
			result = &models.AccountGroup{Id: investmentID(), Uid: uid, Name: name, Version: 1}
			_, err := sess.Insert(result)
			return err
		}
		current.Name = name
		current.Version++
		result = current
		_, err := sess.ID(current.Id).Where("uid=?", uid).Cols("name", "version").Update(current)
		return err
	})
	return result, err
}
func (s *AccountGroupService) Delete(c core.Context, uid int64, req AccountGroupRequest) error {
	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		if err := lockInvestmentOwner(sess, uid); err != nil {
			return err
		}
		g, err := ownedGroup(sess, uid, req.Id)
		if err != nil {
			return err
		}
		if g.Version != req.Version {
			return errs.ErrAccountGroupConflict
		}
		if _, err = sess.Where("uid=? AND group_id=?", uid, g.Id).Delete(&models.AccountGroupMember{}); err != nil {
			return err
		}
		_, err = sess.ID(g.Id).Where("uid=?", uid).Delete(&models.AccountGroup{})
		return err
	})
}
func (s *AccountGroupService) Assign(c core.Context, uid int64, req AccountGroupAssignRequest) error {
	if req.AccountId <= 0 {
		return errs.ErrAccountGroupInvalid
	}
	return s.UserDataDB(uid).DoTransaction(c, func(sess *xorm.Session) error {
		if err := lockInvestmentOwner(sess, uid); err != nil {
			return err
		}
		var account models.Account
		found, err := sess.ID(req.AccountId).Where("uid=? AND deleted=?", uid, false).Get(&account)
		if err != nil {
			return err
		}
		if !found || account.ParentAccountId != 0 {
			return errs.ErrAccountGroupInvalid
		}
		if req.GroupId != "" {
			if _, err := ownedGroup(sess, uid, req.GroupId); err != nil {
				return err
			}
		}
		var old models.AccountGroupMember
		found, err = sess.ID(req.AccountId).Where("uid=?", uid).Get(&old)
		if err != nil {
			return err
		}
		current := ""
		if found {
			current = old.GroupId
		}
		if current != req.ExpectedGroupId {
			return errs.ErrAccountGroupConflict
		}
		if current == req.GroupId {
			return nil
		}
		if _, err = sess.ID(req.AccountId).Where("uid=?", uid).Delete(&models.AccountGroupMember{}); err != nil {
			return err
		}
		if req.GroupId != "" {
			_, err = sess.Insert(&models.AccountGroupMember{AccountId: req.AccountId, Uid: uid, GroupId: req.GroupId})
		}
		return err
	})
}
