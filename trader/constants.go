package trader

import "time"

// Engine timing constants (operational parameters).
const (
	// OrderSyncInterval is how often the background order/position sync
	// loop runs in AutoTrader.Run().
	OrderSyncInterval = 30 * time.Second

	// DailyPnLResetInterval is the rolling window after which the daily
	// P&L anchor (dayStartEquity) is re-established.
	DailyPnLResetInterval = 24 * time.Hour

	// OrderStatusPollInterval and OrderStatusPollAttempts drive the
	// fill-confirmation polling in recordAndConfirmOrder.
	OrderStatusPollInterval  = 500 * time.Millisecond
	OrderStatusPollAttempts  = 5
	PostExecutionDelay       = 1 * time.Second
)

// Risk control defaults, used when the strategy config does not provide a
// value. These guard actual money, so they live here as named constants
// instead of inline literals.
const (
	// DefaultLeverageFallback is used when a position has no leverage data.
	DefaultLeverageFallback = 10

	// Breakeven/trailing stop thresholds in profit percent.
	BreakevenThresholdBTCETH  = 1.5 // 1.5% profit = ~1x risk for typical 1.5% stop
	TrailingThresholdBTCETH   = 3.0 // 3% profit = ~2x risk
	BreakevenThresholdAltcoin = 2.5 // 2.5% profit for altcoins (higher vol)
	TrailingThresholdAltcoin  = 5.0 // 5% profit for altcoins
	TrailingDrawdownPct       = 50.0 // trail at 50% from peak

	// Max position value as a multiple of equity.
	DefaultMaxPositionValueRatioBTCETH  = 5.0 // 5x for BTC/ETH and XYZ assets
	DefaultMaxPositionValueRatioAltcoin = 1.0 // 1x for altcoins

	// DefaultMaxPositions fallback for enforceMaxPositions.
	DefaultMaxPositions = 3
)

// Grid algorithm parameters (tuned constants for grid level sizing).
const (
	// GridLevelSpacingBase is the base spacing multiplier across grid levels.
	GridLevelSpacingBase = 0.03
	// GridLevelSpacingScale is the divisor scaling spacing by grid count.
	GridLevelSpacingScale = 10
	// GridLevelSpacingMax caps the spacing multiplier.
	GridLevelSpacingMax = 2.0
	// GridBiasRatioFallback is the default price bias for level placement.
	GridBiasRatioFallback = 0.7
	// GridBreakoutThresholdStrong / Mild for regime-based breakout detection (percent).
	GridBreakoutThresholdStrong = 2.0
	GridBreakoutThresholdMild   = 1.0
)
