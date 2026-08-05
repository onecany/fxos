package trader

import (
	"fmt"
	"fxos/logger"
	"fxos/trader/aster"
	"fxos/trader/binance"
	"fxos/trader/bitget"
	"fxos/trader/bybit"
	"fxos/trader/gate"
	"fxos/trader/hyperliquid"
	"fxos/trader/indodax"
	"fxos/trader/kucoin"
	"fxos/trader/lighter"
	"fxos/trader/okx"
)

// ExchangeFactory constructs an exchange adapter from the trader config.
// The factory owns exchange-specific initialization (validation, network
// calls, error wrapping) so NewAutoTrader does not need a switch per
// exchange.
type ExchangeFactory func(cfg AutoTraderConfig, userID string) (Trader, error)

var exchangeRegistry = make(map[string]ExchangeFactory)

// RegisterExchange registers an exchange factory by its exchange type name.
// Registration happens in init() below; tests may register additional
// factories before creating traders.
func RegisterExchange(name string, factory ExchangeFactory) {
	exchangeRegistry[name] = factory
}

// CreateTrader creates an exchange adapter via the registry. Unknown
// exchange names return an error instead of falling through a switch.
func CreateTrader(exchange string, cfg AutoTraderConfig, userID string) (Trader, error) {
	factory, ok := exchangeRegistry[exchange]
	if !ok {
		return nil, fmt.Errorf("unsupported trading platform: %s", exchange)
	}
	return factory(cfg, userID)
}

func init() {
	RegisterExchange("binance", func(cfg AutoTraderConfig, userID string) (Trader, error) {
		logger.Infof("🏦 [%s] Using Binance Futures trading", cfg.Name)
		return binance.NewFuturesTrader(cfg.Credentials.Binance.APIKey, cfg.Credentials.Binance.SecretKey, userID), nil
	})
	RegisterExchange("bybit", func(cfg AutoTraderConfig, userID string) (Trader, error) {
		logger.Infof("🏦 [%s] Using Bybit Futures trading", cfg.Name)
		return bybit.NewBybitTrader(cfg.Credentials.Bybit.APIKey, cfg.Credentials.Bybit.SecretKey), nil
	})
	RegisterExchange("okx", func(cfg AutoTraderConfig, userID string) (Trader, error) {
		logger.Infof("🏦 [%s] Using OKX Futures trading", cfg.Name)
		return okx.NewOKXTrader(cfg.Credentials.OKX.APIKey, cfg.Credentials.OKX.SecretKey, cfg.Credentials.OKX.Passphrase), nil
	})
	RegisterExchange("bitget", func(cfg AutoTraderConfig, userID string) (Trader, error) {
		logger.Infof("🏦 [%s] Using Bitget Futures trading", cfg.Name)
		return bitget.NewBitgetTrader(cfg.Credentials.Bitget.APIKey, cfg.Credentials.Bitget.SecretKey, cfg.Credentials.Bitget.Passphrase), nil
	})
	RegisterExchange("gate", func(cfg AutoTraderConfig, userID string) (Trader, error) {
		logger.Infof("🏦 [%s] Using Gate.io Futures trading", cfg.Name)
		return gate.NewGateTrader(cfg.Credentials.Gate.APIKey, cfg.Credentials.Gate.SecretKey), nil
	})
	RegisterExchange("kucoin", func(cfg AutoTraderConfig, userID string) (Trader, error) {
		logger.Infof("🏦 [%s] Using KuCoin Futures trading", cfg.Name)
		return kucoin.NewKuCoinTrader(cfg.Credentials.KuCoin.APIKey, cfg.Credentials.KuCoin.SecretKey, cfg.Credentials.KuCoin.Passphrase), nil
	})
	RegisterExchange("hyperliquid", func(cfg AutoTraderConfig, userID string) (Trader, error) {
		logger.Infof("🏦 [%s] Using Hyperliquid trading", cfg.Name)
		creds := cfg.Credentials.Hyperliquid
		trader, err := hyperliquid.NewHyperliquidTrader(creds.PrivateKey, creds.WalletAddr, creds.Testnet, creds.UnifiedAcct)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Hyperliquid trader: %w", err)
		}
		return trader, nil
	})
	RegisterExchange("aster", func(cfg AutoTraderConfig, userID string) (Trader, error) {
		logger.Infof("🏦 [%s] Using Aster trading", cfg.Name)
		creds := cfg.Credentials.Aster
		trader, err := aster.NewAsterTrader(creds.User, creds.Signer, creds.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Aster trader: %w", err)
		}
		return trader, nil
	})
	RegisterExchange("lighter", func(cfg AutoTraderConfig, userID string) (Trader, error) {
		logger.Infof("🏦 [%s] Using LIGHTER trading", cfg.Name)
		creds := cfg.Credentials.Lighter
		if creds.WalletAddr == "" || creds.APIKeyPrivateKey == "" {
			return nil, fmt.Errorf("Lighter requires wallet address and API Key private key")
		}
		// Lighter only supports mainnet (testnet disabled)
		trader, err := lighter.NewLighterTraderV2(creds.WalletAddr, creds.APIKeyPrivateKey, creds.APIKeyIndex, false)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize LIGHTER trader: %w", err)
		}
		logger.Infof("✓ LIGHTER trader initialized successfully")
		return trader, nil
	})
	RegisterExchange("indodax", func(cfg AutoTraderConfig, userID string) (Trader, error) {
		logger.Infof("🏦 [%s] Using Indodax Spot trading", cfg.Name)
		return indodax.NewIndodaxTrader(cfg.Credentials.Indodax.APIKey, cfg.Credentials.Indodax.SecretKey), nil
	})
}
