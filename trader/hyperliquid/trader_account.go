package hyperliquid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"fxos/httpclient"
	"fxos/logger"
	"fxos/trader/types"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GetBalance gets account balance
func (t *HyperliquidTrader) GetBalance() (*types.Account, error) {
	logger.Infof("🔄 Calling Hyperliquid API to get account balance...")

	// Step 1: Query Spot account balance
	spotState, err := t.exchange.Info().SpotUserState(t.ctx, t.walletAddr)
	var spotUSDCBalance float64 = 0.0
	if err != nil {
		logger.Infof("⚠️ Failed to query Spot balance (may have no spot assets): %v", err)
	} else if spotState != nil && len(spotState.Balances) > 0 {
		for _, balance := range spotState.Balances {
			if balance.Coin == "USDC" {
				spotUSDCBalance, _ = strconv.ParseFloat(balance.Total, 64)
				logger.Infof("✓ Found Spot balance: %.2f USDC", spotUSDCBalance)
				break
			}
		}
	}

	// Step 2: Query Perpetuals contract account status
	accountState, err := t.exchange.Info().UserState(t.ctx, t.walletAddr)
	if err != nil {
		logger.Infof("❌ Hyperliquid Perpetuals API call failed: %v", err)
		return nil, fmt.Errorf("failed to get account information: %w", err)
	}

	// Parse balance information (MarginSummary fields are all strings)
	result := &types.Account{}

	// Step 3: Dynamically select correct summary based on margin mode (CrossMarginSummary or MarginSummary)
	var accountValue, totalMarginUsed float64
	var summaryType string
	var summary interface{}

	var parseErr error
	if t.isCrossMargin {
		// Cross margin mode: use CrossMarginSummary
		if accountValue, parseErr = types.ParseFloatField("accountValue", accountState.CrossMarginSummary.AccountValue); parseErr != nil {
			return nil, parseErr
		}
		if totalMarginUsed, parseErr = types.ParseFloatField("totalMarginUsed", accountState.CrossMarginSummary.TotalMarginUsed); parseErr != nil {
			return nil, parseErr
		}
		summaryType = "CrossMarginSummary (cross margin)"
		summary = accountState.CrossMarginSummary
	} else {
		// Isolated margin mode: use MarginSummary
		if accountValue, parseErr = types.ParseFloatField("accountValue", accountState.MarginSummary.AccountValue); parseErr != nil {
			return nil, parseErr
		}
		if totalMarginUsed, parseErr = types.ParseFloatField("totalMarginUsed", accountState.MarginSummary.TotalMarginUsed); parseErr != nil {
			return nil, parseErr
		}
		summaryType = "MarginSummary (isolated margin)"
		summary = accountState.MarginSummary
	}

	// Debug: Print complete summary structure returned by API
	summaryJSON, _ := json.MarshalIndent(summary, "  ", "  ")
	logger.Infof("🔍 [DEBUG] Hyperliquid API %s complete data:", summaryType)
	logger.Infof("%s", string(summaryJSON))

	// Critical fix: Accumulate actual unrealized PnL from all positions
	totalUnrealizedPnl := 0.0
	for _, assetPos := range accountState.AssetPositions {
		unrealizedPnl, _ := strconv.ParseFloat(assetPos.Position.UnrealizedPnl, 64)
		totalUnrealizedPnl += unrealizedPnl
	}

	// Correctly understand Hyperliquid fields:
	// AccountValue = Total account equity (includes idle funds + position value + unrealized PnL)
	// TotalMarginUsed = Margin used by positions (included in AccountValue, for display only)
	//
	// To be compatible with auto_types.go calculation logic (totalEquity = totalWalletBalance + totalUnrealizedProfit)
	// Need to return "wallet balance without unrealized PnL"
	walletBalanceWithoutUnrealized := accountValue - totalUnrealizedPnl

	// Step 4: Use Withdrawable field (PR #443)
	// Withdrawable is the official real withdrawable balance, more reliable than simple calculation
	availableBalance := 0.0
	if accountState.Withdrawable != "" {
		withdrawable, err := strconv.ParseFloat(accountState.Withdrawable, 64)
		if err == nil && withdrawable > 0 {
			availableBalance = withdrawable
			logger.Infof("✓ Using Withdrawable as available balance: %.2f", availableBalance)
		}
	}

	// Fallback: If no Withdrawable, use simple calculation
	if availableBalance == 0 && accountState.Withdrawable == "" {
		availableBalance = accountValue - totalMarginUsed
		if availableBalance < 0 {
			logger.Infof("⚠️ Calculated available balance is negative (%.2f), reset to 0", availableBalance)
			availableBalance = 0
		}
	}

	// Step 5: Query xyz dex balance (stock perps, forex, commodities)
	var xyzAccountValue, xyzUnrealizedPnl float64
	var xyzPositions []xyzAssetPosition
	xyzAccountValue, xyzUnrealizedPnl, xyzPositions, err = t.getXYZDexBalance()
	if err != nil {
		// xyz dex query failed - log warning but don't fail the entire balance query
		logger.Infof("⚠️ Failed to query xyz dex balance: %v", err)
	}
	// Always log xyz dex state for debugging
	logger.Infof("🔍 xyz dex state: accountValue=%.4f, unrealizedPnl=%.4f, positions=%d",
		xyzAccountValue, xyzUnrealizedPnl, len(xyzPositions))
	for _, pos := range xyzPositions {
		entryPx := "nil"
		if pos.Position.EntryPx != nil {
			entryPx = *pos.Position.EntryPx
		}
		logger.Infof("   └─ %s: size=%s, entryPx=%s, posValue=%s, pnl=%s",
			pos.Position.Coin, pos.Position.Szi, entryPx, pos.Position.PositionValue, pos.Position.UnrealizedPnl)
	}
	xyzMarginUsed := calculateXYZMarginUsed(xyzPositions)
	balanceBreakdown := calculateHyperliquidBalanceBreakdown(
		t.isUnifiedAccount,
		spotUSDCBalance,
		accountValue,
		totalUnrealizedPnl,
		totalMarginUsed,
		availableBalance,
		xyzAccountValue,
		xyzUnrealizedPnl,
		xyzMarginUsed,
	)
	totalWalletBalance := balanceBreakdown.TotalWalletBalance
	totalUnrealizedPnlAll := balanceBreakdown.TotalUnrealizedProfit
	totalEquityCalculated := balanceBreakdown.TotalEquity
	availableBalance = balanceBreakdown.AvailableBalance

	// Step 7: Unified Account mode - Spot USDC is used as collateral for Perps
	// In this mode, xyz/core account values are collateral views backed by the
	// same Spot USDC. They must not be added on top of Spot or the dashboard
	// will double count equity after a position opens.
	if t.isUnifiedAccount && spotUSDCBalance > 0 {
		logger.Infof("✓ Unified Account: Spot %.2f USDC used as shared collateral (available: %.2f)",
			spotUSDCBalance, availableBalance)
	}

	// Suppress unused variable warning
	_ = totalUnrealizedPnlAll

	result.TotalWalletBalance = totalWalletBalance       // Total assets (Perp + Spot + xyz) - unrealized
	result.TotalEquity = totalEquityCalculated           // Total equity = Perp AV + Spot + xyz AV
	result.AvailableBalance = availableBalance           // Available balance (Perp + Spot if unified)
	result.TotalUnrealizedProfit = totalUnrealizedPnlAll // Unrealized PnL (Perpetuals + xyz)
	result.SpotBalance = spotUSDCBalance                 // Spot balance
	result.XYZDexBalance = xyzAccountValue               // xyz dex equity (stock perps, forex, commodities)
	result.TotalMarginUsed = balanceBreakdown.TotalMarginUsed

	logger.Infof("✓ Hyperliquid complete account:")
	logger.Infof("  • Spot balance: %.2f USDC", spotUSDCBalance)
	logger.Infof("  • Perpetuals equity: %.2f USDC (wallet %.2f + unrealized %.2f)",
		accountValue,
		walletBalanceWithoutUnrealized,
		totalUnrealizedPnl)
	logger.Infof("  • Perpetuals available balance: %.2f USDC", availableBalance)
	logger.Infof("  • Margin used: %.2f USDC", totalMarginUsed)
	logger.Infof("  • xyz dex equity: %.2f USDC (wallet %.2f + unrealized %.2f)",
		xyzAccountValue,
		balanceBreakdown.XYZWalletBalance,
		xyzUnrealizedPnl)
	logger.Infof("  • Total wallet balance: %.2f USDC", totalWalletBalance)
	logger.Infof("  ⭐ Total equity: %.2f USDC | Available: %.2f | Spot: %.2f | xyz view: %.2f",
		totalEquityCalculated, availableBalance, spotUSDCBalance, xyzAccountValue)

	return result, nil
}

type hyperliquidBalanceBreakdown struct {
	TotalWalletBalance    float64
	TotalEquity           float64
	AvailableBalance      float64
	TotalUnrealizedProfit float64
	TotalMarginUsed       float64
	PerpWalletBalance     float64
	XYZWalletBalance      float64
}

func calculateHyperliquidBalanceBreakdown(
	isUnifiedAccount bool,
	spotUSDCBalance float64,
	perpAccountValue float64,
	perpUnrealizedPnl float64,
	perpMarginUsed float64,
	perpWithdrawable float64,
	xyzAccountValue float64,
	xyzUnrealizedPnl float64,
	xyzMarginUsed float64,
) hyperliquidBalanceBreakdown {
	perpWalletBalance := perpAccountValue - perpUnrealizedPnl
	xyzWalletBalance := xyzAccountValue - xyzUnrealizedPnl
	totalUnrealizedPnl := perpUnrealizedPnl + xyzUnrealizedPnl
	totalMarginUsed := perpMarginUsed + xyzMarginUsed

	if isUnifiedAccount && spotUSDCBalance > 0 {
		totalEquity := spotUSDCBalance + totalUnrealizedPnl
		availableBalance := totalEquity - totalMarginUsed
		if availableBalance < 0 {
			availableBalance = 0
		}
		return hyperliquidBalanceBreakdown{
			TotalWalletBalance:    spotUSDCBalance,
			TotalEquity:           totalEquity,
			AvailableBalance:      availableBalance,
			TotalUnrealizedProfit: totalUnrealizedPnl,
			TotalMarginUsed:       totalMarginUsed,
			PerpWalletBalance:     perpWalletBalance,
			XYZWalletBalance:      xyzWalletBalance,
		}
	}

	availableBalance := perpWithdrawable
	if availableBalance == 0 {
		availableBalance = perpAccountValue - perpMarginUsed
	}
	if availableBalance < 0 {
		availableBalance = 0
	}

	return hyperliquidBalanceBreakdown{
		TotalWalletBalance:    perpWalletBalance + spotUSDCBalance + xyzWalletBalance,
		TotalEquity:           perpAccountValue + spotUSDCBalance + xyzAccountValue,
		AvailableBalance:      availableBalance,
		TotalUnrealizedProfit: totalUnrealizedPnl,
		TotalMarginUsed:       totalMarginUsed,
		PerpWalletBalance:     perpWalletBalance,
		XYZWalletBalance:      xyzWalletBalance,
	}
}

func calculateXYZMarginUsed(positions []xyzAssetPosition) float64 {
	total := 0.0
	for _, pos := range positions {
		positionValue, _ := strconv.ParseFloat(pos.Position.PositionValue, 64)
		if positionValue < 0 {
			positionValue = -positionValue
		}
		leverage := float64(pos.Position.Leverage.Value)
		if leverage <= 0 {
			leverage = 1
		}
		total += positionValue / leverage
	}
	return total
}

// xyzDexState represents the clearinghouse state for xyz dex
type xyzDexState struct {
	MarginSummary      *xyzMarginSummary  `json:"marginSummary,omitempty"`
	CrossMarginSummary *xyzMarginSummary  `json:"crossMarginSummary,omitempty"`
	Withdrawable       string             `json:"withdrawable,omitempty"`
	AssetPositions     []xyzAssetPosition `json:"assetPositions,omitempty"`
}

type xyzMarginSummary struct {
	AccountValue    string `json:"accountValue"`
	TotalMarginUsed string `json:"totalMarginUsed"`
}

type xyzAssetPosition struct {
	Position struct {
		Coin          string  `json:"coin"`
		Szi           string  `json:"szi"`
		EntryPx       *string `json:"entryPx"`
		PositionValue string  `json:"positionValue"`
		UnrealizedPnl string  `json:"unrealizedPnl"`
		LiquidationPx *string `json:"liquidationPx"`
		Leverage      struct {
			Type  string `json:"type"`
			Value int    `json:"value"`
		} `json:"leverage"`
	} `json:"position"`
}

// getXYZDexBalance queries the xyz dex balance (stock perps, forex, commodities)
func (t *HyperliquidTrader) getXYZDexBalance() (accountValue float64, unrealizedPnl float64, positions []xyzAssetPosition, err error) {
	// Build request for xyz dex clearinghouse state
	reqBody := map[string]interface{}{
		"type": "clearinghouseState",
		"user": t.walletAddr,
		"dex":  "xyz",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Determine API URL
	apiURL := "https://api.hyperliquid.xyz/info"
	// Note: xyz dex may not be available on testnet

	req, err := http.NewRequestWithContext(t.ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := httpclient.New(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, 0, nil, fmt.Errorf("xyz dex API error (status %d): %s", resp.StatusCode, string(body))
	}

	var state xyzDexState
	if err := json.Unmarshal(body, &state); err != nil {
		return 0, 0, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Parse account value - xyz dex uses MarginSummary for isolated margin mode
	// CrossMarginSummary may exist but with 0 values, so check MarginSummary first
	if state.MarginSummary != nil && state.MarginSummary.AccountValue != "" {
		av, _ := strconv.ParseFloat(state.MarginSummary.AccountValue, 64)
		if av > 0 {
			accountValue = av
		}
	}
	// Fallback to CrossMarginSummary if MarginSummary is 0
	if accountValue == 0 && state.CrossMarginSummary != nil && state.CrossMarginSummary.AccountValue != "" {
		accountValue, _ = strconv.ParseFloat(state.CrossMarginSummary.AccountValue, 64)
	}

	// Calculate total unrealized PnL from positions
	for _, pos := range state.AssetPositions {
		pnl, _ := strconv.ParseFloat(pos.Position.UnrealizedPnl, 64)
		unrealizedPnl += pnl
	}

	return accountValue, unrealizedPnl, state.AssetPositions, nil
}

// GetMarketPrice gets market price (supports both crypto and xyz dex assets)
func (t *HyperliquidTrader) GetMarketPrice(symbol string) (float64, error) {
	coin := convertSymbolToHyperliquid(symbol)

	// Check if this is an xyz dex asset
	if strings.HasPrefix(coin, "xyz:") {
		return t.getXyzMarketPrice(coin)
	}

	// Get all market prices for crypto
	allMids, err := t.exchange.Info().AllMids(t.ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get price: %w", err)
	}

	// Find price for corresponding coin (allMids is map[string]string)
	if priceStr, ok := allMids[coin]; ok {
		priceFloat, err := strconv.ParseFloat(priceStr, 64)
		if err == nil {
			return priceFloat, nil
		}
		return 0, fmt.Errorf("price format error: %v", err)
	}

	return 0, fmt.Errorf("price not found for %s", symbol)
}

// getXyzMarketPrice gets market price for xyz dex assets
func (t *HyperliquidTrader) getXyzMarketPrice(coin string) (float64, error) {
	// Build request for xyz dex allMids
	reqBody := map[string]string{
		"type": "allMids",
		"dex":  "xyz",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := "https://api.hyperliquid.xyz/info"

	req, err := http.NewRequestWithContext(t.ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := httpclient.New(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("xyz dex allMids API error (status %d): %s", resp.StatusCode, string(body))
	}

	var mids map[string]string
	if err := json.Unmarshal(body, &mids); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// The API returns keys with xyz: prefix, so ensure the coin has it
	lookupKey := coin
	if !strings.HasPrefix(lookupKey, "xyz:") {
		lookupKey = "xyz:" + lookupKey
	}

	if priceStr, ok := mids[lookupKey]; ok {
		priceFloat, err := strconv.ParseFloat(priceStr, 64)
		if err == nil {
			return priceFloat, nil
		}
		return 0, fmt.Errorf("price format error: %v", err)
	}

	return 0, fmt.Errorf("xyz dex price not found for %s (lookup key: %s)", coin, lookupKey)
}

// GetOrderStatus gets order status
// Hyperliquid uses IOC orders, usually filled or cancelled immediately.
// The order id is matched against open orders (NEW) and the user fills
// history (FILLED with real avg price / executed qty / commission). An
// order in neither set is CANCELED (IOC orders that don't fill are
// immediately cancelled by the exchange). Query failures return an error
// instead of guessing FILLED, so network faults can never masquerade as
// successful fills.
func (t *HyperliquidTrader) GetOrderStatus(symbol string, orderID string) (*types.OrderStatus, error) {
	coin := convertSymbolToHyperliquid(symbol)

	// First check if in open orders
	openOrders, err := t.exchange.Info().OpenOrders(t.ctx, t.walletAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to query open orders: %w", err)
	}
	for _, order := range openOrders {
		if order.Coin == coin && fmt.Sprintf("%d", order.Oid) == orderID {
			// Order is still pending
			return &types.OrderStatus{
				OrderID:     orderID,
				Status:      "NEW",
				AvgPrice:    0.0,
				ExecutedQty: 0.0,
				Commission:  0.0,
			}, nil
		}
	}

	// Not in open orders: look for the order in the fills history. IOC
	// orders fill or cancel immediately, so the last 24h window covers it.
	fills, err := t.exchange.Info().UserFillsByTime(t.ctx, t.walletAddr, time.Now().Add(-24*time.Hour).UnixMilli(), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query fills history: %w", err)
	}
	for _, fill := range fills {
		if fill.Oid != 0 && fmt.Sprintf("%d", fill.Oid) == orderID {
			price, _ := strconv.ParseFloat(fill.Price, 64)
			qty, _ := strconv.ParseFloat(fill.Size, 64)
			fee, _ := strconv.ParseFloat(fill.Fee, 64)
			return &types.OrderStatus{
				OrderID:     orderID,
				Status:      "FILLED",
				AvgPrice:    price,
				ExecutedQty: qty,
				Commission:  fee,
			}, nil
		}
	}

	// In neither set: an IOC order that did not fill was cancelled by the
	// exchange at placement time.
	return &types.OrderStatus{
		OrderID:     orderID,
		Status:      "CANCELED",
		AvgPrice:    0.0,
		ExecutedQty: 0.0,
		Commission:  0.0,
	}, nil
}

// GetClosedPnL gets recent closing trades from Hyperliquid
// Note: Hyperliquid does NOT have a position history API, only fill history.
// This returns individual closing trades for real-time position closure detection.
func (t *HyperliquidTrader) GetClosedPnL(startTime time.Time, limit int) ([]types.ClosedPnLRecord, error) {
	trades, err := t.GetTrades(startTime, limit)
	if err != nil {
		return nil, err
	}

	// Filter only closing trades (realizedPnl != 0)
	var records []types.ClosedPnLRecord
	for _, trade := range trades {
		if trade.RealizedPnL == 0 {
			continue
		}

		// Determine side (Hyperliquid uses one-way mode)
		side := types.SideLong
		if trade.Side == "SELL" || trade.Side == "Sell" {
			side = types.SideLong // Selling closes long
		} else {
			side = types.SideShort // Buying closes short
		}

		// Calculate entry price from PnL
		var entryPrice float64
		if trade.Quantity > 0 {
			if side == types.SideLong {
				entryPrice = trade.Price - trade.RealizedPnL/trade.Quantity
			} else {
				entryPrice = trade.Price + trade.RealizedPnL/trade.Quantity
			}
		}

		records = append(records, types.ClosedPnLRecord{
			Symbol:      trade.Symbol,
			Side:        side,
			EntryPrice:  entryPrice,
			ExitPrice:   trade.Price,
			Quantity:    trade.Quantity,
			RealizedPnL: trade.RealizedPnL,
			Fee:         trade.Fee,
			ExitTime:    trade.Time,
			EntryTime:   trade.Time,
			OrderID:     trade.TradeID,
			ExchangeID:  trade.TradeID,
			CloseType:   "unknown",
		})
	}

	return records, nil
}

// GetTrades retrieves trade history from Hyperliquid
func (t *HyperliquidTrader) GetTrades(startTime time.Time, limit int) ([]types.TradeRecord, error) {
	// Use UserFillsByTime API
	startTimeMs := startTime.UnixMilli()
	fills, err := t.exchange.Info().UserFillsByTime(t.ctx, t.walletAddr, startTimeMs, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user fills: %w", err)
	}

	var trades []types.TradeRecord
	for _, fill := range fills {
		price, _ := strconv.ParseFloat(fill.Price, 64)
		qty, _ := strconv.ParseFloat(fill.Size, 64)
		fee, _ := strconv.ParseFloat(fill.Fee, 64)
		pnl, _ := strconv.ParseFloat(fill.ClosedPnl, 64)

		// Determine side: "B" = Buy, "S" = Sell (or "A" = Ask, "B" = Bid)
		var side string
		if fill.Side == "B" || fill.Side == "Buy" || fill.Side == "bid" {
			side = "BUY"
		} else {
			side = "SELL"
		}

		// Parse Dir field to get order action
		// Hyperliquid Dir values: "Open Long", "Open Short", "Close Long", "Close Short"
		var orderAction string
		switch strings.ToLower(fill.Dir) {
		case "open long":
			orderAction = types.ActionOpenLong
		case "open short":
			orderAction = types.ActionOpenShort
		case "close long":
			orderAction = types.ActionCloseLong
		case "close short":
			orderAction = types.ActionCloseShort
		default:
			// Fallback: use RealizedPnL if Dir is missing/unknown
			if pnl != 0 {
				if side == "BUY" {
					orderAction = types.ActionCloseShort
				} else {
					orderAction = types.ActionCloseLong
				}
			} else {
				if side == "BUY" {
					orderAction = types.ActionOpenLong
				} else {
					orderAction = types.ActionOpenShort
				}
			}
		}

		// Hyperliquid uses one-way mode, so PositionSide is "BOTH"
		trade := types.TradeRecord{
			TradeID:      strconv.FormatInt(fill.Tid, 10),
			Symbol:       fill.Coin,
			Side:         side,
			PositionSide: "BOTH", // Hyperliquid doesn't have hedge mode
			OrderAction:  orderAction,
			Price:        price,
			Quantity:     qty,
			RealizedPnL:  pnl,
			Fee:          fee,
			Time:         time.UnixMilli(fill.Time).UTC(),
		}
		trades = append(trades, trade)
	}

	return trades, nil
}

// GetOpenOrders gets all open/pending orders for a symbol
func (t *HyperliquidTrader) GetOpenOrders(symbol string) ([]types.OpenOrder, error) {
	openOrders, err := t.exchange.Info().OpenOrders(t.ctx, t.walletAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	// The caller passes the canonical symbol (e.g. "BTCUSDT") while the
	// SDK's OpenOrder.Coin is the bare coin ("BTC"); normalize before
	// comparing so coin-margined orders are not filtered out.
	coin := convertSymbolToHyperliquid(symbol)

	var result []types.OpenOrder
	for _, order := range openOrders {
		if order.Coin != coin {
			continue
		}

		side := "BUY"
		if order.Side == "A" {
			side = "SELL"
		}

		result = append(result, types.OpenOrder{
			OrderID:      fmt.Sprintf("%d", order.Oid),
			Symbol:       symbol,
			Side:         side,
			PositionSide: "",
			Type:         "LIMIT",
			Price:        order.LimitPx,
			StopPrice:    0,
			Quantity:     order.Size,
			Status:       "NEW",
		})
	}

	return result, nil
}

// GetOrderBook gets the order book for a symbol
// Implements GridTrader interface
func (t *HyperliquidTrader) GetOrderBook(symbol string, depth int) (bids, asks [][]float64, err error) {
	coin := convertSymbolToHyperliquid(symbol)

	l2Book, err := t.exchange.Info().L2Snapshot(t.ctx, coin)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get order book: %w", err)
	}

	if l2Book == nil || len(l2Book.Levels) < 2 {
		return nil, nil, fmt.Errorf("invalid order book data")
	}

	// Parse bids (first level array)
	for i, level := range l2Book.Levels[0] {
		if i >= depth {
			break
		}
		bids = append(bids, []float64{level.Px, level.Sz})
	}

	// Parse asks (second level array)
	for i, level := range l2Book.Levels[1] {
		if i >= depth {
			break
		}
		asks = append(asks, []float64{level.Px, level.Sz})
	}

	return bids, asks, nil
}
