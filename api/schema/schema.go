// Package schema holds the API request/response DTO types shared across
// handlers. Splitting them out of package api keeps the wire contract
// (JSON field names) in one place and lets handlers, tests and future
// clients reference the types without importing the whole HTTP stack.
//
// NOTE: these types are the API contract — JSON tags must not change
// without a coordinated frontend update (web/src/types/*.ts mirrors them).
package schema


type ModelConfig struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	Enabled      bool   `json:"enabled"`
	APIKey       string `json:"apiKey,omitempty"`
	CustomAPIURL string `json:"customApiUrl,omitempty"`
}


type SafeModelConfig struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Provider        string `json:"provider"`
	Enabled         bool   `json:"enabled"`
	HasAPIKey       bool   `json:"has_api_key"`
	CustomAPIURL    string `json:"customApiUrl"`    // Custom API URL (usually not sensitive)
	CustomModelName string `json:"customModelName"` // Custom model name (not sensitive)
	WalletAddress   string `json:"walletAddress,omitempty"`
	BalanceUSDC     string `json:"balanceUsdc,omitempty"`
}


type ModelConfigUpdate struct {
	Enabled         bool   `json:"enabled"`
	APIKey          string `json:"api_key"`
	CustomAPIURL    string `json:"custom_api_url"`
	CustomModelName string `json:"custom_model_name"`
}


type UpdateModelConfigRequest struct {
	Models map[string]ModelConfigUpdate `json:"models"`
}


type ExchangeConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // "cex" or "dex"
	Enabled   bool   `json:"enabled"`
	APIKey    string `json:"apiKey,omitempty"`
	SecretKey string `json:"secretKey,omitempty"`
	Testnet   bool   `json:"testnet,omitempty"`
}


type SafeExchangeConfig struct {
	ID                         string `json:"id"`            // UUID
	ExchangeType               string `json:"exchange_type"` // "binance", "bybit", "okx", "hyperliquid", "aster", "lighter"
	AccountName                string `json:"account_name"`  // User-defined account name
	Name                       string `json:"name"`          // Display name
	Type                       string `json:"type"`          // "cex" or "dex"
	Enabled                    bool   `json:"enabled"`
	HasAPIKey                  bool   `json:"has_api_key"`
	HasSecretKey               bool   `json:"has_secret_key"`
	HasPassphrase              bool   `json:"has_passphrase"`
	Testnet                    bool   `json:"testnet,omitempty"`
	HyperliquidWalletAddr      string `json:"hyperliquidWalletAddr"` // Hyperliquid wallet address (not sensitive)
	HyperliquidUnifiedAcct     bool   `json:"hyperliquidUnifiedAccount"`
	HyperliquidBuilderApproved bool   `json:"hyperliquidBuilderApproved"`
	HasAsterPrivateKey         bool   `json:"has_aster_private_key"`
	AsterUser                  string `json:"asterUser"`         // Aster username (not sensitive)
	AsterSigner                string `json:"asterSigner"`       // Aster signer (not sensitive)
	LighterWalletAddr          string `json:"lighterWalletAddr"` // LIGHTER wallet address (not sensitive)
	HasLighterPrivateKey       bool   `json:"has_lighter_private_key"`
	HasLighterAPIKey           bool   `json:"has_lighter_api_key_private_key"`
}


type ExchangeConfigUpdate struct {
	Enabled                    bool   `json:"enabled"`
	APIKey                     string `json:"api_key"`
	SecretKey                  string `json:"secret_key"`
	Passphrase                 string `json:"passphrase"` // OKX specific
	Testnet                    bool   `json:"testnet"`
	HyperliquidWalletAddr      string `json:"hyperliquid_wallet_addr"`
	HyperliquidUnifiedAcct     *bool  `json:"hyperliquid_unified_account"` // Unified Account mode
	HyperliquidBuilderApproved *bool  `json:"hyperliquid_builder_approved"`
	AsterUser                  string `json:"aster_user"`
	AsterSigner                string `json:"aster_signer"`
	AsterPrivateKey            string `json:"aster_private_key"`
	LighterWalletAddr          string `json:"lighter_wallet_addr"`
	LighterPrivateKey          string `json:"lighter_private_key"`
	LighterAPIKeyPrivateKey    string `json:"lighter_api_key_private_key"`
	LighterAPIKeyIndex         int    `json:"lighter_api_key_index"`
}


type UpdateExchangeConfigRequest struct {
	Exchanges map[string]ExchangeConfigUpdate `json:"exchanges"`
}


type CreateExchangeRequest struct {
	ExchangeType               string `json:"exchange_type" binding:"required"` // "binance", "bybit", "okx", "hyperliquid", "aster", "lighter"
	AccountName                string `json:"account_name"`                     // User-defined account name
	Enabled                    bool   `json:"enabled"`
	APIKey                     string `json:"api_key"`
	SecretKey                  string `json:"secret_key"`
	Passphrase                 string `json:"passphrase"`
	Testnet                    bool   `json:"testnet"`
	HyperliquidWalletAddr      string `json:"hyperliquid_wallet_addr"`
	HyperliquidUnifiedAcct     *bool  `json:"hyperliquid_unified_account"` // Unified Account mode: Spot as Perp collateral
	HyperliquidBuilderApproved bool   `json:"hyperliquid_builder_approved"`
	AsterUser                  string `json:"aster_user"`
	AsterSigner                string `json:"aster_signer"`
	AsterPrivateKey            string `json:"aster_private_key"`
	LighterWalletAddr          string `json:"lighter_wallet_addr"`
	LighterPrivateKey          string `json:"lighter_private_key"`
	LighterAPIKeyPrivateKey    string `json:"lighter_api_key_private_key"`
	LighterAPIKeyIndex         int    `json:"lighter_api_key_index"`
}


type CreateTraderRequest struct {
	Name                string  `json:"name" binding:"required"`
	AIModelID           string  `json:"ai_model_id" binding:"required"`
	ExchangeID          string  `json:"exchange_id" binding:"required"`
	StrategyID          string  `json:"strategy_id"` // Strategy ID (new version)
	InitialBalance      float64 `json:"initial_balance"`
	ScanIntervalMinutes int     `json:"scan_interval_minutes"`
	IsCrossMargin       *bool   `json:"is_cross_margin"`     // Pointer type, nil means use default value true
	ShowInCompetition   *bool   `json:"show_in_competition"` // Pointer type, nil means use default value true
	// The following fields are kept for backward compatibility, new version uses strategy config
	BTCETHLeverage       int    `json:"btc_eth_leverage"`
	AltcoinLeverage      int    `json:"altcoin_leverage"`
	TradingSymbols       string `json:"trading_symbols"`
	CustomPrompt         string `json:"custom_prompt"`
	OverrideBasePrompt   bool   `json:"override_base_prompt"`
	SystemPromptTemplate string `json:"system_prompt_template"` // System prompt template name
	UseAI500             bool   `json:"use_ai500"`
	UseOITop             bool   `json:"use_oi_top"`
}


type UpdateTraderRequest struct {
	Name                string  `json:"name" binding:"required"`
	AIModelID           string  `json:"ai_model_id" binding:"required"`
	ExchangeID          string  `json:"exchange_id" binding:"required"`
	StrategyID          string  `json:"strategy_id"` // Strategy ID (new version)
	InitialBalance      float64 `json:"initial_balance"`
	ScanIntervalMinutes int     `json:"scan_interval_minutes"`
	IsCrossMargin       *bool   `json:"is_cross_margin"`
	ShowInCompetition   *bool   `json:"show_in_competition"`
	// The following fields are kept for backward compatibility, new version uses strategy config
	BTCETHLeverage       int    `json:"btc_eth_leverage"`
	AltcoinLeverage      int    `json:"altcoin_leverage"`
	TradingSymbols       string `json:"trading_symbols"`
	CustomPrompt         string `json:"custom_prompt"`
	OverrideBasePrompt   bool   `json:"override_base_prompt"`
	SystemPromptTemplate string `json:"system_prompt_template"`
}


type APIErrorResponse struct {
	Error      string            `json:"error"`
	ErrorKey   string            `json:"error_key,omitempty"`
	ErrorParams map[string]string `json:"error_params,omitempty"`
}
