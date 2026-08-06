package okxdata

import (
	"strconv"
)

// CoinQuant is the exchange-direct quant snapshot for a single coin, built
// from one bulk tickers call + one bulk OI call (no per-coin requests).
type CoinQuant struct {
	Symbol      string  // bare base, e.g. BTC
	Price       float64 // last price
	Open24h     float64 // 24h open (for price-change computation)
	OI          float64 // open interest in USD
	OIDelta     float64 // 24h OI change in USD (0 when history unavailable)
	OIDeltaPct  float64 // 24h OI change percent (0 when history unavailable)
	PriceChange float64 // 24h price change (decimal: 0.07 = 7%)
}

// QuantSnapshots fetches quant data for a set of symbols in TWO bulk calls
// (tickers + open-interest), returning a map keyed by bare base symbol.
func QuantSnapshots() (map[string]CoinQuant, error) {
	ts, err := tickers()
	if err != nil {
		return nil, err
	}
	var ois []OIEntry
	if err := okxGet(publicURL+oiPath, &ois); err != nil {
		return nil, err
	}

	oiByBase := make(map[string]float64, len(ois))
	for _, e := range ois {
		if !stringsHasSuffix(e.InstID, "-USDT-SWAP") {
			continue
		}
		oi, err := strconv.ParseFloat(e.OIUsd, 64)
		if err != nil || oi <= 0 {
			continue
		}
		oiByBase[baseSymbol(e.InstID)] = oi
	}

	out := make(map[string]CoinQuant, len(ts))
	for _, t := range ts {
		last, err1 := strconv.ParseFloat(t.Last, 64)
		open, err2 := strconv.ParseFloat(t.Open24h, 64)
		if err1 != nil || err2 != nil || last <= 0 {
			continue
		}
		base := baseSymbol(t.InstID)
		q := CoinQuant{
			Symbol:  base,
			Price:   last,
			Open24h: open,
		}
		if open > 0 {
			q.PriceChange = (last - open) / open
		}
		q.OI = oiByBase[base]
		out[base] = q
	}
	return out, nil
}
