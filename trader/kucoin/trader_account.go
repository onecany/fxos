package kucoin

import (
	"encoding/json"
	"fmt"
	"fxos/logger"
	"fxos/trader/types"
	"time"
)

// GetBalance gets account balance
func (t *KuCoinTrader) GetBalance() (*types.Account, error) {
	// Check cache
	t.balanceCacheMutex.RLock()
	if t.cachedBalance != nil && time.Since(t.balanceCacheTime) < t.cacheDuration {
		t.balanceCacheMutex.RUnlock()
		return t.cachedBalance, nil
	}
	t.balanceCacheMutex.RUnlock()

	data, err := t.doRequest("GET", kucoinAccountPath+"?currency=USDT", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get account balance: %w", err)
	}

	var account struct {
		AccountEquity    float64 `json:"accountEquity"`
		UnrealisedPNL    float64 `json:"unrealisedPNL"`
		MarginBalance    float64 `json:"marginBalance"`
		PositionMargin   float64 `json:"positionMargin"`
		OrderMargin      float64 `json:"orderMargin"`
		FrozenFunds      float64 `json:"frozenFunds"`
		AvailableBalance float64 `json:"availableBalance"`
		Currency         string  `json:"currency"`
	}

	if err := json.Unmarshal(data, &account); err != nil {
		return nil, fmt.Errorf("failed to parse balance data: %w", err)
	}

	result := &types.Account{
		TotalWalletBalance:    account.MarginBalance, // Wallet balance (without unrealized PnL)
		AvailableBalance:      account.AvailableBalance,
		TotalUnrealizedProfit: account.UnrealisedPNL,
		TotalEquity:           account.AccountEquity,
	}

	logger.Infof("✓ KuCoin balance: Total equity=%.2f, Available=%.2f, Unrealized PnL=%.2f",
		account.AccountEquity, account.AvailableBalance, account.UnrealisedPNL)

	// Update cache
	t.balanceCacheMutex.Lock()
	t.cachedBalance = result
	t.balanceCacheTime = time.Now()
	t.balanceCacheMutex.Unlock()

	return result, nil
}
