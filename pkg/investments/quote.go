package investments

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"time"
)

// QuoteProvider is read-only. Prices never modify position cost or cash journals.
type QuoteProvider interface {
	Fetch(context.Context) (*GoldQuote, error)
}
type GoldQuote struct {
	Source       string `json:"source"`
	Unit         string `json:"unit"`
	CustomerSell string `json:"customerSell"`
	FetchedAt    string `json:"fetchedAt"`
	APINowTime   string `json:"apiNowTime"`
	MarketDay    bool   `json:"marketDay"`
	QuotedPrice  bool   `json:"quotedPrice"`
	RawSHA256    string `json:"rawSha256"`
	VersionID    int64  `json:"versionId"`
}
type CMBPlatformProvider struct {
	Endpoint string
	Token    string
	Provider string
	MaxAge   time.Duration
}

func (p CMBPlatformProvider) Fetch(ctx context.Context) (*GoldQuote, error) {
	u, err := url.Parse(p.Endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" {
		return nil, errors.New("quote provider is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.Endpoint, nil)
	if err != nil {
		return nil, err
	}
	if p.Token != "" {
		req.Header.Set("Authorization", "Bearer "+p.Token)
	}
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		return nil, errors.New("quote provider request failed")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New("quote provider returned an error")
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 2*1024*1024+1))
	if err != nil || len(raw) > 2*1024*1024 {
		return nil, errors.New("quote response exceeds limit")
	}
	age := p.MaxAge
	if age <= 0 {
		age = 30 * time.Minute
	}
	if p.Provider == "gold_json" {
		return ParseGenericGoldQuote(raw, time.Now(), age)
	}
	if p.Provider != "" && p.Provider != "cmb_platform" {
		return nil, errors.New("unsupported quote provider")
	}
	return parseCMBPlatformQuote(raw, time.Now(), age)
}
func ParseCMBPlatformQuote(raw []byte, now time.Time) (*GoldQuote, error) {
	return parseCMBPlatformQuote(raw, now, 30*time.Minute)
}
func parseCMBPlatformQuote(raw []byte, now time.Time, maxAge time.Duration) (*GoldQuote, error) {
	var response struct {
		Observation struct {
			Source  string `json:"source_id"`
			Series  string `json:"series_id"`
			Unit    string `json:"unit"`
			Fetched string `json:"fetched_at"`
			SHA     string `json:"raw_sha256"`
			Version int64  `json:"version_id"`
			Data    struct {
				Sell   string `json:"customer_sell_cny_per_g"`
				Now    string `json:"api_now_time"`
				Market *bool  `json:"market_day"`
				Quoted *bool  `json:"quoted_price"`
			} `json:"data"`
		} `json:"observation"`
		LastAttempt struct {
			Outcome string `json:"outcome"`
		} `json:"last_attempt"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, errors.New("invalid quote response")
	}
	o := response.Observation
	if o.Source != "cmb_spot" || o.Series != "cmb.gold.customer_quote" || o.Unit != "CNY/g" || o.Version <= 0 || len(o.SHA) != 64 || o.Data.Market == nil || o.Data.Quoted == nil || o.Data.Now == "" {
		return nil, errors.New("quote provenance is incomplete")
	}
	if response.LastAttempt.Outcome != "success" {
		return nil, errors.New("most recent collection did not succeed")
	}
	fetched, err := time.Parse(time.RFC3339Nano, o.Fetched)
	if err != nil || now.Sub(fetched) > maxAge || fetched.After(now.Add(time.Minute)) {
		return nil, errors.New("quote sample is stale or has an invalid time")
	}
	if _, err := hex.DecodeString(o.SHA); err != nil {
		return nil, errors.New("invalid quote provenance hash")
	}
	price, err := Quantity(o.Data.Sell)
	if err != nil || price.Sign() <= 0 {
		return nil, errors.New("invalid quote price")
	}
	return &GoldQuote{Source: o.Source, Unit: o.Unit, CustomerSell: o.Data.Sell, FetchedAt: o.Fetched, APINowTime: o.Data.Now, MarketDay: *o.Data.Market, QuotedPrice: *o.Data.Quoted, RawSHA256: o.SHA, VersionID: o.Version}, nil
}

// Value rounds a displayed estimate to cents. It is never an executable price.
func Value(quantity, unitPrice string) (int64, error) {
	q, err := Quantity(quantity)
	if err != nil {
		return 0, err
	}
	p, err := Quantity(unitPrice)
	if err != nil || p.Sign() <= 0 {
		return 0, errors.New("invalid price")
	}
	numerator := new(big.Int).Mul(q, p)
	numerator.Mul(numerator, big.NewInt(100))
	denominator := new(big.Int).Mul(factor, factor)
	numerator.Add(numerator, new(big.Int).Quo(denominator, big.NewInt(2)))
	value := numerator.Quo(numerator, denominator)
	if !value.IsInt64() || value.Cmp(big.NewInt(MaxMoney)) > 0 {
		return 0, errors.New("valuation exceeds range")
	}
	return value.Int64(), nil
}

// Generic adapter contract for deployments without a CMB data platform.
// The operator supplies a read-only endpoint; browser users cannot select network targets.
func ParseGenericGoldQuote(raw []byte, now time.Time, maxAge time.Duration) (*GoldQuote, error) {
	var q GoldQuote
	if err := json.Unmarshal(raw, &q); err != nil {
		return nil, errors.New("invalid quote response")
	}
	if q.Source == "" || len(q.Source) > 128 || q.Unit != "CNY/g" {
		return nil, errors.New("invalid quote source or unit")
	}
	at, err := time.Parse(time.RFC3339Nano, q.FetchedAt)
	if err != nil || now.Sub(at) > maxAge || at.After(now.Add(time.Minute)) {
		return nil, errors.New("quote sample is stale or has an invalid time")
	}
	if _, err = Value("1", q.CustomerSell); err != nil {
		return nil, err
	}
	return &q, nil
}
