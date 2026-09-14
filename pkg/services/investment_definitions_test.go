package services

import (
	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/models"
	"testing"
	"time"
)

func TestDefinitionUnitsBindingPrecisionAndLedger(t *testing.T) {
	c := setupInvestmentTest(t)
	rows, err := Investments.Definitions(c, 1)
	if err != nil || len(rows) != 1 || rows[0].Unit != "g" {
		t.Fatal(rows, err)
	}
	definition := models.InvestmentDefinition{Id: investmentID(), Name: "测试黄金", Kind: "gold", Unit: "kg", UnitName: "千克", Precision: 3}
	saved, err := Investments.SaveDefinition(c, 1, definition)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Investments.SaveDefinition(c, 2, *saved); err != errs.ErrInvestmentNotFound {
		t.Fatal("foreign definition", err)
	}
	p, err := Investments.Create(c, 1, InvestmentCreateRequest{RequestKey: "definition-create", Name: "test", CostAccountId: 2, DefinitionId: saved.Id})
	if err != nil {
		t.Fatal(err)
	}
	if p.Unit != "kg" || p.UnitName != "千克" || p.QuantityPrecision != 3 || p.DefinitionId != saved.Id {
		t.Fatal(p)
	}
	changed := *saved
	changed.Unit = "g"
	if _, err = Investments.SaveDefinition(c, 1, changed); err != errs.ErrInvestmentConflict {
		t.Fatal("bound unit changed", err)
	}
	changed = *saved
	changed.Precision = 4
	if _, err = Investments.SaveDefinition(c, 1, changed); err != errs.ErrInvestmentConflict {
		t.Fatal("bound precision changed", err)
	}
	changed = *saved
	changed.Name = "黄金自定义名称"
	if _, err = Investments.SaveDefinition(c, 1, changed); err != nil {
		t.Fatal(err)
	}
	req := InvestmentOperationRequest{RequestKey: "definition-opening", PositionId: p.Id, ExpectedVersion: 1, Kind: "opening", OccurredAt: time.Now().Unix() - 10, Quantity: "0.1234", Gross: 10000}
	if _, err = Investments.Apply(c, 1, req, false); err != errs.ErrInvestmentInvalid {
		t.Fatal("precision accepted", err)
	}
	before, err := Investments.Detail(c, 1, p.Id)
	if err != nil || before.Position.Quantity != "0" || investmentBalance(t, c, 2) != 0 {
		t.Fatal("failed request wrote ledger", err)
	}
	req.Quantity = "0.123"
	d, err := Investments.Apply(c, 1, req, false)
	if err != nil || d.Position.Quantity != "0.123" || d.Position.Unit != "kg" {
		t.Fatal(d, err)
	}
	rows, err = Investments.Definitions(c, 1)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rows {
		if r.Id == saved.Id {
			found = r.InUse && r.Name == changed.Name
		}
	}
	if !found {
		t.Fatal(rows)
	}
}
func TestLegacyGoldDefinitionDoesNotRewritePosition(t *testing.T) {
	c := setupInvestmentTest(t)
	p, err := Investments.Create(c, 1, InvestmentCreateRequest{RequestKey: "legacy-definition", Name: "gold", CostAccountId: 2})
	if err != nil {
		t.Fatal(err)
	}
	// Emulate pre-definition position: remove only the synthetic test binding.
	sess := Investments.UserDataDB(1).NewSession(c)
	_, err = sess.Where("position_id=?", p.Id).Delete(&models.InvestmentDefinitionBinding{})
	sess.Close()
	if err != nil {
		t.Fatal(err)
	}
	d, err := Investments.Detail(c, 1, p.Id)
	if err != nil || d.Position.Unit != "g" || d.Position.QuantityPrecision != 12 || d.Position.Version != 1 {
		t.Fatal(d, err)
	}
	rows, err := Investments.Definitions(c, 1)
	if err != nil || !rows[0].InUse {
		t.Fatal(rows, err)
	}
	edit := *rows[0]
	edit.Name = "黄金"
	if _, err = Investments.SaveDefinition(c, 1, edit); err != nil {
		t.Fatal(err)
	}
	if investmentBalance(t, c, 2) != 0 {
		t.Fatal("definition posted cash")
	}
}
