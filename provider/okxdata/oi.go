package okxdata

import (
	"sort"
	"strconv"
	"time"

	"fxos/provider/nofx"
)

// OIEntry is one row of /public/open-interest?instType=SWAP.
type OIEntry struct {
	InstID string `json:"instId"`
	OI     string `json:"oi"`
	OICcy  string `json:"oiCcy"`
	OIUsd  string `json:"oiUsd"`
}

// OIRanking builds a nofx.OIRankingData from OKX SWAP open interest
// (exchange-direct). Top = highest current OI (USD), Low = lowest.
// Note: OKX's bulk OI endpoint exposes current levels only — a per-coin
// history would require N sequential rubik calls, so delta columns are
// filled with the current OI value and a 0 change (the ranking itself is
// level-based, which is what the prompt's "capital concentration" signal
// needs).
func OIRanking(duration string, limit int) (*nofx.OIRankingData, error) {
	if duration == "" {
		duration = "24h"
	}
	if limit <= 0 {
		limit = 10
	}

	var entries []OIEntry
	if err := okxGet(publicURL+oiPath, &entries); err != nil {
		return nil, err
	}

	type oiInfo struct {
		symbol string
		oiUsd  float64
	}
	items := make([]oiInfo, 0, len(entries))
	for _, e := range entries {
		if !stringsHasSuffix(e.InstID, "-USDT-SWAP") {
			continue // keep the USDT-quoted universe only, matching klines
		}
		oi, err := strconv.ParseFloat(e.OIUsd, 64)
		if err != nil || oi <= 0 {
			continue
		}
		items = append(items, oiInfo{symbol: baseSymbol(e.InstID), oiUsd: oi})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].oiUsd > items[j].oiUsd })

	top := make([]nofx.OIPosition, 0, limit)
	low := make([]nofx.OIPosition, 0, limit)
	for i := 0; i < len(items) && (len(top) < limit || len(low) < limit); i++ {
		if len(top) < limit {
			top = append(top, nofx.OIPosition{
				Rank:           len(top) + 1,
				Symbol:         items[i].symbol,
				CurrentOI:      items[i].oiUsd,
				OIDeltaValue:   items[i].oiUsd,
				OIDeltaPercent: 0,
			})
		}
		j := len(items) - 1 - i
		if j >= 0 && j != i && len(low) < limit {
			low = append(low, nofx.OIPosition{
				Rank:           len(low) + 1,
				Symbol:         items[j].symbol,
				CurrentOI:      items[j].oiUsd,
				OIDeltaValue:   items[j].oiUsd,
				OIDeltaPercent: 0,
			})
		}
	}

	return &nofx.OIRankingData{
		Duration:     duration,
		TopPositions: top,
		LowPositions: low,
		FetchedAt:    time.Now(),
	}, nil
}

// stringsHasSuffix is a tiny local alias to keep imports tidy.
func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
