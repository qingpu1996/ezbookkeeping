package investments

import (
	"reflect"
	"testing"
)

func TestGoldenExample(t *testing.T) {
	events := []Event{{ID: "o", Kind: "opening", OccurredAt: 1, Quantity: "10", Gross: 900000}, {ID: "b", Kind: "buy", OccurredAt: 2, Quantity: "5", Gross: 480000, Fee: 500}, {ID: "s", Kind: "sell", OccurredAt: 3, Quantity: "3", Gross: 300000, Fee: 500}}
	r, e := Replay(events)
	if e != nil {
		t.Fatal(e)
	}
	if r.Quantity != "12" || r.Cost != 1104400 || r.Realized != 23400 || r.Effects[2].AllocatedCost != 276100 {
		t.Fatalf("%+v", r)
	}
	events[1].Gross = 510000
	corrected, e := Replay(events)
	if e != nil {
		t.Fatal(e)
	}
	if corrected.Cost != 1128400 || corrected.Realized != 17400 {
		t.Fatalf("correction: %+v", corrected)
	}
	events[1].Cancelled = true
	events[2].Quantity = "11"
	if _, e = Replay(events); e == nil {
		t.Fatal("historical oversell accepted")
	}
}
func TestRoundingAndFullDisposal(t *testing.T) {
	for cost := int64(0); cost < 201; cost++ {
		es := []Event{{ID: "o", Kind: "opening", OccurredAt: 1, Quantity: "3", Gross: cost}, {ID: "a", Kind: "sell", OccurredAt: 2, Quantity: "1", Gross: 100}, {ID: "b", Kind: "sell", OccurredAt: 3, Quantity: "1", Gross: 100}, {ID: "c", Kind: "sell", OccurredAt: 4, Quantity: "1", Gross: 100}}
		r, e := Replay(es)
		if e != nil {
			t.Fatal(e)
		}
		if r.Cost != 0 || r.Quantity != "0" || r.Realized != 300-cost {
			t.Fatalf("cost=%d: %+v", cost, r)
		}
		allocated := int64(0)
		for _, x := range r.Effects {
			allocated += x.AllocatedCost
		}
		if allocated != cost {
			t.Fatal("cost not conserved")
		}
	}
}
func TestDecimalValidation(t *testing.T) {
	for _, s := range []string{"NaN", "Inf", "-1", "+1", "1e3", "01", ".1", "1.", "1.0000000000001", " 1", "123456789012345678901234567890123456789"} {
		if _, e := Quantity(s); e == nil {
			t.Fatalf("accepted %q", s)
		}
	}
	for s, want := range map[string]string{"0": "0", "1.0000": "1", "0.000000000001": "0.000000000001", "12345678901234567890123456.123456789012": "12345678901234567890123456.123456789012"} {
		q, e := Quantity(s)
		if e != nil || FormatQuantity(q) != want {
			t.Fatalf("%s: %v", s, e)
		}
	}
}
func TestInvalidOperations(t *testing.T) {
	base := Event{ID: "b", Kind: "buy", OccurredAt: 1, Quantity: "1", Gross: 1}
	for _, bad := range []Event{{ID: "s", Kind: "sell", OccurredAt: 2, Quantity: "2", Gross: 2}, {ID: "s", Kind: "sell", OccurredAt: 2, Quantity: "1", Gross: 1, Fee: 2}, {ID: "o", Kind: "opening", OccurredAt: 2, Quantity: "1", Gross: 1}, {ID: "b", Kind: "buy", OccurredAt: 2, Quantity: "1", Gross: 1}, {ID: "x", Kind: "buy", OccurredAt: 2, Quantity: "1", Gross: MaxMoney}, {ID: "x", Kind: "split", OccurredAt: 2, Quantity: "1"}} {
		if _, e := Replay([]Event{base, bad}); e == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}
func TestReplayOrdersCopy(t *testing.T) {
	es := []Event{{ID: "s", Kind: "sell", OccurredAt: 2, Quantity: "0.000000000001", Gross: 1}, {ID: "b", Kind: "buy", OccurredAt: 1, Quantity: "0.000000000001", Gross: 1}}
	original := append([]Event(nil), es...)
	r, e := Replay(es)
	if e != nil || r.Quantity != "0" {
		t.Fatalf("%+v %v", r, e)
	}
	if !reflect.DeepEqual(es, original) {
		t.Fatal("mutated input")
	}
}
