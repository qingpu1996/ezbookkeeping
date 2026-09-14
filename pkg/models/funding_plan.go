package models

// FundingPlan is planning metadata. It never posts transactions or changes balances.
// AccountIDsJSON contains explicitly selected leaf accounts; new accounts are excluded.
type FundingPlan struct {
	Uid            int64  `xorm:"PK" json:"-"`
	Currency       string `xorm:"VARCHAR(3) NOT NULL" json:"currency"`
	Reserve        int64  `xorm:"NOT NULL" json:"reserve,string"`
	AccountIDsJSON string `xorm:"TEXT NOT NULL" json:"-"`
	Version        int64  `xorm:"NOT NULL" json:"version"`
}
