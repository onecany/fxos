package types

// Typed aliases for the domain vocabulary. The alias form (type X = string)
// deliberately does not create new types: existing string fields in kernel,
// store and api layers keep compiling and the JSON/DB wire formats stay
// byte-identical ("long", "open_long", "filled", ...). The constants are
// the canonical spellings — grep should find zero bare strings once the
// migration completes, and callers that invent values (e.g. "longg") stand
// out immediately.

// Side is the position direction as used in decisions, positions and API
// payloads.
type Side = string

const (
	SideLong  Side = "long"
	SideShort Side = "short"
)

// Action is the trading action recorded in decision records and stored in
// the decision_actions table.
type Action = string

const (
	ActionOpenLong      Action = "open_long"
	ActionOpenShort     Action = "open_short"
	ActionCloseLong     Action = "close_long"
	ActionCloseShort    Action = "close_short"
	ActionSetStopLoss   Action = "set_stop_loss"
	ActionSetTakeProfit Action = "set_take_profit"
	ActionModify        Action = "modify"
	ActionHold          Action = "hold"
	ActionWait          Action = "wait"
)

// CloseType describes why a position was closed (position history records).
type CloseType = string

const (
	CloseTypeUnknown     CloseType = "unknown"
	CloseTypeManual      CloseType = "manual"
	CloseTypeStopLoss    CloseType = "stop_loss"
	CloseTypeTakeProfit  CloseType = "take_profit"
	CloseTypeLiquidation CloseType = "liquidation"
)

// GridState is the lifecycle state of a grid level.
type GridState = string

const (
	GridStatePending   GridState = "pending"
	GridStatePlaced    GridState = "placed"
	GridStateFilled    GridState = "filled"
	GridStateCancelled GridState = "cancelled"
	GridStateFailed    GridState = "failed"
)

// ExchangeName is the exchange type identifier used across config, the
// exchange registry and the API.
type ExchangeName = string

const (
	ExchangeBinance    ExchangeName = "binance"
	ExchangeBybit      ExchangeName = "bybit"
	ExchangeOKX        ExchangeName = "okx"
	ExchangeBitget     ExchangeName = "bitget"
	ExchangeGate       ExchangeName = "gate"
	ExchangeKuCoin     ExchangeName = "kucoin"
	ExchangeHyperliquid ExchangeName = "hyperliquid"
	ExchangeAster      ExchangeName = "aster"
	ExchangeLighter    ExchangeName = "lighter"
	ExchangeIndodax    ExchangeName = "indodax"
)
