package investments

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestQuoteProvenanceAndStaleness(t *testing.T) {
	now := time.Now().UTC()
	payload := func(fetched, outcome, price string) []byte {
		return []byte(fmt.Sprintf(`{"observation":{"source_id":"cmb_spot","series_id":"cmb.gold.customer_quote","unit":"CNY/g","version_id":1,"raw_sha256":"%s","fetched_at":"%s","data":{"customer_sell_cny_per_g":"%s","api_now_time":"raw-bank-value","market_day":false,"quoted_price":true}},"last_attempt":{"outcome":"%s"}}`, strings.Repeat("a", 64), fetched, price, outcome))
	}
	q, err := ParseCMBPlatformQuote(payload(now.Format(time.RFC3339Nano), "success", "999.12"), now)
	if err != nil || q.MarketDay || q.APINowTime != "raw-bank-value" {
		t.Fatalf("quote %+v %v", q, err)
	}
	for _, raw := range [][]byte{payload(now.Add(-time.Hour).Format(time.RFC3339Nano), "success", "999"), payload(now.Format(time.RFC3339Nano), "failure", "999"), payload(now.Format(time.RFC3339Nano), "success", "NaN")} {
		if _, err = ParseCMBPlatformQuote(raw, now); err == nil {
			t.Fatal("invalid quote accepted")
		}
	}
	v, err := Value("12", "999.12")
	if err != nil || v != 1198944 {
		t.Fatalf("value %d %v", v, err)
	}
}

func TestGenericQuoteAndExactGoldValue(t *testing.T) {
	now := time.Now()
	raw := []byte(fmt.Sprintf(`{"source":"operator gold feed","unit":"CNY/g","customerSell":"1045.67","fetchedAt":"%s"}`, now.Format(time.RFC3339Nano)))
	q, err := ParseGenericGoldQuote(raw, now, 5*time.Minute)
	if err != nil || q.CustomerSell != "1045.67" {
		t.Fatal(err)
	}
	if _, err = ParseGenericGoldQuote(raw, now.Add(6*time.Minute), 5*time.Minute); err == nil {
		t.Fatal("stale accepted")
	}
	if _, err = ParseGenericGoldQuote([]byte(strings.ReplaceAll(string(raw), "CNY/g", "USD/oz")), now, 5*time.Minute); err == nil {
		t.Fatal("wrong unit accepted")
	}
	value, err := Value("29.3008", "1045.67")
	if err != nil || value != 3063897 {
		t.Fatal("exact cent rounding", value, err)
	}
}
