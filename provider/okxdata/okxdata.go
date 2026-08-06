// Package okxdata provides exchange-direct market analytics from OKX's
// public API (no auth, no claw402/x402 gateway). It replaces the
// claw402-routed fxosClient calls for quant data, OI ranking, netflow
// ranking and price ranking. Return shapes mirror the nofx types so the
// prompt formatters (nofx.Format*ForAI) and kernel consumers keep working
// unchanged.
package okxdata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fxos/httpclient"
)

// API base endpoints (public, no auth).
const (
	publicURL   = "https://www.okx.com/api/v5/public"
	marketURL   = "https://www.okx.com/api/v5/market"
	rubikURL    = "https://www.okx.com/api/v5/rubik/stat"
	tickersPath = "/tickers?instType=SWAP"
	oiPath      = "/open-interest?instType=SWAP"
)

var client = httpclient.New(30 * time.Second)

// okxGet performs a GET and decodes the OKX envelope. okCode "0" = success.
func okxGet(url string, target any) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("okx request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("okx read failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("okx status %d: %s", resp.StatusCode, string(body))
	}

	var envelope struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data json.RawMessage
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("okx envelope parse: %w", err)
	}
	if envelope.Code != "0" {
		return fmt.Errorf("okx api error: code=%s msg=%s", envelope.Code, envelope.Msg)
	}
	if target != nil {
		if err := json.Unmarshal(envelope.Data, target); err != nil {
			return fmt.Errorf("okx data parse: %w", err)
		}
	}
	return nil
}

// Ticker mirrors the OKX /market/tickers SWAP fields we consume.
type Ticker struct {
	InstID  string `json:"instId"`
	Last    string `json:"last"`
	Open24h string `json:"open24h"`
}

// tickers fetches all SWAP tickers once (one call, reused by every function).
func tickers() ([]Ticker, error) {
	var out []Ticker
	if err := okxGet(marketURL+tickersPath, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// baseSymbol strips the USDT/USDC/quote suffix from an OKX instId to the
// bare coin base (BTC-USDT-SWAP → BTC, BTC-USD-SWAP → BTC).
func baseSymbol(instID string) string {
	s := strings.ToUpper(strings.TrimSpace(instID))
	s = strings.TrimSuffix(s, "-SWAP")
	for _, quote := range []string{"-USDT", "-USDC", "-USD"} {
		if strings.HasSuffix(s, quote) {
			s = strings.TrimSuffix(s, quote)
			break
		}
	}
	return s
}
