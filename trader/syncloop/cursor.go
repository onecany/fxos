package syncloop

import (
	"sync"
	"time"

	"fxos/logger"
	"fxos/store"
)

// initialSyncLookback is how far back the first sync goes when no cursor
// exists in memory or in the database.
const initialSyncLookback = 24 * time.Hour

// syncCursorBufferMs is added to the recovered DB time so the same fill is
// not re-fetched on restart.
const syncCursorBufferMs = 1000

// SyncCursor tracks the last-synced trade timestamp per exchange account,
// replacing the per-exchange package-level maps (binanceSyncState etc.).
// The state is kept in memory and recovered from the database on first use
// so a restart resumes from where the previous run left off.
type SyncCursor struct {
	mu    sync.RWMutex
	state map[string]int64 // exchangeID -> last sync time (Unix ms)
}

// NewSyncCursor creates an empty cursor.
func NewSyncCursor() *SyncCursor {
	return &SyncCursor{state: make(map[string]int64)}
}

// GetOrInit returns the last sync time (Unix ms) for the exchange account,
// restoring from the database if needed and defaulting to initialSyncLookback
// ago on first-ever sync. A recovered time in the future (clock skew or bad
// data) falls back to the default lookback.
func (c *SyncCursor) GetOrInit(exchangeID string, st *store.Store, nowMs int64) int64 {
	c.mu.RLock()
	lastSyncTimeMs, exists := c.state[exchangeID]
	c.mu.RUnlock()
	if exists {
		return lastSyncTimeMs
	}

	// Try to get last fill time from database (persist across restarts)
	defaultTime := nowMs - initialSyncLookback.Milliseconds()
	if st == nil {
		return defaultTime
	}
	lastFillTimeMs, err := st.Order().GetLastFillTimeByExchange(exchangeID)
	if err == nil && lastFillTimeMs > 0 {
		if lastFillTimeMs > nowMs {
			logger.Infof("⚠️ DB sync time %d is in the future (now: %d), using default",
				lastFillTimeMs, nowMs)
			return defaultTime
		}
		// Add buffer to avoid re-fetching the same fill
		lastSyncTimeMs = lastFillTimeMs + syncCursorBufferMs
		logger.Infof("📅 Recovered last sync time from DB: %s (UTC)",
			time.UnixMilli(lastSyncTimeMs).UTC().Format("2006-01-02 15:04:05"))
		c.mu.Lock()
		c.state[exchangeID] = lastSyncTimeMs
		c.mu.Unlock()
		return lastSyncTimeMs
	}

	logger.Infof("📅 First sync, starting from 24 hours ago: %s (UTC)",
		time.UnixMilli(defaultTime).UTC().Format("2006-01-02 15:04:05"))
	c.mu.Lock()
	c.state[exchangeID] = defaultTime
	c.mu.Unlock()
	return defaultTime
}

// Advance moves the cursor to the latest processed trade time. The caller
// must only advance on a fully successful sync so failed syncs retry from
// the same position.
func (c *SyncCursor) Advance(exchangeID string, latestTradeTimeMs int64) {
	c.mu.Lock()
	c.state[exchangeID] = latestTradeTimeMs
	c.mu.Unlock()
	logger.Infof("📅 Updated lastSyncTime to latest trade: %s (UTC)",
		time.UnixMilli(latestTradeTimeMs).UTC().Format("2006-01-02 15:04:05"))
}
