// Package investments contains exact, deterministic investment calculations.
// It never changes account balances or executes a financial transaction.
package investments

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strings"
)

const Algorithm = "MA_v1"
const MaxMoney int64 = 9999999999999 // v1.6.1 transaction contract, minor units
const Scale = 12

var decimalPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]{1,12})?$`)
var factor = new(big.Int).Exp(big.NewInt(10), big.NewInt(Scale), nil)

// Quantity parses canonical, nonnegative decimal input. No exponent or rounding.
func Quantity(s string) (*big.Int, error) {
	if len(s) > 39 || !decimalPattern.MatchString(s) {
		return nil, errors.New("quantity must be a decimal with at most 38 digits and 12 decimal places")
	}
	parts := strings.Split(s, ".")
	fractional := ""
	if len(parts) == 2 {
		fractional = parts[1]
	}
	if len(parts[0])+len(fractional) > 38 {
		return nil, errors.New("quantity exceeds 38 digits")
	}
	n, ok := new(big.Int).SetString(parts[0]+fractional+strings.Repeat("0", Scale-len(fractional)), 10)
	if !ok {
		return nil, errors.New("invalid quantity")
	}
	return n, nil
}
func FormatQuantity(n *big.Int) string {
	whole, rem := new(big.Int), new(big.Int)
	whole.QuoRem(n, factor, rem)
	if rem.Sign() == 0 {
		return whole.String()
	}
	frac := rem.String()
	frac = strings.Repeat("0", Scale-len(frac)) + frac
	return whole.String() + "." + strings.TrimRight(frac, "0")
}

// Event is a current revision, not a mutable database row. Amounts are minor units.
type Event struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"` // opening, buy, sell
	OccurredAt int64  `json:"occurredAt"`
	Sequence   int64  `json:"sequence"` // stable tie-break, unchanged by correction
	Quantity   string `json:"quantity"`
	Gross      int64  `json:"gross,string"` // excludes fee; opening: starting cost
	Fee        int64  `json:"fee,string"`
	Cancelled  bool   `json:"cancelled"`
}
type Effect struct {
	EventID       string `json:"eventId"`
	Quantity      string `json:"quantity"`
	Cost          int64  `json:"cost,string"`
	CashDelta     int64  `json:"cashDelta,string"`
	AllocatedCost int64  `json:"allocatedCost,string"`
	Realized      int64  `json:"realized,string"`
}
type Result struct {
	Algorithm string   `json:"algorithm"`
	Quantity  string   `json:"quantity"`
	Cost      int64    `json:"cost,string"`
	Realized  int64    `json:"realized,string"`
	Effects   []Effect `json:"effects"`
}

func validMoney(n int64) bool { return n >= 0 && n <= MaxMoney }
func addBounded(a, b int64) (int64, error) {
	n := new(big.Int).Add(big.NewInt(a), big.NewInt(b))
	if !n.IsInt64() || n.Cmp(big.NewInt(MaxMoney)) > 0 || n.Cmp(big.NewInt(-MaxMoney)) < 0 {
		return 0, errors.New("money exceeds supported range")
	}
	return n.Int64(), nil
}

// Replay uses half-up rounding in minor units; a complete disposal consumes all cost.
// Replaying an invalid historical sequence returns no partial result.
func Replay(input []Event) (Result, error) {
	events := append([]Event(nil), input...)
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].OccurredAt == events[j].OccurredAt {
			return events[i].Sequence < events[j].Sequence
		}
		return events[i].OccurredAt < events[j].OccurredAt
	})
	out := Result{Algorithm: Algorithm, Quantity: "0", Effects: []Effect{}}
	quantity := new(big.Int)
	seen := map[string]bool{}
	order := map[string]bool{}
	active := 0
	for _, e := range events {
		if e.ID == "" || seen[e.ID] {
			return Result{}, errors.New("duplicate or empty event id")
		}
		seen[e.ID] = true
		key := fmt.Sprintf("%d/%d", e.OccurredAt, e.Sequence)
		if e.OccurredAt <= 0 || e.Sequence < 0 || order[key] {
			return Result{}, errors.New("invalid or ambiguous event order")
		}
		order[key] = true
		if e.Cancelled {
			continue
		}
		q, err := Quantity(e.Quantity)
		if err != nil || q.Sign() <= 0 {
			return Result{}, fmt.Errorf("event %s: quantity must be positive", e.ID)
		}
		if !validMoney(e.Gross) || !validMoney(e.Fee) {
			return Result{}, errors.New("amount or fee out of range")
		}
		effect := Effect{EventID: e.ID}
		switch e.Kind {
		case "opening":
			if active != 0 || e.Fee != 0 {
				return Result{}, errors.New("opening must be first and cannot have a fee")
			}
			quantity.Set(q)
			out.Cost = e.Gross
		case "buy":
			total, err := addBounded(e.Gross, e.Fee)
			if err != nil {
				return Result{}, err
			}
			if total <= 0 {
				return Result{}, errors.New("buy cost must be positive")
			}
			cost, err := addBounded(out.Cost, total)
			if err != nil {
				return Result{}, err
			}
			out.Cost = cost
			quantity.Add(quantity, q)
			effect.CashDelta = -total
		case "sell":
			if quantity.Sign() == 0 || q.Cmp(quantity) > 0 {
				return Result{}, fmt.Errorf("event %s: insufficient quantity", e.ID)
			}
			if e.Fee > e.Gross {
				return Result{}, errors.New("sell fee exceeds proceeds")
			}
			allocated := out.Cost
			if q.Cmp(quantity) != 0 {
				num := new(big.Int).Mul(big.NewInt(out.Cost), q)
				quot, rem := new(big.Int), new(big.Int)
				quot.QuoRem(num, quantity, rem)
				if new(big.Int).Lsh(rem, 1).Cmp(quantity) >= 0 {
					quot.Add(quot, big.NewInt(1))
				}
				allocated = quot.Int64()
			}
			quantity.Sub(quantity, q)
			out.Cost -= allocated
			effect.CashDelta = e.Gross - e.Fee
			effect.AllocatedCost = allocated
			effect.Realized = effect.CashDelta - allocated
			realized, err := addBounded(out.Realized, effect.Realized)
			if err != nil {
				return Result{}, err
			}
			out.Realized = realized
		default:
			return Result{}, errors.New("unsupported investment operation")
		}
		// Bound accumulated quantity as well as each input.
		if _, err := Quantity(FormatQuantity(quantity)); err != nil {
			return Result{}, err
		}
		active++
		effect.Quantity = FormatQuantity(quantity)
		effect.Cost = out.Cost
		out.Effects = append(out.Effects, effect)
	}
	out.Quantity = FormatQuantity(quantity)
	return out, nil
}
