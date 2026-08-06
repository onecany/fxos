package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"fxos/config"
	"fxos/api/schema"
	"fxos/crypto"
	"fxos/logger"
	"fxos/store"

	"github.com/gin-gonic/gin"
)


// schema.SafeExchangeConfig Safe exchange configuration structure (does not contain sensitive information)

func safeExchangeConfigFromStore(exchange *store.Exchange) schema.SafeExchangeConfig {
	return schema.SafeExchangeConfig{
		ID:                         exchange.ID,
		ExchangeType:               exchange.ExchangeType,
		AccountName:                exchange.AccountName,
		Name:                       exchange.Name,
		Type:                       exchange.Type,
		Enabled:                    exchange.Enabled,
		HasAPIKey:                  exchange.APIKey != "",
		HasSecretKey:               exchange.SecretKey != "",
		HasPassphrase:              exchange.Passphrase != "",
		Testnet:                    exchange.Testnet,
		HyperliquidWalletAddr:      exchange.HyperliquidWalletAddr,
		HyperliquidUnifiedAcct:     exchange.HyperliquidUnifiedAcct,
		HyperliquidBuilderApproved: exchange.HyperliquidBuilderApproved,
		HasAsterPrivateKey:         exchange.AsterPrivateKey != "",
		AsterUser:                  exchange.AsterUser,
		AsterSigner:                exchange.AsterSigner,
		LighterWalletAddr:          exchange.LighterWalletAddr,
		HasLighterPrivateKey:       exchange.LighterPrivateKey != "",
		HasLighterAPIKey:           exchange.LighterAPIKeyPrivateKey != "",
	}
}

// schema.ExchangeConfigUpdate is a single exchange account's update payload. It is a
// named type (rather than an inline anonymous struct) so the log-sanitizer in
// utils.go is guaranteed to cover every sensitive field — a drift between the
// two shapes is what let passphrases / private keys reach the logs previously.


// schema.CreateExchangeRequest request structure for creating a new exchange account

// handleGetExchangeConfigs Get exchange configurations
func (s *Server) handleGetExchangeConfigs(c *gin.Context) {
	userID := c.GetString("user_id")
	logger.Infof("🔍 Querying exchange configs for user %s", userID)
	exchanges, err := s.store.Exchange().List(userID)
	if err != nil {
		SafeInternalError(c, "Failed to get exchange configs", err)
		return
	}

	// If no exchanges in database, return empty array (user needs to create accounts)
	if len(exchanges) == 0 {
		logger.Infof("⚠️ No exchanges in database for user %s", userID)
		c.JSON(http.StatusOK, []schema.SafeExchangeConfig{})
		return
	}

	logger.Infof("✅ Found %d exchange configs", len(exchanges))

	// Convert to safe response structure, remove sensitive information
	safeExchanges := make([]schema.SafeExchangeConfig, 0, len(exchanges))
	for _, exchange := range exchanges {
		if !store.IsVisibleExchange(exchange) {
			continue
		}
		safeExchanges = append(safeExchanges, safeExchangeConfigFromStore(exchange))
	}

	c.JSON(http.StatusOK, safeExchanges)
}

func effectiveHyperliquidUnifiedAccount(exchangeType string, requested *bool, fallback ...bool) bool {
	if requested != nil {
		return *requested
	}
	if strings.EqualFold(exchangeType, "hyperliquid") {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return true
	}
	return false
}

// handleUpdateExchangeConfigs Update exchange configurations (supports both encrypted and plain text based on config)
func (s *Server) handleUpdateExchangeConfigs(c *gin.Context) {
	userID := c.GetString("user_id")
	cfg := config.Get()

	// Read raw request body
	bodyBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	var req schema.UpdateExchangeConfigRequest

	// Check if transport encryption is enabled
	if !cfg.TransportEncryption {
		// Transport encryption disabled, accept plain JSON
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			logger.Infof("❌ Failed to parse plain JSON request: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}
		logger.Infof("📝 Received plain text exchange config (UserID: %s)", userID)
	} else {
		// Transport encryption enabled, require encrypted payload
		var encryptedPayload crypto.EncryptedPayload
		if err := json.Unmarshal(bodyBytes, &encryptedPayload); err != nil {
			logger.Infof("❌ Failed to parse encrypted payload: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format, encrypted transmission required"})
			return
		}

		// Verify encrypted data
		if encryptedPayload.WrappedKey == "" {
			logger.Infof("❌ Detected unencrypted request (UserID: %s)", userID)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "This endpoint only supports encrypted transmission, please use encrypted client",
				"code":    "ENCRYPTION_REQUIRED",
				"message": "Encrypted transmission is required for security reasons",
			})
			return
		}

		// Decrypt data
		decrypted, err := s.cryptoHandler.cryptoService.DecryptSensitiveData(&encryptedPayload)
		if err != nil {
			logger.Infof("❌ Failed to decrypt exchange config (UserID: %s): %v", userID, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decrypt data"})
			return
		}

		// Parse decrypted data
		if err := json.Unmarshal([]byte(decrypted), &req); err != nil {
			logger.Infof("❌ Failed to parse decrypted data: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse decrypted data"})
			return
		}
		logger.Infof("🔓 Decrypted exchange config data (UserID: %s)", userID)
	}

	// Update each exchange's configuration and track traders that need reload
	tradersToReload := make(map[string]bool)
	for exchangeID, exchangeData := range req.Exchanges {
		existing, err := s.store.Exchange().GetByID(userID, exchangeID)
		if err != nil {
			SafeInternalError(c, fmt.Sprintf("Load exchange %s", exchangeID), err)
			return
		}
		effectiveAPIKey := strings.TrimSpace(exchangeData.APIKey)
		if effectiveAPIKey == "" {
			effectiveAPIKey = strings.TrimSpace(string(existing.APIKey))
		}
		effectiveSecretKey := strings.TrimSpace(exchangeData.SecretKey)
		if effectiveSecretKey == "" {
			effectiveSecretKey = strings.TrimSpace(string(existing.SecretKey))
		}
		effectivePassphrase := strings.TrimSpace(exchangeData.Passphrase)
		if effectivePassphrase == "" {
			effectivePassphrase = strings.TrimSpace(string(existing.Passphrase))
		}
		effectiveAsterPrivateKey := strings.TrimSpace(exchangeData.AsterPrivateKey)
		if effectiveAsterPrivateKey == "" {
			effectiveAsterPrivateKey = strings.TrimSpace(string(existing.AsterPrivateKey))
		}
		effectiveLighterAPIKeyPrivateKey := strings.TrimSpace(exchangeData.LighterAPIKeyPrivateKey)
		if effectiveLighterAPIKeyPrivateKey == "" {
			effectiveLighterAPIKeyPrivateKey = strings.TrimSpace(string(existing.LighterAPIKeyPrivateKey))
		}
		effectiveHyperliquidWalletAddr := strings.TrimSpace(exchangeData.HyperliquidWalletAddr)
		if effectiveHyperliquidWalletAddr == "" {
			effectiveHyperliquidWalletAddr = strings.TrimSpace(existing.HyperliquidWalletAddr)
		}
		effectiveAsterUser := strings.TrimSpace(exchangeData.AsterUser)
		if effectiveAsterUser == "" {
			effectiveAsterUser = strings.TrimSpace(existing.AsterUser)
		}
		effectiveAsterSigner := strings.TrimSpace(exchangeData.AsterSigner)
		if effectiveAsterSigner == "" {
			effectiveAsterSigner = strings.TrimSpace(existing.AsterSigner)
		}
		effectiveLighterWalletAddr := strings.TrimSpace(exchangeData.LighterWalletAddr)
		if effectiveLighterWalletAddr == "" {
			effectiveLighterWalletAddr = strings.TrimSpace(existing.LighterWalletAddr)
		}
		effectiveHyperliquidBuilderApproved := existing.HyperliquidBuilderApproved
		if exchangeData.HyperliquidBuilderApproved != nil {
			effectiveHyperliquidBuilderApproved = *exchangeData.HyperliquidBuilderApproved
		}
		effectiveHyperliquidUnifiedAcct := effectiveHyperliquidUnifiedAccount(
			existing.ExchangeType,
			exchangeData.HyperliquidUnifiedAcct,
			existing.HyperliquidUnifiedAcct,
		)

		if missing := store.MissingRequiredExchangeCredentialFields(
			existing.ExchangeType,
			effectiveAPIKey,
			effectiveSecretKey,
			effectivePassphrase,
			effectiveHyperliquidWalletAddr,
			effectiveAsterUser,
			effectiveAsterSigner,
			effectiveAsterPrivateKey,
			effectiveLighterWalletAddr,
			effectiveLighterAPIKeyPrivateKey,
		); len(missing) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":          fmt.Sprintf("Missing required exchange fields: %s", strings.Join(missing, ", ")),
				"missing_fields": missing,
			})
			return
		}

		// Find traders using this exchange BEFORE updating
		traders, _ := s.store.Trader().ListByExchangeID(userID, exchangeID)
		for _, t := range traders {
			tradersToReload[t.ID] = true
		}

		err = s.store.Exchange().Update(userID, exchangeID, true, exchangeData.APIKey, exchangeData.SecretKey, exchangeData.Passphrase, exchangeData.Testnet, effectiveHyperliquidWalletAddr, effectiveHyperliquidUnifiedAcct, effectiveHyperliquidBuilderApproved, effectiveAsterUser, effectiveAsterSigner, exchangeData.AsterPrivateKey, effectiveLighterWalletAddr, exchangeData.LighterPrivateKey, exchangeData.LighterAPIKeyPrivateKey, exchangeData.LighterAPIKeyIndex)
		if err != nil {
			SafeInternalError(c, fmt.Sprintf("Update exchange %s", exchangeID), err)
			return
		}
	}

	s.exchangeAccountStateCache.Invalidate(userID)

	// Remove affected traders from memory BEFORE reloading to pick up new config
	for traderID := range tradersToReload {
		logger.Infof("🔄 Removing trader %s from memory to reload with new exchange config", traderID)
		s.traderManager.RemoveTrader(traderID)
	}

	// Reload all traders for this user to make new config take effect immediately
	err = s.traderManager.LoadUserTradersFromStore(s.store, userID)
	if err != nil {
		logger.Infof("⚠️ Failed to reload user traders into memory: %v", err)
		// Don't return error here since exchange config was successfully updated to database
	}

	logger.Infof("✓ Exchange config updated: %+v", SanitizeExchangeConfigForLog(req.Exchanges))
	c.JSON(http.StatusOK, gin.H{"message": "Exchange configuration updated"})
}

// handleCreateExchange Create a new exchange account
func (s *Server) handleCreateExchange(c *gin.Context) {
	userID := c.GetString("user_id")
	cfg := config.Get()

	// Read raw request body
	bodyBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	var req schema.CreateExchangeRequest

	// Check if transport encryption is enabled
	if !cfg.TransportEncryption {
		// Transport encryption disabled, accept plain JSON
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			logger.Infof("❌ Failed to parse plain JSON request: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}
	} else {
		// Transport encryption enabled, require encrypted payload
		var encryptedPayload crypto.EncryptedPayload
		if err := json.Unmarshal(bodyBytes, &encryptedPayload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format, encrypted transmission required"})
			return
		}

		if encryptedPayload.WrappedKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "This endpoint only supports encrypted transmission",
				"code":    "ENCRYPTION_REQUIRED",
				"message": "Encrypted transmission is required for security reasons",
			})
			return
		}

		decrypted, err := s.cryptoHandler.cryptoService.DecryptSensitiveData(&encryptedPayload)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decrypt data"})
			return
		}

		if err := json.Unmarshal([]byte(decrypted), &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse decrypted data"})
			return
		}
	}

	// Validate exchange type
	validTypes := map[string]bool{
		"binance": true, "bybit": true, "okx": true, "bitget": true,
		"hyperliquid": true, "aster": true, "lighter": true, "gate": true, "kucoin": true, "indodax": true,
	}
	if !validTypes[req.ExchangeType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid exchange type: %s", req.ExchangeType)})
		return
	}
	if missing := store.MissingRequiredExchangeCredentialFields(
		req.ExchangeType,
		req.APIKey,
		req.SecretKey,
		req.Passphrase,
		req.HyperliquidWalletAddr,
		req.AsterUser,
		req.AsterSigner,
		req.AsterPrivateKey,
		req.LighterWalletAddr,
		req.LighterAPIKeyPrivateKey,
	); len(missing) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          fmt.Sprintf("Missing required exchange fields: %s", strings.Join(missing, ", ")),
			"missing_fields": missing,
		})
		return
	}

	// Exchange configs only persist once complete; persisted configs are always enabled.
	effectiveHyperliquidUnifiedAcct := effectiveHyperliquidUnifiedAccount(req.ExchangeType, req.HyperliquidUnifiedAcct)
	id, err := s.store.Exchange().Create(
		userID, req.ExchangeType, req.AccountName, true,
		req.APIKey, req.SecretKey, req.Passphrase, req.Testnet,
		req.HyperliquidWalletAddr, effectiveHyperliquidUnifiedAcct, req.HyperliquidBuilderApproved,
		req.AsterUser, req.AsterSigner, req.AsterPrivateKey,
		req.LighterWalletAddr, req.LighterPrivateKey, req.LighterAPIKeyPrivateKey, req.LighterAPIKeyIndex,
	)
	if err != nil {
		logger.Infof("❌ Failed to create exchange account: %v", err)
		SafeInternalError(c, "Failed to create exchange account", err)
		return
	}

	s.exchangeAccountStateCache.Invalidate(userID)

	logger.Infof("✓ Created exchange account: type=%s, name=%s, id=%s", req.ExchangeType, req.AccountName, id)
	c.JSON(http.StatusOK, gin.H{
		"message": "Exchange account created",
		"id":      id,
	})
}

// handleDeleteExchange Delete an exchange account
func (s *Server) handleDeleteExchange(c *gin.Context) {
	userID := c.GetString("user_id")
	exchangeID := c.Param("id")

	if exchangeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Exchange ID is required"})
		return
	}

	// Check if any traders are using this exchange
	traders, err := s.store.Trader().List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check traders"})
		return
	}

	for _, trader := range traders {
		if trader.ExchangeID == exchangeID {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":       "Cannot delete exchange account that is in use by traders",
				"trader_id":   trader.ID,
				"trader_name": trader.Name,
			})
			return
		}
	}

	// Delete exchange account
	err = s.store.Exchange().Delete(userID, exchangeID)
	if err != nil {
		logger.Infof("❌ Failed to delete exchange account: %v", err)
		SafeInternalError(c, "Failed to delete exchange account", err)
		return
	}

	s.exchangeAccountStateCache.Invalidate(userID)

	logger.Infof("✓ Deleted exchange account: id=%s", exchangeID)
	c.JSON(http.StatusOK, gin.H{"message": "Exchange account deleted"})
}

// handleGetSupportedExchanges Get list of exchanges supported by the system
func (s *Server) handleGetSupportedExchanges(c *gin.Context) {
	// Return static list of supported exchange types
	// Note: ID is empty for supported exchanges (they are templates, not actual accounts)
	supportedExchanges := []schema.SafeExchangeConfig{
		{ExchangeType: "binance", Name: "Binance Futures", Type: "cex"},
		{ExchangeType: "bybit", Name: "Bybit Futures", Type: "cex"},
		{ExchangeType: "okx", Name: "OKX Futures", Type: "cex"},
		{ExchangeType: "gate", Name: "Gate.io Futures", Type: "cex"},
		{ExchangeType: "kucoin", Name: "KuCoin Futures", Type: "cex"},
		{ExchangeType: "hyperliquid", Name: "Hyperliquid", Type: "dex"},
		{ExchangeType: "aster", Name: "Aster DEX", Type: "dex"},
		{ExchangeType: "lighter", Name: "LIGHTER DEX", Type: "dex"},
		{ExchangeType: "alpaca", Name: "Alpaca (US Stocks)", Type: "stock"},
		{ExchangeType: "forex", Name: "Forex (TwelveData)", Type: "forex"},
		{ExchangeType: "metals", Name: "Metals (TwelveData)", Type: "metals"},
	}

	c.JSON(http.StatusOK, supportedExchanges)
}
