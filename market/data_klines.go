package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"fxos/logger"
	"fxos/provider/coinank/coinank_api"
	"fxos/provider/coinank/coinank_enum"
	"fxos/provider/hyperliquid"
	"strconv"
	"strings"
	"time"
)

// Note: Kline data now uses free/open API (coinank_api.Kline) which doesn't require authentication

// okxKlinesURL is the OKX public market candles endpoint (no auth required).
// SWAP instId format: BTC-USDT-SWAP. Bar units: 1m/3m/5m/15m/30m/1H/2H/4H/6H/12H/1D...
const okxKlinesURL = "https://www.okx.com/api/v5/market/candles"

// getKlinesFromOKX fetches kline data from the OKX public market API
// (exchange-direct, no claw402/third-party gateway). Used as the default
// source for regular crypto assets; xyz dex assets keep Hyperliquid.
func getKlinesFromOKX(symbol, interval string, limit int) ([]Kline, error) {
	base := okxBaseSymbol(symbol)
	instID := base + "-USDT-SWAP"

	bar := okxMapInterval(interval)
	if bar == "" {
		return nil, fmt.Errorf("unsupported interval for OKX: %s", interval)
	}
	if limit <= 0 || limit > 300 {
		limit = 200
	}

	apiClient := NewAPIClient()
	url := fmt.Sprintf("%s?instId=%s&bar=%s&limit=%d", okxKlinesURL, instID, bar, limit)
	resp, err := apiClient.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code string     `json:"code"`
		Data [][]string `json:"data"`
		Msg  string     `json:"msg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != "0" || len(result.Data) == 0 {
		return nil, fmt.Errorf("okx klines failed: code=%s msg=%s", result.Code, result.Msg)
	}

	// OKX candle format: [ts, o, h, l, c, vol, volCcy, volCcyQuote, confirm]
	klines := make([]Kline, 0, len(result.Data))
	for _, row := range result.Data {
		if len(row) < 6 {
			continue
		}
		openTime, _ := strconv.ParseInt(row[0], 10, 64)
		open, _ := strconv.ParseFloat(row[1], 64)
		high, _ := strconv.ParseFloat(row[2], 64)
		low, _ := strconv.ParseFloat(row[3], 64)
		closePrice, _ := strconv.ParseFloat(row[4], 64)
		volume, _ := strconv.ParseFloat(row[5], 64)
		klines = append(klines, Kline{
			OpenTime:  openTime,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			Volume:    volume,
			CloseTime: openTime + okxBarMillis(bar) - 1,
		})
	}

	// OKX returns newest-first; the rest of the pipeline expects oldest-first.
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}
	return klines, nil
}

// GetOKXKlines fetches kline data from OKX (exchange-direct, no auth).
// Exported for the API layer (handler_klines); the kernel uses
// getKlinesFromOKX directly.
func GetOKXKlines(symbol, interval string, limit int) ([]Kline, error) {
	return getKlinesFromOKX(symbol, interval, limit)
}

// okxMapInterval maps the market package's interval names to OKX bar units.
// Matches on the exact interval first so "1M" (monthly, unsupported) is not
// swallowed by the lowercase "1m" (minute) case.
func okxMapInterval(interval string) string {
	switch interval {
	case "1m", "3m", "5m", "15m", "30m":
		return interval
	case "1h":
		return "1H"
	case "2h":
		return "2H"
	case "4h":
		return "4H"
	case "6h":
		return "6H"
	case "12h":
		return "12H"
	case "1d":
		return "1D"
	default:
		return ""
	}
}

// okxBarMillis returns the duration of an OKX bar in milliseconds.
func okxBarMillis(bar string) int64 {
	switch bar {
	case "1m":
		return 60_000
	case "3m":
		return 180_000
	case "5m":
		return 300_000
	case "15m":
		return 900_000
	case "30m":
		return 1_800_000
	case "1H":
		return 3_600_000
	case "2H":
		return 7_200_000
	case "4H":
		return 14_400_000
	case "6H":
		return 21_600_000
	case "12H":
		return 43_200_000
	case "1D":
		return 86_400_000
	default:
		return 60_000
	}
}

// getKlinesFromCoinAnk fetches kline data from CoinAnk API (replacement for WSMonitorCli)
func getKlinesFromCoinAnk(symbol, interval, exchange string, limit int) ([]Kline, error) {
	// Map interval string to coinank enum
	var coinankInterval coinank_enum.Interval
	switch interval {
	case "1m":
		coinankInterval = coinank_enum.Minute1
	case "3m":
		coinankInterval = coinank_enum.Minute3
	case "5m":
		coinankInterval = coinank_enum.Minute5
	case "15m":
		coinankInterval = coinank_enum.Minute15
	case "30m":
		coinankInterval = coinank_enum.Minute30
	case "1h":
		coinankInterval = coinank_enum.Hour1
	case "2h":
		coinankInterval = coinank_enum.Hour2
	case "4h":
		coinankInterval = coinank_enum.Hour4
	case "6h":
		coinankInterval = coinank_enum.Hour6
	case "8h":
		coinankInterval = coinank_enum.Hour8
	case "12h":
		coinankInterval = coinank_enum.Hour12
	case "1d":
		coinankInterval = coinank_enum.Day1
	case "3d":
		coinankInterval = coinank_enum.Day3
	case "1w":
		coinankInterval = coinank_enum.Week1
	default:
		return nil, fmt.Errorf("unsupported interval: %s", interval)
	}

	// Map exchange string to coinank enum
	var coinankExchange coinank_enum.Exchange
	switch strings.ToLower(exchange) {
	case "binance":
		coinankExchange = coinank_enum.Binance
	case "bybit":
		coinankExchange = coinank_enum.Bybit
	case "okx":
		coinankExchange = coinank_enum.Okex
	case "bitget":
		coinankExchange = coinank_enum.Bitget
	case "gate":
		coinankExchange = coinank_enum.Gate
	case "hyperliquid":
		coinankExchange = coinank_enum.Hyperliquid
	case "aster":
		coinankExchange = coinank_enum.Aster
	default:
		// Default to Binance for unknown exchanges
		coinankExchange = coinank_enum.Binance
	}

	// Call CoinAnk free/open API (no authentication required)
	ctx := context.Background()
	ts := time.Now().UnixMilli()
	// Use "To" side to search backward from current time (get historical klines)
	coinankKlines, err := coinank_api.Kline(ctx, symbol, coinankExchange, ts, coinank_enum.To, limit, coinankInterval)
	if err != nil || len(coinankKlines) == 0 {
		// If exchange-specific data fails or returns empty, fallback to Binance
		if coinankExchange != coinank_enum.Binance {
			if err != nil {
				logger.Warnf("⚠️ CoinAnk %s data failed, falling back to Binance: %v", exchange, err)
			} else {
				logger.Warnf("⚠️ CoinAnk %s %s data empty for %s, falling back to Binance", exchange, interval, symbol)
			}
			coinankKlines, err = coinank_api.Kline(ctx, symbol, coinank_enum.Binance, ts, coinank_enum.To, limit, coinankInterval)
			if err != nil {
				return nil, fmt.Errorf("CoinAnk API error (fallback): %w", err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("CoinAnk API error: %w", err)
		}
	}

	// Convert coinank kline format to market.Kline format
	klines := make([]Kline, len(coinankKlines))
	for i, ck := range coinankKlines {
		klines[i] = Kline{
			OpenTime:  ck.StartTime,
			Open:      ck.Open,
			High:      ck.High,
			Low:       ck.Low,
			Close:     ck.Close,
			Volume:    ck.Volume,
			CloseTime: ck.EndTime,
		}
	}

	return klines, nil
}

// getKlinesFromHyperliquid fetches kline data from Hyperliquid API for xyz dex assets
func getKlinesFromHyperliquid(symbol, interval string, limit int) ([]Kline, error) {
	// Pass the symbol AS-IS to GetCandles. It internally calls FormatCoinForAPI
	// which handles the xyz: prefix correctly. Stripping the prefix here was a
	// bug: if the base symbol (e.g. "QNT") was not in our hardcoded
	// StockPerpsSymbols list, FormatCoinForAPI couldn't tell it was an xyz
	// asset and the request hit the crypto perp endpoint instead — which
	// returns 500 for stock symbols that have no crypto perp on Hyperliquid.
	hlInterval := hyperliquid.MapTimeframe(interval)

	client := hyperliquid.NewClient()
	ctx := context.Background()
	candles, err := client.GetCandles(ctx, symbol, hlInterval, limit)
	if err != nil {
		return nil, fmt.Errorf("Hyperliquid API error: %w", err)
	}

	// Convert to market.Kline format
	klines := make([]Kline, len(candles))
	for i, c := range candles {
		open, _ := strconv.ParseFloat(c.Open, 64)
		high, _ := strconv.ParseFloat(c.High, 64)
		low, _ := strconv.ParseFloat(c.Low, 64)
		closePrice, _ := strconv.ParseFloat(c.Close, 64)
		volume, _ := strconv.ParseFloat(c.Volume, 64)

		klines[i] = Kline{
			OpenTime:  c.OpenTime,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			Volume:    volume,
			CloseTime: c.CloseTime,
		}
	}

	return klines, nil
}

// calculateTimeframeSeries calculates series data for a single timeframe
func calculateTimeframeSeries(klines []Kline, timeframe string, count int) *TimeframeSeriesData {
	if count <= 0 {
		count = 10 // default
	}

	data := &TimeframeSeriesData{
		Timeframe:   timeframe,
		Klines:      make([]KlineBar, 0, count),
		MidPrices:   make([]float64, 0, count),
		EMA20Values: make([]float64, 0, count),
		EMA50Values: make([]float64, 0, count),
		MACDValues:  make([]float64, 0, count),
		RSI7Values:  make([]float64, 0, count),
		RSI14Values: make([]float64, 0, count),
		Volume:      make([]float64, 0, count),
		BOLLUpper:   make([]float64, 0, count),
		BOLLMiddle:  make([]float64, 0, count),
		BOLLLower:   make([]float64, 0, count),
	}

	// Get latest N data points based on count from config
	start := len(klines) - count
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		// Store full OHLCV kline data
		data.Klines = append(data.Klines, KlineBar{
			Time:   klines[i].OpenTime,
			Open:   klines[i].Open,
			High:   klines[i].High,
			Low:    klines[i].Low,
			Close:  klines[i].Close,
			Volume: klines[i].Volume,
		})

		// Keep MidPrices and Volume for backward compatibility
		data.MidPrices = append(data.MidPrices, klines[i].Close)
		data.Volume = append(data.Volume, klines[i].Volume)

		// Calculate EMA20 for each point
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			data.EMA20Values = append(data.EMA20Values, ema20)
		}

		// Calculate EMA50 for each point
		if i >= 49 {
			ema50 := calculateEMA(klines[:i+1], 50)
			data.EMA50Values = append(data.EMA50Values, ema50)
		}

		// Calculate MACD for each point
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}

		// Calculate RSI for each point
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}

		// Calculate Bollinger Bands (period 20, std dev multiplier 2)
		if i >= 19 {
			upper, middle, lower := calculateBOLL(klines[:i+1], 20, 2.0)
			data.BOLLUpper = append(data.BOLLUpper, upper)
			data.BOLLMiddle = append(data.BOLLMiddle, middle)
			data.BOLLLower = append(data.BOLLLower, lower)
		}
	}

	// Calculate ATR14
	data.ATR14 = calculateATR(klines, 14)

	return data
}

// calculatePriceChangeByBars calculates how many K-lines to look back for price change based on timeframe
func calculatePriceChangeByBars(klines []Kline, timeframe string, targetMinutes int) float64 {
	if len(klines) < 2 {
		return 0
	}

	// Parse timeframe to minutes
	tfMinutes := parseTimeframeToMinutes(timeframe)
	if tfMinutes <= 0 {
		return 0
	}

	// Calculate how many K-lines to look back
	barsBack := targetMinutes / tfMinutes
	if barsBack < 1 {
		barsBack = 1
	}

	currentPrice := klines[len(klines)-1].Close
	idx := len(klines) - 1 - barsBack
	if idx < 0 {
		idx = 0
	}

	oldPrice := klines[idx].Close
	if oldPrice > 0 {
		return ((currentPrice - oldPrice) / oldPrice) * 100
	}
	return 0
}

// parseTimeframeToMinutes parses timeframe string to minutes
func parseTimeframeToMinutes(tf string) int {
	switch tf {
	case "1m":
		return 1
	case "3m":
		return 3
	case "5m":
		return 5
	case "15m":
		return 15
	case "30m":
		return 30
	case "1h":
		return 60
	case "2h":
		return 120
	case "4h":
		return 240
	case "6h":
		return 360
	case "8h":
		return 480
	case "12h":
		return 720
	case "1d":
		return 1440
	case "3d":
		return 4320
	case "1w":
		return 10080
	default:
		return 0
	}
}

// calculateIntradaySeries calculates intraday series data
func calculateIntradaySeries(klines []Kline) *IntradayData {
	data := &IntradayData{
		MidPrices:   make([]float64, 0, 10),
		EMA20Values: make([]float64, 0, 10),
		MACDValues:  make([]float64, 0, 10),
		RSI7Values:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
		Volume:      make([]float64, 0, 10),
	}

	// Get latest 10 data points
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)
		data.Volume = append(data.Volume, klines[i].Volume)

		// Calculate EMA20 for each point
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			data.EMA20Values = append(data.EMA20Values, ema20)
		}

		// Calculate MACD for each point
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}

		// Calculate RSI for each point
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	// Calculate 3m ATR14
	data.ATR14 = calculateATR(klines, 14)

	return data
}

// calculateLongerTermData calculates longer-term data
func calculateLongerTermData(klines []Kline) *LongerTermData {
	data := &LongerTermData{
		MACDValues:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// Calculate EMA
	data.EMA20 = calculateEMA(klines, 20)
	data.EMA50 = calculateEMA(klines, 50)

	// Calculate ATR
	data.ATR3 = calculateATR(klines, 3)
	data.ATR14 = calculateATR(klines, 14)

	// Calculate volume
	if len(klines) > 0 {
		data.CurrentVolume = klines[len(klines)-1].Volume
		// Calculate average volume
		sum := 0.0
		for _, k := range klines {
			sum += k.Volume
		}
		data.AverageVolume = sum / float64(len(klines))
	}

	// Calculate MACD and RSI series
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	return data
}

// GetBoxData fetches 1h klines and calculates box data for a symbol
func GetBoxData(symbol string) (*BoxData, error) {
	symbol = Normalize(symbol)

	// Fetch 500 1h klines
	var klines []Kline
	var err error

	if IsXyzDexAsset(symbol) {
		klines, err = getKlinesFromHyperliquid(symbol, "1h", LongBoxPeriod)
	} else {
		klines, err = getKlinesFromCoinAnk(symbol, "1h", "binance", LongBoxPeriod)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get 1h klines: %w", err)
	}

	if len(klines) == 0 {
		return nil, fmt.Errorf("no kline data available")
	}

	currentPrice := klines[len(klines)-1].Close

	return calculateBoxData(klines, currentPrice), nil
}
