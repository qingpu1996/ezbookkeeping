package models

// InvestmentPosition binds a quantity ledger to one monetary cost account.
// Investment tables are additive; legacy Account.Balance remains minor currency units.
type InvestmentPosition struct {
	Id              string `xorm:"VARCHAR(36) PK" json:"id"`
	Uid             int64  `xorm:"INDEX NOT NULL" json:"-"`
	Name            string `xorm:"VARCHAR(128) NOT NULL" json:"name"`
	AssetType       string `xorm:"VARCHAR(16) NOT NULL" json:"assetType"`
	Unit            string `xorm:"VARCHAR(16) NOT NULL" json:"unit"`
	Currency        string `xorm:"VARCHAR(3) NOT NULL" json:"currency"`
	CostAccountId   int64  `xorm:"UNIQUE NOT NULL" json:"costAccountId,string"`
	Quantity        string `xorm:"VARCHAR(40) NOT NULL" json:"quantity"`
	Cost            int64  `xorm:"NOT NULL" json:"cost,string"`
	Realized        int64  `xorm:"NOT NULL" json:"realized,string"`
	Algorithm       string `xorm:"VARCHAR(16) NOT NULL" json:"algorithm"`
	Version         int64  `xorm:"NOT NULL" json:"version"`
	CreatedUnixTime int64  `xorm:"NOT NULL" json:"createdAt"`
}

// InvestmentRevision is append-only. Current facts are the greatest Revision per OperationId.
type InvestmentRevision struct {
	Id                 string `xorm:"VARCHAR(36) PK" json:"id"`
	Uid                int64  `xorm:"INDEX NOT NULL" json:"-"`
	PositionId         string `xorm:"VARCHAR(36) INDEX NOT NULL" json:"positionId"`
	OperationId        string `xorm:"VARCHAR(36) UNIQUE(inv_revision) NOT NULL" json:"operationId"`
	Revision           int64  `xorm:"UNIQUE(inv_revision) NOT NULL" json:"revision"`
	Sequence           int64  `xorm:"NOT NULL" json:"sequence"`
	Kind               string `xorm:"VARCHAR(16) NOT NULL" json:"kind"`
	OccurredAt         int64  `xorm:"NOT NULL" json:"occurredAt"`
	TimezoneUtcOffset  int16  `xorm:"NOT NULL" json:"utcOffset"`
	Quantity           string `xorm:"VARCHAR(40) NOT NULL" json:"quantity"`
	Gross              int64  `xorm:"NOT NULL" json:"gross,string"`
	Fee                int64  `xorm:"NOT NULL" json:"fee,string"`
	CashAccountId      int64  `xorm:"NOT NULL" json:"cashAccountId,string"`
	TransferCategoryId int64  `xorm:"NOT NULL" json:"transferCategoryId,string"`
	IncomeCategoryId   int64  `xorm:"NOT NULL" json:"incomeCategoryId,string"`
	ExpenseCategoryId  int64  `xorm:"NOT NULL" json:"expenseCategoryId,string"`
	Cancelled          bool   `xorm:"NOT NULL" json:"cancelled"`
	Comment            string `xorm:"VARCHAR(255)" json:"comment"`
	RecordedAt         int64  `xorm:"NOT NULL" json:"recordedAt"`
}
type InvestmentPosting struct {
	Id                   string `xorm:"VARCHAR(36) PK"`
	Uid                  int64  `xorm:"INDEX NOT NULL"`
	PositionId           string `xorm:"VARCHAR(36) INDEX NOT NULL"`
	OperationId          string `xorm:"VARCHAR(36) NOT NULL"`
	RevisionId           string `xorm:"VARCHAR(36) NOT NULL"`
	TransactionId        int64  `xorm:"UNIQUE NOT NULL"`
	RelatedTransactionId int64  `xorm:"INDEX NOT NULL"`
	PositionVersion      int64  `xorm:"NOT NULL"`
	Active               bool   `xorm:"NOT NULL"`
}
type InvestmentCommand struct {
	Id          string `xorm:"VARCHAR(36) PK"`
	Uid         int64  `xorm:"UNIQUE(inv_command) NOT NULL"`
	RequestKey  string `xorm:"VARCHAR(64) UNIQUE(inv_command) NOT NULL"`
	PayloadHash string `xorm:"VARCHAR(64) NOT NULL"`
	Result      string `xorm:"TEXT NOT NULL"`
}

// InvestmentAttachment owns a receipt independently of regenerated cash postings.
type InvestmentAttachment struct {
	PictureId   int64  `xorm:"PK" json:"pictureId,string"`
	Uid         int64  `xorm:"INDEX NOT NULL" json:"-"`
	PositionId  string `xorm:"VARCHAR(36) INDEX NOT NULL" json:"positionId"`
	OperationId string `xorm:"VARCHAR(36) NOT NULL" json:"operationId"`
	RevisionId  string `xorm:"VARCHAR(36) NOT NULL" json:"revisionId"`
	Extension   string `xorm:"VARCHAR(10) NOT NULL" json:"extension"`
	RecordedAt  int64  `xorm:"NOT NULL" json:"recordedAt"`
}

// InvestmentValuationSetting changes display estimates only, never the cost ledger.
type InvestmentValuationSetting struct {
	PositionId    string `xorm:"VARCHAR(36) PK" json:"positionId"`
	Uid           int64  `xorm:"INDEX NOT NULL" json:"-"`
	Version       int64  `xorm:"NOT NULL" json:"version"`
	Mode          string `xorm:"VARCHAR(16) NOT NULL" json:"mode"`
	ManualPrice   string `xorm:"VARCHAR(40)" json:"manualPrice"`
	ManualAsOf    int64  `json:"manualAsOf"`
	MaxAgeMinutes int64  `json:"maxAgeMinutes"`
	UpdatedAt     int64  `json:"updatedAt"`
}
