package models

// AccountGroup is presentation metadata only, never a balance-bearing account.
type AccountGroup struct {
	Id      string `xorm:"VARCHAR(36) PK" json:"id"`
	Uid     int64  `xorm:"INDEX NOT NULL" json:"-"`
	Name    string `xorm:"VARCHAR(64) NOT NULL" json:"name"`
	Version int64  `xorm:"NOT NULL" json:"version"`
}

// A root account belongs to at most one group; its existing children inherit it.
type AccountGroupMember struct {
	AccountId int64  `xorm:"PK" json:"accountId,string"`
	Uid       int64  `xorm:"INDEX NOT NULL" json:"-"`
	GroupId   string `xorm:"VARCHAR(36) INDEX NOT NULL" json:"groupId"`
}
