import { useEffect, useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import type { CSSProperties } from 'react'
import useSWR from 'swr'
import { api } from '../../lib/api'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import type {
  SystemStatus,
  AccountInfo,
  Position,
  DecisionRecord,
  TraderInfo,
} from '../../types'
import { OrchestrationTopology } from './OrchestrationTopology'
import { OrderBook } from './OrderBook'
import { LiquidationMap } from './LiquidationMap'
import { KlineChart } from './KlineChart'
import { ExecutionLog } from './ExecutionLog'
import { SignalMatrix } from './SignalMatrix'
import { RiskRadar } from './RiskRadar'
import { EdgeProfile } from './EdgeProfile'
import DecisionDetailModal from '../trader/DecisionDetailModal'
import { useDemoEngine } from '../../lib/demo/useDemoEngine'

// crypto majors trade on the Hyperliquid main dex (no hip3 cost/liq heatmap);
// everything else in the universe is an xyz-dex synthetic market that does.
const CRYPTO_MAJORS = new Set([
  'BTC', 'ETH', 'SOL', 'HYPE', 'BNB', 'XRP', 'DOGE', 'AVAX', 'LINK', 'SUI', 'APT', 'ARB', 'OP',
  'TON', 'ADA', 'TRX', 'LTC', 'BCH', 'NEAR', 'INJ', 'SEI', 'TIA', 'PEPE', 'WIF', 'BONK', 'AAVE',
  'UNI', 'ENA', 'ONDO', 'JUP', 'PENDLE', 'KPEPE', 'ZEC', 'XPL', 'LIT',
])

// fixed height for the three row-1 panels so the row stays balanced at any width
const ROW1_H = 500
import { FlowMarkets } from './FlowMarkets'
import './terminal.css'

interface TerminalDashboardProps {
  selectedTrader?: TraderInfo
  traders?: TraderInfo[]
  selectedTraderId?: string
  onTraderSelect: (traderId: string) => void
  status?: SystemStatus
  account?: AccountInfo
  positions?: Position[]
  decisions?: DecisionRecord[]
}

function fmtUsd(n: number | undefined, signed = false): string {
  if (n == null || Number.isNaN(n)) return '—'
  const sign = signed && n > 0 ? '+' : n < 0 ? '-' : ''
  return `${sign}$${Math.abs(n).toLocaleString('en-US', { maximumFractionDigits: 2 })}`
}
function fmtPct(n: number | undefined): string {
  if (n == null || Number.isNaN(n)) return '—'
  return `${n >= 0 ? '+' : ''}${n.toFixed(2)}%`
}
/** Price with magnitude-aware precision: 64,416 · 184.2 · 2.3775 · 0.0067 */
function fmtPx(n: number | undefined): string {
  if (n == null || Number.isNaN(n) || n === 0) return '—'
  const dp = n >= 1000 ? 0 : n >= 100 ? 1 : n >= 1 ? 2 : 4
  return n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp })
}
function baseLabel(raw?: string): string {
  if (!raw) return ''
  return raw.toUpperCase().replace(/^XYZ:/, '').replace(/[-_]/g, '').replace(/(USDT|USDC|USD)$/, '')
}
function parseScanMinutes(scan?: string): number {
  if (!scan) return 15
  const m = scan.match(/(\d+)\s*m/i)
  if (m) return parseInt(m[1], 10)
  const h = scan.match(/(\d+)\s*h/i)
  if (h) return parseInt(h[1], 10) * 60
  const n = parseInt(scan, 10)
  return Number.isFinite(n) && n > 0 ? n : 15
}
function fmtTime(raw?: string | number): string {
  if (raw == null || raw === '') return ''
  let n = typeof raw === 'number' ? raw : Number(raw)
  if (Number.isFinite(n)) {
    if (n < 1e12) n *= 1000
    return new Date(n).toLocaleString('en-GB', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })
  }
  const d = new Date(raw as string)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleString('en-GB', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hour12: false })
}

function useTick(ms = 1000) {
  const [, set] = useState(0)
  useEffect(() => {
    const id = setInterval(() => set((n) => n + 1), ms)
    return () => clearInterval(id)
  }, [ms])
}

export function TerminalDashboard({
  selectedTrader,
  traders,
  selectedTraderId,
  onTraderSelect,
  status: propStatus,
  account: propAccount,
  positions: propPositions,
  decisions: propDecisions,
}: TerminalDashboardProps) {
  const { language } = useLanguage()
  const traderId = selectedTrader?.trader_id || selectedTraderId
  useTick(1000)
  const clock = new Date().toLocaleTimeString('en-GB', { hour12: false })

  const { data: realFullStats } = useSWR(
    traderId ? ['full-stats', traderId] : null,
    () => api.getFullStats(traderId!, true),
    { refreshInterval: 30000, shouldRetryOnError: false }
  )
  const { data: realHistory } = useSWR(
    traderId ? ['pos-history', traderId] : null,
    () => api.getPositionHistory(traderId!, 50, true),
    { refreshInterval: 60000, shouldRetryOnError: false }
  )
  const { data: realConfig } = useSWR(
    traderId ? ['trader-config', traderId] : null,
    () => api.getTraderConfig(traderId!, true),
    { refreshInterval: 120000, shouldRetryOnError: false }
  )
  const { data: realFlow } = useSWR(
    traderId ? ['flow-markets', traderId] : null,
    () => api.getFlowMarkets(selectedTrader?.ai_model, 'mainnet', '1h', 50, true),
    // paid x402 endpoint — poll slowly (5m) to conserve claw402 funds; the
    // topology beam animation is client-side and stays fast regardless
    { refreshInterval: 300000, shouldRetryOnError: false }
  )
  const { data: realSignalRank } = useSWR(
    traderId ? ['signal-rank', traderId] : null,
    () => api.getSignalRanking(selectedTrader?.ai_model, 'mainnet', 'all', 30, true),
    // paid x402 endpoint — poll slowly (5m) to conserve claw402 funds
    { refreshInterval: 300000, shouldRetryOnError: false }
  )

  // Demo / showcase mode for product walkthroughs. Toggle with Shift+D (or the
  // discreet corner dot). Generates a fast, profitable-looking US-equity dataset
  // entirely client-side — it never touches the backend or any real account.
  // When off, real data flows through unchanged.
  const [demo, setDemo] = useState(false)
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.shiftKey && (e.key === 'D' || e.key === 'd')) {
        const el = document.activeElement
        if (el && /^(input|textarea|select)$/i.test(el.tagName)) return
        setDemo((v) => !v)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])
  const sim = useDemoEngine(demo)
  const on = demo && !!sim

  const status = on ? sim!.status : propStatus
  const account = on ? sim!.account : propAccount
  const positions = on ? sim!.positions : propPositions
  const decisions = on ? sim!.decisions : propDecisions
  const fullStats = on ? sim!.fullStats : realFullStats
  const history = on ? sim!.history : realHistory
  const config = on ? (sim!.config as unknown as typeof realConfig) : realConfig
  const flow = on ? sim!.flow : realFlow
  const signalRank = on ? sim!.signalRank : realSignalRank

  const latest = decisions && decisions.length > 0 ? decisions[0] : undefined
  const candidateCoins = latest?.candidate_coins ?? []
  const flowItems = flow?.data?.inflow ?? []

  // Both the cost/liq map and the order book follow this symbol so they stay in
  // sync. The heatmap only covers hip3_perp synthetic markets, so we pick a
  // synthetic (non-crypto) the bot trades — preferring the BUSIEST one (most
  // 1h trades, per flow-markets) so the shared order book ticks as fast as
  // possible. Falls back to any held synthetic, then SP500.
  const heatmapSymbol = useMemo(() => {
    const held = new Set(
      [...(positions ?? []).map((p) => p.symbol), ...candidateCoins]
        .map(baseLabel)
        .filter((b) => b && !CRYPTO_MAJORS.has(b)),
    )
    const synthByActivity = flowItems
      .map((i) => ({ b: baseLabel(i.symbol), trades: i.trades || 0 }))
      .filter((x) => x.b && !CRYPTO_MAJORS.has(x.b))
      .sort((a, b) => b.trades - a.trades)
    const busiestHeld = synthByActivity.find((x) => held.has(x.b))
    if (busiestHeld) return busiestHeld.b
    if (held.size) return [...held][0]
    if (synthByActivity.length) return synthByActivity[0].b
    return 'SP500'
  }, [positions, candidateCoins, flowItems])

  // user can click a signal-matrix cell to drive both the cost/liq map and the
  // order book. Default to the instrument the bot is ACTUALLY holding (first
  // open position, else this cycle's first candidate) so the price panels match
  // the real traded symbol; only fall back to the busiest synthetic if the bot
  // holds nothing.
  const [selectedSym, setSelectedSym] = useState<string | null>(null)
  const [detailDecision, setDetailDecision] = useState<DecisionRecord | null>(null)
  const defaultSym = useMemo(() => {
    // the bot's actual first open position (else this cycle's first candidate);
    // every market — synthetic or crypto — now has a cost/liq heatmap, so no
    // need to prefer one type. Falls back to the busiest synthetic if flat.
    const heldBases = [...(positions ?? []).map((p) => p.symbol), ...candidateCoins].map(baseLabel).filter(Boolean)
    return heldBases[0] || heatmapSymbol || 'SP500'
  }, [positions, candidateCoins, heatmapSymbol])
  const activeSym = (selectedSym || defaultSym).toUpperCase()

  const pnl = account?.total_pnl ?? 0
  const pnlPct = account?.total_pnl_pct ?? 0
  const up = pnl >= 0
  const running = status?.is_running

  // direction per symbol — priority: AI's actual decision > signal bias >
  // net flow > prevailing market majority (never blindly default to long).
  const dirFor = useMemo(() => {
    const dec = new Map<string, 'long' | 'short'>()
    ;(latest?.decisions ?? []).forEach((d) => {
      const b = baseLabel(d.symbol)
      if (d.action === 'open_long' || d.action === 'close_short') dec.set(b, 'long')
      else if (d.action === 'open_short' || d.action === 'close_long') dec.set(b, 'short')
    })
    const sig = new Map<string, 'long' | 'short'>()
    let bull = 0
    let bear = 0
    ;(signalRank?.items ?? []).forEach((s) => {
      const b = baseLabel(s.symbol)
      const bias = (s.bias || '').toLowerCase()
      if (bias === 'bearish') { sig.set(b, 'short'); bear++ }
      else if (bias === 'bullish') { sig.set(b, 'long'); bull++ }
    })
    const fl = new Map<string, 'long' | 'short'>()
    ;(flow?.data?.inflow ?? []).forEach((i) => fl.set(baseLabel(i.symbol), 'long'))
    ;(flow?.data?.outflow ?? []).forEach((i) => fl.set(baseLabel(i.symbol), 'short'))
    const majority: 'long' | 'short' = bear > bull ? 'short' : 'long'
    return (sym: string): 'long' | 'short' => {
      const b = baseLabel(sym)
      return dec.get(b) ?? sig.get(b) ?? fl.get(b) ?? majority
    }
  }, [latest, signalRank, flow])

  const scanMin = config?.scan_interval_minutes || parseScanMinutes(status?.scan_interval)
  const nextCycleMs = useMemo(() => {
    if (!latest?.timestamp) return null
    return new Date(latest.timestamp).getTime() + scanMin * 60_000
  }, [latest?.timestamp, scanMin])
  let countdown = '—'
  if (nextCycleMs) {
    const ms = nextCycleMs - Date.now()
    if (ms <= 0) countdown = 'due now'
    else {
      const s = Math.floor(ms / 1000)
      countdown = `${Math.floor(s / 60)}m ${s % 60}s`
    }
  }

  const recentTrades = (history?.positions ?? []).slice(0, 8)
  const symbolStats = useMemo(
    () => (history?.symbol_stats ?? []).slice().sort((a, b) => b.total_trades - a.total_trades).slice(0, 6),
    [history]
  )
  const maxSymTrades = symbolStats.reduce((m, s) => Math.max(m, s.total_trades), 1)

  const sc: CSSProperties = { padding: '10px 14px' }
  const cellBorder = '1px solid var(--tm-hair)'

  // Portal the trader selector + run status into the global nav so the app has
  // a single top bar (no separate dashboard titlebar).
  const [navSlot, setNavSlot] = useState<HTMLElement | null>(null)
  useEffect(() => {
    setNavSlot(document.getElementById('dash-header-slot'))
  }, [])

  return (
    <div className="fxos-terminal" style={{ minHeight: '100vh', padding: 0 }}>
      {/* discreet, unlabelled showcase toggle (Shift+D also works) */}
      <button
        type="button"
        onClick={() => setDemo((v) => !v)}
        aria-label="toggle presentation mode"
        style={{
          position: 'fixed',
          right: 10,
          bottom: 10,
          zIndex: 9999,
          width: 12,
          height: 12,
          padding: 0,
          borderRadius: '50%',
          border: 'none',
          cursor: 'pointer',
          background: on ? 'var(--tm-up)' : 'color-mix(in srgb, var(--tm-ink) 14%, transparent)',
          opacity: on ? 0.55 : 0.18,
        }}
      />
      {/* centered, capped content column — no border (keeps it from feeling
          embedded) but bounded so the aspect-ratio SVGs don't balloon on wide screens */}
      {navSlot &&
        createPortal(
          <span className="fxos-terminal" style={{ background: 'transparent', display: 'flex', alignItems: 'center', gap: 12, marginLeft: 16, paddingLeft: 16, borderLeft: '1px solid var(--tm-rule)', fontSize: 11 }}>
            <span className="tm-sc" style={{ color: 'var(--tm-muted)' }}>orchestration</span>
            {traders && traders.length > 0 && (
              <select value={traderId || ''} onChange={(e) => onTraderSelect(e.target.value)} className="tm-mono"
                style={{ background: 'var(--tm-panel)', color: 'var(--tm-ink)', border: '1px solid var(--tm-hair)', borderRadius: 0, fontSize: 11, padding: '3px 6px' }}>
                {traders.map((t) => (<option key={t.trader_id} value={t.trader_id} style={{ color: '#111' }}>{t.trader_name}</option>))}
              </select>
            )}
            <span style={{ color: running ? 'var(--tm-up)' : 'var(--tm-muted)' }}>{running ? '● running' : '○ stopped'}</span>
            <span className="tm-sc" style={{ color: 'var(--tm-muted)' }}>cycle</span><span className="tm-mono" style={{ color: 'var(--tm-ink)' }}>{status?.call_count ?? '—'}</span>
            <span className="tm-px" style={{ fontSize: 12, color: 'var(--tm-ink)' }}>{clock}</span>
          </span>,
          navSlot,
        )}
      <div className="tm-box" style={{ maxWidth: 1280, margin: '0 auto', border: 'none' }}>
        {/* runtime health banner — AI fee wallet dry / safe mode would otherwise
            only be visible in server logs while the bot silently idles */}
        {!on && status && (status.safe_mode || status.ai_wallet_status === 'empty' || status.ai_wallet_status === 'low') && (
          <div className="tm-mono" style={{ display: 'flex', gap: 10, alignItems: 'center', margin: '8px 14px 0', padding: '8px 12px', fontSize: 11, border: '1px solid var(--tm-down)', color: 'var(--tm-down)', background: 'rgba(200,60,40,0.06)', flexWrap: 'wrap' }}>
            <span style={{ fontWeight: 600 }}>
              {status.ai_wallet_status === 'empty'
                ? t('aiFeeWalletEmpty', language)
                : status.ai_wallet_status === 'low'
                  ? `${language === 'zh' ? 'AI 费用钱包余额不足' : 'AI fee wallet is low'} (${(status.ai_wallet_balance_usdc ?? 0).toFixed(2)} USDC) — ${language === 'zh' ? '请尽快充值' : 'top up soon'}.`
                  : t('safeMode', language)}
            </span>
            <span style={{ color: 'var(--tm-ink-2)' }}>
              {status.ai_wallet_status === 'empty' || status.ai_wallet_status === 'low'
                ? t('depositHint', language)
                : status.safe_mode_reason || ''}
            </span>
          </div>
        )}
        {/* first-run reassurance — a fresh autopilot looks idle for its first
            minute (the AI is reading the market); tell newcomers what to expect */}
        {!on && status?.is_running && (status.call_count ?? 0) <= 1 && !status.safe_mode && (
          <div className="tm-mono" style={{ display: 'flex', gap: 10, alignItems: 'center', margin: '8px 14px 0', padding: '8px 12px', fontSize: 11, border: '1px solid var(--tm-up)', color: 'var(--tm-ink)', background: 'rgba(40,140,80,0.06)', flexWrap: 'wrap' }}>
            <span style={{ fontWeight: 600, color: 'var(--tm-up)' }}>{t('aiIsLive', language)}</span>
            <span style={{ color: 'var(--tm-ink-2)' }}>
              {t('aiIsLiveDesc', language)}
            </span>
          </div>
        )}
        {/* config / identity strip — first row, flows directly under the global nav */}
        <div className="tm-mono" style={{ display: 'flex', gap: 16, padding: '6px 14px', fontSize: 11, color: 'var(--tm-ink-2)', flexWrap: 'wrap' }}>
          <span style={{ fontWeight: 500 }}>{selectedTrader?.trader_name ?? 'FXOS'}</span>
          <span><span className="tm-sc">{t('model', language)} </span>{(() => {
            const raw = config?.ai_model || status?.ai_model || ''
            if (!raw) return '—'
            if (/claw402/i.test(raw)) return 'CLAW402'
            return raw.length > 16 ? raw.slice(0, 16).toUpperCase() : raw.toUpperCase()
          })()}</span>
          <span><span className="tm-sc">{t('strategy', language)} </span>{config?.strategy_name || selectedTrader?.strategy_name || '—'}</span>
          <span><span className="tm-sc">{t('lev', language)} </span>{config?.btc_eth_leverage ?? '—'}× / {config?.altcoin_leverage ?? '—'}×</span>
          <span><span className="tm-sc">{t('scanInterval', language)} </span>{scanMin}{language === 'zh' ? '分钟' : 'm'}</span>
          <span><span className="tm-sc">{t('universe', language)} </span>{candidateCoins.length}</span>
          <span><span className="tm-sc">{t('positionsLabel', language)} </span>{positions?.length ?? 0}</span>
          <span style={{ marginLeft: 'auto' }}><span className="tm-sc">{t('nextCycle', language)} </span>{countdown === 'due now' ? (language === 'zh' ? '即将开始' : 'due now') : countdown}</span>
        </div>
        <div className="tm-rule" />

        {/* metric row — "Total P/L" is equity-based (includes unrealized);
            "Realized P/L" is closed-trades only and matches PF/win-rate/sharpe,
            so the two never read as contradicting each other */}
        <div className="tm-grid-5" style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)' }}>
          {[
            { l: t('equity', language), v: fmtUsd(account?.total_equity), c: 'var(--tm-ink)' },
            { l: t('totalPnL', language) + ' · ' + (language === 'zh' ? '含未实现' : 'incl. unrealized'), v: `${fmtUsd(pnl, true)} (${fmtPct(pnlPct)})`, c: up ? 'var(--tm-up)' : 'var(--tm-dn)' },
            {
              l: t('totalPnL', language) + ' · ' + (language === 'zh' ? '已平仓' : 'closed trades'),
              v: fullStats != null ? fmtUsd(fullStats.total_pnl, true) : '—',
              c: fullStats != null && fullStats.total_pnl >= 0 ? 'var(--tm-up)' : 'var(--tm-dn)',
            },
            { l: language === 'zh' ? '盈亏比' : 'Profit factor', v: fullStats != null ? fullStats.profit_factor.toFixed(2) : '—', c: 'var(--tm-ink)' },
            { l: language === 'zh' ? '最大回撤' : 'Max drawdown', v: fullStats != null ? `-${fullStats.max_drawdown_pct.toFixed(1)}%` : '—', c: 'var(--tm-dn)' },
          ].map((m, i) => (
            <div key={m.l} style={{ padding: '12px 14px', borderRight: i < 4 ? cellBorder : 'none' }}>
              <div className="tm-sc">{m.l}</div>
              <div className="tm-mono" style={{ fontSize: 17, fontWeight: 500, color: m.c, marginTop: 3 }}>{m.v}</div>
            </div>
          ))}
        </div>
        <div className="tm-rule" />

        {/* trades summary */}
        {fullStats != null && (
          <>
            <div className="tm-mono" style={{ display: 'flex', gap: 18, padding: '6px 14px', fontSize: 11, color: 'var(--tm-ink-2)', flexWrap: 'wrap' }}>
              <span className="tm-sc">{language === 'zh' ? '交易' : 'trades'} <b style={{ color: 'var(--tm-ink)' }}>{fullStats.total_trades}</b></span>
              <span className="tm-sc tm-up">{language === 'zh' ? '盈利' : 'win'} {fullStats.win_trades} ({fullStats.win_rate.toFixed(1)}%)</span>
              <span className="tm-sc tm-dn">{language === 'zh' ? '亏损' : 'loss'} {fullStats.loss_trades}</span>
              {/* fee-drag chain: gross realized − fees = net realized */}
              <span className="tm-sc">{t('gross', language)} <b style={{ color: 'var(--tm-ink)' }}>{fmtUsd(fullStats.total_pnl + fullStats.total_fee, true)}</b></span>
              <span className="tm-sc">{t('feesLabel', language)} <b style={{ color: 'var(--tm-ink)' }}>-{fmtUsd(fullStats.total_fee)}</b></span>
              <span className="tm-sc">{t('net', language)} <b style={{ color: fullStats.total_pnl >= 0 ? 'var(--tm-up)' : 'var(--tm-dn)' }}>{fmtUsd(fullStats.total_pnl, true)}</b></span>
              <span className="tm-sc">{t('sharpeTrade', language)} <b style={{ color: 'var(--tm-ink)' }}>{fullStats.sharpe_ratio.toFixed(2)}</b></span>
              <span className="tm-sc">{t('avgWinLoss', language)} <b style={{ color: 'var(--tm-ink)' }}>{fullStats.avg_win.toFixed(2)}/{fullStats.avg_loss.toFixed(2)}</b></span>
            </div>
            <div className="tm-rule" />
          </>
        )}

        {/* ── row 1: cost/liq map · live L2 order book · signal matrix (instrument selector)
              all three columns are locked to one fixed height so the row is always
              balanced; the K-line flexes to fill any remaining space ── */}
        <div className="tm-grid-3" style={{ display: 'grid', gridTemplateColumns: 'minmax(0,1.1fr) minmax(0,0.95fr) minmax(0,1.05fr)' }}>
          <div className="tm-row1" style={{ ...sc, borderRight: cellBorder, height: ROW1_H, overflow: 'hidden' }}>
            {/* cost/liq heatmap works for both synthetic (hip3_perp) and crypto
                (perp) markets — pass the likely marketType; the component falls
                back to the other one if the guess is wrong */}
            <LiquidationMap
              symbol={activeSym}
              demo={on}
              marketType={CRYPTO_MAJORS.has(activeSym) ? 'perp' : 'hip3_perp'}
              height={ROW1_H - 130}
            />
          </div>
          <div style={{ ...sc, borderRight: cellBorder, height: ROW1_H, overflow: 'hidden' }}>
            <OrderBook symbol={activeSym} demo={on} markPrice={positions?.find((p) => baseLabel(p.symbol) === activeSym)?.entry_price} />
          </div>
          <div style={{ ...sc, height: ROW1_H, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
            <SignalMatrix items={signalRank?.items} max={18} active={activeSym} onSelect={setSelectedSym} />
            {/* the live K-line always sits under the selector and flexes to fill */}
            <div className="tm-rule" style={{ margin: '10px 0 8px' }} />
            <div style={{ flex: 1, minHeight: 0 }}>
              <KlineChart symbol={activeSym} fill demo={on} />
            </div>
          </div>
        </div>
        <div className="tm-rule" />

        {/* orchestration topology — second row, full width (the agent workflow) */}
        <div style={sc}>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 10, marginBottom: 4 }}>
            <span className="tm-px" style={{ fontSize: 12 }}>{t('orchestrationTopology', language)}</span>
            <span className="tm-sc">{t('orchestrationTopologyDesc', language)}</span>
          </div>
          <OrchestrationTopology
            layers={[
              {
                key: 'flow',
                title: t('marketNetInflow', language),
                zh: t('marketNetInflow', language),
                items: [
                  ...(flow?.data?.inflow ?? []).map((i) => ({ symbol: i.symbol, dir: 'long' as const })),
                  ...(flow?.data?.outflow ?? []).map((i) => ({ symbol: i.symbol, dir: 'short' as const })),
                ],
              },
              {
                key: 'signal',
                title: language === 'zh' ? '信号' : 'SIGNAL',
                zh: language === 'zh' ? '信号' : 'SIGNAL',
                items: (signalRank?.items ?? []).map((s) => ({
                  symbol: s.symbol,
                  dir: (s.bias || '').toLowerCase() === 'bearish' ? ('short' as const) : ('long' as const),
                })),
              },
              {
                key: 'decision',
                title: language === 'zh' ? '决策' : 'DECISION',
                zh: language === 'zh' ? '决策' : 'DECISION',
                items: candidateCoins.map((c) => ({ symbol: c, dir: dirFor(c) })),
              },
              {
                key: 'exec',
                title: language === 'zh' ? '执行' : 'EXECUTE',
                zh: language === 'zh' ? '执行' : 'EXECUTE',
                items: (positions ?? []).map((p) => ({
                  symbol: p.symbol,
                  dir: (p.side || '').toLowerCase().includes('short') ? ('short' as const) : ('long' as const),
                })),
              },
              {
                key: 'hold',
                title: language === 'zh' ? '持仓' : 'HOLD',
                zh: language === 'zh' ? '持仓' : 'HOLD',
                items: (positions ?? []).map((p) => ({
                  symbol: p.symbol,
                  dir: (p.side || '').toLowerCase().includes('short') ? ('short' as const) : ('long' as const),
                })),
              },
            ]}
          />
        </div>
        <div className="tm-rule" />

        {/* ── row 3: execution log · risk radar · recent trades ── */}
        <div className="tm-grid-3" style={{ display: 'grid', gridTemplateColumns: 'minmax(0,1.1fr) minmax(0,1fr) minmax(0,1fr)' }}>
          <div style={{ ...sc, borderRight: cellBorder }}>
            <ExecutionLog decisions={decisions} height={432} onCycleClick={setDetailDecision} />
          </div>
          <div style={{ ...sc, borderRight: cellBorder }}>
            <RiskRadar positions={positions} account={account} config={config} fullStats={fullStats} />
          </div>
          <div style={sc}>
            {/* live open positions (the book right now) */}
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 6 }}>
              <span className="tm-px" style={{ fontSize: 11 }}>{t('positions', language)}</span>
              <span className="tm-sc">{t('currentPositions', language)} · live</span>
              <span className="tm-sc" style={{ marginLeft: 'auto' }}>{positions?.length ?? 0} {language === 'zh' ? '个持仓' : 'open'}</span>
            </div>
            {positions && positions.length > 0 ? (
              <div className="tm-scroll-x">
              <table className="tm-mono" style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11, minWidth: 420 }}>
                <thead>
                  <tr className="tm-sc" style={{ fontSize: 9 }}>
                    <td style={{ padding: '0 0 3px' }}>{language === 'zh' ? '币种' : 'symbol'}</td>
                    <td style={{ padding: '0 0 3px' }}>{language === 'zh' ? '方向·杠杆' : 'side·lev'}</td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>{language === 'zh' ? '入场价' : 'entry'}</td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>{language === 'zh' ? '数量' : 'size'}</td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>PnL</td>
                    <td style={{ padding: '0 0 3px', textAlign: 'right' }}>{language === 'zh' ? '收益率%' : 'return%'}</td>
                  </tr>
                </thead>
                <tbody>
                  {positions.map((p, i) => {
                    const long = /long|buy/i.test(p.side)
                    const win = (p.unrealized_pnl ?? 0) >= 0
                    const notional = Math.abs(p.quantity ?? 0) * (p.mark_price || p.entry_price || 0)
                    return (
                      <tr key={`${p.symbol}-${i}`} style={{ borderTop: '1px solid var(--tm-hair)' }}>
                        <td style={{ padding: '5px 0', fontWeight: 500 }}>{baseLabel(p.symbol)}</td>
                        <td style={{ padding: '5px 0' }} className={long ? 'tm-up' : 'tm-dn'}>{long ? 'long' : 'short'} <span style={{ color: 'var(--tm-muted)' }}>{p.leverage}×</span></td>
                        <td style={{ padding: '5px 0', textAlign: 'right', color: 'var(--tm-ink-2)' }}>{fmtPx(p.entry_price)}</td>
                        <td style={{ padding: '5px 0', textAlign: 'right', color: 'var(--tm-ink-2)' }}>{fmtUsd(notional)}</td>
                        <td style={{ padding: '5px 0', textAlign: 'right' }} className={win ? 'tm-up' : 'tm-dn'}>{fmtUsd(p.unrealized_pnl, true)}</td>
                        <td style={{ padding: '5px 0', textAlign: 'right' }} className={win ? 'tm-up' : 'tm-dn'}>{(p.unrealized_pnl_pct ?? 0).toFixed(2)}%</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
              </div>
            ) : <div className="tm-sc" style={{ padding: '8px 0' }}>No open positions.</div>}

            <div className="tm-rule" style={{ margin: '12px 0 10px' }} />

            <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 6 }}>
              <span className="tm-px" style={{ fontSize: 11 }}>Recent trades</span>
              <span className="tm-sc">Recent closes · symbol/side/time/pnl</span>
            </div>
            {recentTrades.length > 0 ? (
              <div className="tm-scroll-x">
              <table className="tm-mono" style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11, minWidth: 320 }}>
                <tbody>
                  {recentTrades.map((p) => {
                    const win = p.realized_pnl >= 0
                    return (
                      <tr key={p.id} style={{ borderTop: '1px solid var(--tm-hair)' }}>
                        <td style={{ padding: '5px 0', fontWeight: 500 }}>{baseLabel(p.symbol)}</td>
                        <td style={{ padding: '5px 0' }} className={p.side === 'long' || p.side === 'LONG' ? 'tm-up' : 'tm-dn'}>{p.side.toLowerCase()}</td>
                        <td style={{ padding: '5px 0', color: 'var(--tm-muted)' }}>{fmtTime(p.exit_time)}</td>
                        <td style={{ padding: '5px 0', textAlign: 'right' }} className={win ? 'tm-up' : 'tm-dn'}>{fmtUsd(p.realized_pnl, true)}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
              </div>
            ) : <div className="tm-sc" style={{ padding: '8px 0' }}>No closed trades yet.</div>}
          </div>
        </div>
        <div className="tm-rule" />

        {/* market net inflow (Vergex) · by-symbol history · edge profile — footer */}
        <div className="tm-grid-3" style={{ display: 'grid', gridTemplateColumns: 'minmax(0,1.2fr) minmax(0,0.9fr) minmax(0,0.9fr)' }}>
          <div style={{ ...sc, borderRight: cellBorder }}>
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 10, marginBottom: 8 }}>
              <span className="tm-px" style={{ fontSize: 12 }}>{t('marketNetInflow', language)}</span>
              <span className="tm-sc">{t('marketNetInflow', language)} · {flow?.data?.window || '1h'} · Vergex</span>
              <span className="tm-sc" style={{ marginLeft: 'auto' }}>{flowItems.length} {language === 'zh' ? '个市场' : 'markets'}</span>
            </div>
            <FlowMarkets items={flowItems} window={flow?.data?.window} />
          </div>
          <div style={{ ...sc, borderRight: cellBorder }}>
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 8 }}>
              <span className="tm-px" style={{ fontSize: 11 }}>{t('bySymbol', language)}</span>
              <span className="tm-sc">{t('bySymbolHistory', language)} · {t('tradesWinPnl', language)}</span>
            </div>
            {symbolStats.length > 0 ? symbolStats.map((s) => (
              <div key={s.symbol} style={{ marginBottom: 7 }}>
                <div className="tm-mono" style={{ display: 'flex', fontSize: 11, marginBottom: 2 }}>
                  <span style={{ fontWeight: 500 }}>{baseLabel(s.symbol)}</span>
                  <span className="tm-sc" style={{ marginLeft: 8 }}>{s.total_trades} {t('trades', language)} · {s.win_rate.toFixed(0)}% {t('win', language)}</span>
                  <span className={s.total_pnl >= 0 ? 'tm-up' : 'tm-dn'} style={{ marginLeft: 'auto' }}>{fmtUsd(s.total_pnl, true)}</span>
                </div>
                <div style={{ height: 4, background: 'var(--tm-hair)' }}>
                  <div style={{ height: 4, width: `${(s.total_trades / maxSymTrades) * 100}%`, background: s.total_pnl >= 0 ? 'var(--tm-up)' : 'var(--tm-dn)' }} />
                </div>
              </div>
            )) : <div className="tm-sc">No symbol history.</div>}
          </div>
          <div style={sc}>
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 8, marginBottom: 8 }}>
              <span className="tm-px" style={{ fontSize: 11 }}>{t('edgeProfile', language)}</span>
              <span className="tm-sc">Net by hold time &amp; side · after fees</span>
            </div>
            <EdgeProfile positions={history?.positions} />
          </div>
        </div>
      </div>

      {/* Decision Detail Modal */}
      {detailDecision && (
        <DecisionDetailModal
          decision={detailDecision}
          language={language}
          onClose={() => setDetailDecision(null)}
        />
      )}
    </div>
  )
}

export default TerminalDashboard
