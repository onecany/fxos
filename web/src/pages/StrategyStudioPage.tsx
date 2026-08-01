import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  ArrowDownRight,
  ArrowUpRight,
  Bot,
  Check,
  Loader2,
  Plus,
  RefreshCw,
  Save,
  Shield,
  Sparkles,
  Target,
  Trash2,
} from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import { DeepVoidBackground } from '../components/common/DeepVoidBackground'
import { api } from '../lib/api'
import { confirmToast, notify } from '../lib/notify'
import type {
  AIStrategyConfig,
  CoinSourceConfig,
  IndicatorConfig,
  RiskControlConfig,
  Strategy,
  StrategyConfig,
} from '../types'
import { launchAutopilot } from '../lib/launch/launchAutopilot'
import type {
  MarketSymbol,
  VergexHeatmapBin,
  VergexHeatmapResponse,
  VergexSignalDimension,
  VergexSignalItem,
  VergexSignalLabResponse,
} from '../lib/api/data'
import { buildDashboardPath, ROUTES } from '../router/paths'
import { apiUrl } from '../lib/api/helpers'
import { t, type Language } from '../i18n/translations'

type Scope =
  | 'all'
  | 'crypto'
  | 'stock'
  | 'commodity'
  | 'index'
  | 'forex'
  | 'pre_ipo'
type ListMode = 'claw402' | 'pool'

const scopeOptions: Array<{ value: Scope; zh: string; en: string }> = [
  { value: 'all', zh: 'All Claw402', en: 'All Claw402' },
  { value: 'stock', zh: 'US Stocks', en: 'US Stocks' },
  { value: 'crypto', zh: 'Crypto', en: 'Crypto' },
  { value: 'commodity', zh: 'Commodities', en: 'Commodities' },
  { value: 'index', zh: 'Indices', en: 'Indices' },
  { value: 'forex', zh: 'FX', en: 'FX' },
  { value: 'pre_ipo', zh: 'Pre-IPO', en: 'Pre-IPO' },
]

const categoryPriority: Record<string, number> = {
  stock: 1,
  crypto: 2,
  commodity: 3,
  index: 4,
  forex: 5,
  pre_ipo: 6,
}

const timeframeOptions = ['5m', '15m', '30m', '1h', '4h', '1d']
const barCountOptions = [20, 30, 50]
const topNOptions = [5, 6, 7, 8, 9, 10]
const detailBandOptions = ['5', '10', '15', '20']
const claw402BoardLimit = 30
const confidenceOptions = [65, 75, 82]

const text = (language: string, zh: string, en: string, id?: string) =>
  language === 'zh' ? zh : language === 'id' && id ? id : en

type Profile = 'careful' | 'balanced' | 'active'

const profileOptions: Array<{
  value: Profile
  zh: string
  en: string
  zhNote: string
  enNote: string
  id: string
  maxPositions: number
  leverage: number
  confidence: number
  topN: number
  timeframe: string
  bars: number
  margin: number
  promptZh: string
  promptEn: string
}> = [
  {
    value: 'careful',
    zh: '稳健',
    en: 'Careful',
    id: 'Konservatif',
    zhNote: '交易较少，仅对齐信号',
    enNote: 'Fewer trades, only aligned signals',
    maxPositions: 1,
    leverage: 10,
    confidence: 82,
    topN: 5,
    timeframe: '1h',
    bars: 30,
    margin: 1.0,
    promptZh:
      'Careful mode: open only when the Claw402 board direction, Signal Lab, cost/liquidation heatmap and raw candles agree; wait on conflicts.',
    promptEn:
      'Careful mode: open only when the Claw402 board direction, Signal Lab, cost/liquidation heatmap and raw candles agree; wait on conflicts.',
  },
  {
    value: 'balanced',
    zh: '均衡',
    en: 'Balanced',
    id: 'Seimbang',
    zhNote: '机会与风险的推荐平衡',
    enNote: 'Recommended balance of opportunity and risk',
    maxPositions: 2,
    leverage: 10,
    confidence: 75,
    topN: 5,
    timeframe: '15m',
    bars: 30,
    margin: 1.0,
    promptZh:
      'Balanced mode: prioritize top Claw402-ranked symbols when Signal Lab agrees with raw candles; use the liquidation heatmap for stop and target zones.',
    promptEn:
      'Balanced mode: prioritize top Claw402-ranked symbols when Signal Lab agrees with raw candles; use the liquidation heatmap for stop and target zones.',
  },
  {
    value: 'active',
    zh: '积极',
    en: 'Active',
    id: 'Aktif',
    zhNote: '更快捕捉趋势，持仓更多',
    enNote: 'Faster trend capture with more positions',
    maxPositions: 3,
    leverage: 10,
    confidence: 68,
    topN: 8,
    timeframe: '5m',
    bars: 50,
    margin: 1.0,
    promptZh:
      'Active mode: follow strong Claw402 signals faster, but require Signal Lab confirmation, avoid crowded liquidation zones, and always set explicit stops.',
    promptEn:
      'Active mode: follow strong Claw402 signals faster, but require Signal Lab confirmation, avoid crowded liquidation zones, and always set explicit stops.',
  },
]

function getAIConfig(config: StrategyConfig): AIStrategyConfig | null {
  if (config.ai_config) return config.ai_config
  if (config.coin_source && config.indicators && config.risk_control) {
    return {
      coin_source: config.coin_source,
      indicators: config.indicators,
      risk_control: config.risk_control,
      prompt_sections: config.prompt_sections,
      custom_prompt: config.custom_prompt,
    }
  }
  return null
}

function defaultCoinSource(
  source?: Partial<CoinSourceConfig>
): CoinSourceConfig {
  const staticCoins = source?.static_coins || []
  const minVergexLimit =
    staticCoins.length > 0 ? Math.min(staticCoins.length, 10) : 10
  const vergexLimit = Math.min(
    Math.max(source?.vergex_limit || minVergexLimit, minVergexLimit),
    10
  )
  return {
    source_type: staticCoins.length > 0 ? 'static' : 'vergex_signal',
    static_coins: staticCoins,
    excluded_coins: [],
    use_ai500: false,
    ai500_limit: 0,
    use_oi_top: false,
    oi_top_limit: 0,
    use_oi_low: false,
    oi_low_limit: 0,
    use_hyper_all: false,
    use_hyper_main: false,
    hyper_main_limit: 0,
    hyper_rank_category: source?.hyper_rank_category || 'all',
    hyper_rank_direction: 'gainers',
    hyper_rank_limit: 0,
    vergex_limit: vergexLimit,
    vergex_market_type: source?.vergex_market_type || 'all',
    vergex_chain: source?.vergex_chain || 'hyperliquid',
    vergex_liq_band: source?.vergex_liq_band || '',
  }
}

function defaultIndicators(
  indicators?: Partial<IndicatorConfig>
): IndicatorConfig {
  const klines = indicators?.klines || {
    primary_timeframe: '15m',
    primary_count: 30,
    enable_multi_timeframe: false,
  }

  return {
    klines: {
      primary_timeframe: klines.primary_timeframe || '15m',
      primary_count: klines.primary_count || 30,
      longer_timeframe: '',
      longer_count: 0,
      enable_multi_timeframe: false,
      selected_timeframes: [klines.primary_timeframe || '15m'],
    },
    enable_raw_klines: true,
    enable_ema: false,
    enable_macd: false,
    enable_rsi: false,
    enable_atr: false,
    enable_boll: false,
    enable_volume: false,
    enable_oi: false,
    enable_funding_rate: false,
    fxosos_api_key: '',
    enable_quant_data: false,
    enable_quant_oi: false,
    enable_quant_netflow: false,
    enable_oi_ranking: false,
    enable_netflow_ranking: false,
    enable_price_ranking: false,
  }
}

function defaultRisk(risk?: Partial<RiskControlConfig>): RiskControlConfig {
  const leverage =
    risk?.altcoin_max_leverage || risk?.btc_eth_max_leverage || 10
  return {
    max_positions: risk?.max_positions || 2,
    btc_eth_max_leverage: leverage,
    altcoin_max_leverage: leverage,
    btc_eth_max_position_value_ratio:
      risk?.btc_eth_max_position_value_ratio || 10,
    altcoin_max_position_value_ratio:
      risk?.altcoin_max_position_value_ratio || 10,
    max_margin_usage: risk?.max_margin_usage || 1.0,
    min_position_size: risk?.min_position_size || 12,
    min_risk_reward_ratio: risk?.min_risk_reward_ratio || 3,
    min_confidence: risk?.min_confidence || 78,
  }
}

function simplifyConfig(
  config: StrategyConfig | null | undefined
): StrategyConfig {
  const ai = config ? getAIConfig(config) : null
  return {
    strategy_type: 'ai_trading',
    language: config?.language || 'zh',
    ai_config: {
      coin_source: defaultCoinSource(ai?.coin_source),
      indicators: defaultIndicators(ai?.indicators),
      risk_control: defaultRisk(ai?.risk_control),
      custom_prompt: ai?.custom_prompt || '',
      prompt_sections: ai?.prompt_sections,
    },
    grid_config: null,
    publish_config: config?.publish_config,
  }
}

function normalizeSymbol(symbol: string) {
  return symbol
    .trim()
    .toUpperCase()
    .replace(/^XYZ:/, '')
    .replace(/-USDC$/, '')
}

function signalMarketType(item: VergexSignalItem) {
  return (
    item.market_type || (item.category === 'crypto' ? 'core_perp' : 'hip3_perp')
  )
}

function strategySymbolForSignal(item: VergexSignalItem) {
  const symbol = normalizeSymbol(item.symbol)
  return signalMarketType(item) === 'core_perp' ? symbol : `xyz:${symbol}`
}

function categoryLabel(category: string | undefined, language: string) {
  const option = scopeOptions.find((item) => item.value === category)
  if (!option) return category || 'TradeFi'
  return text(language, option.zh, option.en)
}

function profileFromRisk(risk: RiskControlConfig | null | undefined): Profile {
  if (!risk) return 'balanced'
  if (risk.min_confidence >= 80 || risk.max_positions <= 1) return 'careful'
  if (risk.altcoin_max_leverage >= 5 || risk.max_positions >= 3) return 'active'
  return 'balanced'
}

function formatChange(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) return ''
  const sign = value > 0 ? '+' : ''
  return `${sign}${value.toFixed(2)}%`
}

function signalBiasInfo(bias: string | undefined, lang?: string) {
  const normalized = (bias || '').toLowerCase()
  const bullish = ['bullish', 'long', 'buy', 'open_long'].includes(normalized)
  const bearish = ['bearish', 'short', 'sell', 'open_short'].includes(
    normalized
  )
  const isZh = lang === 'zh'
  if (bullish) {
    return {
      label: isZh ? '偏多' : 'Long Bias',
      hint: isZh ? '偏多' : 'Long bias',
      classes: 'border-fxos-success/35 bg-fxos-success/10 text-fxos-success',
      icon: ArrowUpRight,
    }
  }
  if (bearish) {
    return {
      label: isZh ? '偏空' : 'Short Bias',
      hint: isZh ? '偏空' : 'Short bias',
      classes: 'border-fxos-danger/35 bg-fxos-danger/10 text-fxos-danger',
      icon: ArrowDownRight,
    }
  }
  return {
    label: isZh ? '中性' : 'Neutral',
    hint: isZh ? '中性' : 'Neutral bias',
    classes:
      'border-[var(--panel-border)] bg-fxos-bg-deeper text-fxos-text-muted',
    icon: Target,
  }
}

function formatSignalStrength(item: VergexSignalItem) {
  const parts: string[] = []
  if (typeof item.score === 'number' && Number.isFinite(item.score)) {
    const sign = item.score > 0 ? '+' : ''
    parts.push(`z ${sign}${item.score.toFixed(2)}`)
  }
  if (typeof item.confidence === 'number' && Number.isFinite(item.confidence)) {
    const confidence =
      item.confidence <= 1 ? item.confidence * 100 : item.confidence
    if (confidence > 0) {
      parts.push(`${confidence.toFixed(0)}% conf`)
    }
  }
  return parts.join(' · ') || 'details ready'
}

function signalSortValue(item: VergexSignalItem) {
  return categoryPriority[item.category || ''] || 99
}

function compareSignalItems(a: VergexSignalItem, b: VergexSignalItem) {
  const categoryDelta = signalSortValue(a) - signalSortValue(b)
  if (categoryDelta !== 0) return categoryDelta
  return (
    (a.rank || Number.MAX_SAFE_INTEGER) - (b.rank || Number.MAX_SAFE_INTEGER)
  )
}

function sameSignalItem(a: VergexSignalItem | null, b: VergexSignalItem) {
  if (!a) return false
  return (
    normalizeSymbol(a.symbol) === normalizeSymbol(b.symbol) &&
    signalMarketType(a) === signalMarketType(b)
  )
}

function clampPct(value: number) {
  if (!Number.isFinite(value)) return 0
  return Math.min(100, Math.max(0, value))
}

function formatMoney(value: number | undefined) {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  const sign = value < 0 ? '-' : ''
  const abs = Math.abs(value)
  if (abs >= 1_000_000_000)
    return `${sign}$${(abs / 1_000_000_000).toFixed(2)}B`
  if (abs >= 1_000_000) return `${sign}$${(abs / 1_000_000).toFixed(2)}M`
  if (abs >= 1_000) return `${sign}$${(abs / 1_000).toFixed(2)}K`
  return `${sign}$${abs.toFixed(2)}`
}

function formatNumber(value: number | undefined, digits = 2) {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  return value.toFixed(digits)
}

function formatPrice(value: number | undefined) {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  return `$${value.toFixed(value >= 100 ? 2 : 4)}`
}

function formatSignedPct(value: number | undefined) {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  const sign = value > 0 ? '+' : ''
  return `${sign}${value.toFixed(1)}%`
}

function metricPct(value: number | undefined) {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  const pct = Math.abs(value) <= 1 ? value * 100 : value
  return `${pct.toFixed(1)}%`
}

function directionStyle(direction?: string) {
  const normalized = (direction || '').toLowerCase()
  if (normalized === 'bullish') {
    return {
      dot: 'bg-fxos-success',
      text: 'text-fxos-success',
      chip: 'border-fxos-success/25 bg-fxos-success/10 text-fxos-success',
      bar: 'bg-fxos-success/70',
    }
  }
  if (normalized === 'bearish') {
    return {
      dot: 'bg-fxos-danger',
      text: 'text-fxos-danger',
      chip: 'border-fxos-danger/25 bg-fxos-danger/10 text-fxos-danger',
      bar: 'bg-fxos-danger/70',
    }
  }
  return {
    dot: 'bg-slate-400',
    text: 'text-fxos-text-muted',
    chip: 'border-[var(--panel-border)] bg-fxos-bg-deeper text-fxos-text-muted',
    bar: 'bg-slate-500/70',
  }
}

function SignalDimensionRow({ item }: { item: VergexSignalDimension }) {
  const style = directionStyle(item.direction)
  const percentile =
    typeof item.percentile === 'number' && Number.isFinite(item.percentile)
      ? clampPct(item.percentile)
      : null
  const chipLabel = [item.direction, item.strength].filter(Boolean).join(' · ')

  return (
    <div className="border-t border-[var(--panel-border)] px-3 py-3">
      <div className="grid gap-3 md:grid-cols-[1fr_150px]">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className={`h-2 w-2 rounded-full ${style.dot}`} />
            <span className="text-sm font-semibold text-fxos-text">
              {item.label || item.key || 'Signal factor'}
            </span>
            {chipLabel ? (
              <span
                className={`rounded-md border px-2 py-0.5 text-[11px] ${style.chip}`}
              >
                {chipLabel}
              </span>
            ) : null}
          </div>
          <div className="mt-1 text-xs leading-5 text-fxos-text-muted">
            {item.detail || item.what || 'No detail returned.'}
          </div>
        </div>
        {percentile !== null ? (
          <div className="flex items-center gap-2">
            <div className="h-2 flex-1 overflow-hidden rounded-full bg-fxos-bg-deeper">
              <div
                className={`h-full rounded-full ${style.bar}`}
                style={{ width: `${percentile}%` }}
              />
            </div>
            <span className="w-10 text-right font-mono text-xs text-fxos-text-muted">
              {percentile.toFixed(0)}
            </span>
          </div>
        ) : null}
      </div>
    </div>
  )
}

function DetailMetricCard({
  label,
  value,
  note,
  tone = 'neutral',
}: {
  label: string
  value: string
  note?: string
  tone?: 'neutral' | 'green' | 'red' | 'cyan' | 'gold'
}) {
  const toneClass =
    tone === 'green'
      ? 'text-fxos-success'
      : tone === 'red'
        ? 'text-fxos-danger'
        : tone === 'cyan'
          ? 'text-fxos-gold'
          : tone === 'gold'
            ? 'text-fxos-gold'
            : 'text-fxos-text'

  return (
    <div className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper p-3">
      <div className="text-xs text-fxos-text-muted">{label}</div>
      <div className={`mt-2 font-mono text-lg ${toneClass}`}>{value}</div>
      {note ? (
        <div className="mt-2 text-xs leading-5 text-fxos-text-muted">
          {note}
        </div>
      ) : null}
    </div>
  )
}

function BandSelector({
  activeBand,
  loading,
  onBandChange,
}: {
  activeBand: string
  loading?: boolean
  onBandChange?: (band: string) => void
}) {
  return (
    <div className="flex flex-wrap gap-2">
      {detailBandOptions.map((band) => (
        <button
          key={band}
          type="button"
          onClick={() => onBandChange?.(band)}
          disabled={loading}
          className={`rounded-lg border px-4 py-2 font-mono text-sm transition disabled:cursor-not-allowed disabled:opacity-50 ${
            activeBand === band
              ? 'border-fxos-gold/60 bg-fxos-gold/12 text-fxos-gold'
              : 'border-[var(--panel-border)] bg-fxos-bg-deeper text-fxos-text-muted hover:border-[var(--panel-border)] hover:text-fxos-text'
          }`}
        >
          ±{band}%
        </button>
      ))}
    </div>
  )
}

function SignalLabPanel({
  lab,
  activeBand,
  loading,
  onBandChange,
  language,
}: {
  lab: VergexSignalLabResponse | null
  activeBand: string
  loading?: boolean
  onBandChange?: (band: string) => void
  language: Language
}) {
  const data = lab?.data
  if (!data) {
    return (
      <div className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper p-4 text-sm text-fxos-text-muted">
        Signal Lab has not loaded yet.
      </div>
    )
  }

  const bias = signalBiasInfo(data.bias, language)
  const BiasIcon = bias.icon
  const levels = data.levels || {}
  const metrics = data.metrics || {}
  const dimensions = data.dimensions || []

  return (
    <section className="overflow-hidden rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter shadow-lg">
      <div className="border-b border-[var(--panel-border)] bg-fxos-bg-lighter px-5 py-4">
        <div className="grid gap-4 lg:grid-cols-[1fr_auto] lg:items-start">
          <div>
            <div className="text-base font-semibold text-fxos-text">
              Signal Lab
              <span className="ml-3 text-sm font-normal text-fxos-text-muted">
                full-book cost basis + node liquidation map · facts only
              </span>
            </div>
            <div className="mt-4">
              <BandSelector
                activeBand={activeBand}
                loading={loading}
                onBandChange={onBandChange}
              />
            </div>
          </div>
          <div className="flex flex-col items-start gap-2 lg:items-end">
            <div className="rounded-full bg-fxos-success/10 px-3 py-1 text-xs font-semibold text-fxos-success">
              live
            </div>
            {data.confidence ? (
              <div className="font-mono text-sm text-fxos-text-muted">
                confidence {data.confidence}
              </div>
            ) : null}
          </div>
        </div>
      </div>

      <div className="px-5 py-5">
        <div className="flex flex-wrap items-end gap-4">
          <div
            className={`text-4xl font-bold ${directionStyle(data.bias).text}`}
          >
            {bias.label}
          </div>
          {data.rank ? (
            <div className="pb-1 font-mono text-base text-fxos-success">
              market #{data.rank}/{data.universeSize || 30}
            </div>
          ) : null}
          {typeof data.compositeZ === 'number' ? (
            <div className="pb-1 font-mono text-base text-fxos-success">
              z {data.compositeZ >= 0 ? '+' : ''}
              {data.compositeZ.toFixed(2)}
            </div>
          ) : null}
          <BiasIcon
            className={`mb-1 h-6 w-6 ${directionStyle(data.bias).text}`}
          />
        </div>
      </div>

      {data.structureRead ? (
        <div className="mx-5 rounded-md border-l-4 border-fxos-gold/35 bg-fxos-bg-deeper px-4 py-4 text-base leading-8 text-fxos-text">
          {data.structureRead}
        </div>
      ) : null}

      {dimensions.length > 0 ? (
        <div className="pt-5">
          <div className="px-5 pb-2 text-sm font-semibold text-fxos-text">
            Factors
            <span className="ml-2 text-sm font-normal text-fxos-text-muted">
              bar = cross-market percentile
            </span>
          </div>
          {dimensions.map((dimension, index) => (
            <SignalDimensionRow
              key={`${dimension.key || dimension.label || 'factor'}-${index}`}
              item={dimension}
            />
          ))}
        </div>
      ) : null}

      <div className="border-t border-[var(--panel-border)] p-5">
        <div className="text-base font-semibold text-fxos-text">
          Key levels price {formatPrice(levels.markPrice)}
        </div>
        <div className="mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <DetailMetricCard
            label="Fair-value magnet (POC)"
            value={`${formatPrice(levels.poc)} (${formatSignedPct(levels.pocDistPct)})`}
            note="Where the most cost is concentrated."
            tone="neutral"
          />
          <DetailMetricCard
            label="Strongest liq cluster"
            value={`${formatPrice(levels.magnet)} (${formatSignedPct(levels.magnetDistPct)})`}
            note="Largest forced-close cluster in the selected band."
            tone="cyan"
          />
          <DetailMetricCard
            label="Resistance above"
            value={`${formatPrice(levels.resistance)} (${formatSignedPct(levels.resistanceDistPct)})`}
            note="Trapped longs may sell to break even as price returns."
            tone="red"
          />
          <DetailMetricCard
            label="Support below"
            value={`${formatPrice(levels.support)} (${formatSignedPct(levels.supportDistPct)})`}
            note="Trapped shorts may cover as price falls back."
            tone="green"
          />
        </div>

        <div className="mt-6 text-base font-semibold text-fxos-text">
          Structure metrics
        </div>
        <div className="mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <DetailMetricCard
            label="Squeeze fuel above"
            value={formatMoney(metrics.shortLiqAbove)}
            note="Short liquidation fuel above current price."
            tone="green"
          />
          <DetailMetricCard
            label="Flush fuel below"
            value={formatMoney(metrics.longLiqBelow)}
            note="Long liquidation fuel below current price."
            tone="red"
          />
          <DetailMetricCard
            label="Cascade vulnerability"
            value={metricPct(metrics.cascadeVulnPct)}
            note="Share of OI close to force-close."
            tone="neutral"
          />
          <DetailMetricCard
            label="Long book PnL"
            value={formatMoney(metrics.longOverhangPnl)}
            note={`avg ${metricPct(metrics.gLong)}`}
            tone="green"
          />
          <DetailMetricCard
            label="Short book PnL"
            value={formatMoney(metrics.shortOverhangPnl)}
            note={`avg ${metricPct(metrics.gShort)}`}
            tone="red"
          />
          <DetailMetricCard
            label="Top-10 concentration"
            value={metricPct(metrics.top10Pct)}
            note="Share held by the top 10 addresses."
            tone="neutral"
          />
        </div>
      </div>
    </section>
  )
}

function binPrice(bin: VergexHeatmapBin) {
  if (typeof bin.px === 'number') return bin.px
  if (
    typeof bin.bucketStartPrice === 'number' &&
    typeof bin.bucketEndPrice === 'number'
  ) {
    return (bin.bucketStartPrice + bin.bucketEndPrice) / 2
  }
  return 0
}

function binValue(bin: VergexHeatmapBin) {
  return (
    Math.abs(bin.longCost || 0) +
    Math.abs(bin.shortCost || 0) +
    Math.abs(bin.longLiq || 0) +
    Math.abs(bin.shortLiq || 0)
  )
}

function sideBarWidth(value: number | undefined, maxValue: number) {
  if (!value || !Number.isFinite(value) || maxValue <= 0) return '0%'
  return `${Math.max(1.5, Math.min(46, (Math.abs(value) / maxValue) * 46))}%`
}

function ChartGridLines() {
  return (
    <div className="pointer-events-none absolute inset-y-0 left-[92px] right-0">
      {[12.5, 25, 37.5, 50, 62.5, 75, 87.5].map((left) => (
        <div
          key={left}
          className={`absolute top-0 h-full w-px ${
            left === 50 ? 'bg-[var(--panel-border)]' : 'bg-fxos-bg-deeper'
          }`}
          style={{ left: `${left}%` }}
        />
      ))}
    </div>
  )
}

function HeatmapChartRow({
  bin,
  maxLeft,
  maxRight,
  markPrice,
}: {
  bin: VergexHeatmapBin
  maxLeft: number
  maxRight: number
  markPrice?: number
}) {
  const price = binPrice(bin)
  const isCurrent =
    typeof markPrice === 'number' &&
    typeof bin.bucketStartPrice === 'number' &&
    typeof bin.bucketEndPrice === 'number' &&
    markPrice >= bin.bucketStartPrice &&
    markPrice <= bin.bucketEndPrice

  return (
    <div
      className="relative z-10 grid grid-cols-[78px_minmax(0,1fr)] items-center gap-3"
      title={[
        `Price ${formatPrice(price)}`,
        `Long cost ${formatMoney(bin.longCost)}`,
        `Short cost ${formatMoney(bin.shortCost)}`,
        `Long liquidation ${formatMoney(bin.longLiq)}`,
        `Short liquidation ${formatMoney(bin.shortLiq)}`,
      ].join(' · ')}
    >
      <div
        className={`text-right font-mono text-xs ${
          isCurrent ? 'text-fxos-gold' : 'text-fxos-text-muted'
        }`}
      >
        {formatPrice(price)}
      </div>
      <div
        className={`relative h-6 overflow-visible rounded-sm ${
          isCurrent ? 'bg-fxos-gold/10' : 'bg-fxos-bg-lighter'
        }`}
      >
        {isCurrent ? (
          <>
            <div className="absolute inset-x-0 top-1/2 h-px bg-fxos-gold" />
            <div className="absolute right-2 top-1/2 -translate-y-1/2 rounded-md bg-fxos-gold px-2 py-1 font-mono text-xs font-bold text-fxos-bg">
              Mark {formatPrice(markPrice)}
            </div>
          </>
        ) : null}
        <div
          className="absolute right-1/2 top-[5px] h-2 rounded-l bg-fxos-danger/80"
          style={{ width: sideBarWidth(bin.shortCost, maxLeft) }}
        />
        <div
          className="absolute left-1/2 top-[5px] h-2 rounded-r bg-fxos-success/80"
          style={{ width: sideBarWidth(bin.longCost, maxRight) }}
        />
        <div
          className="absolute right-1/2 bottom-[5px] h-2 rounded-l bg-orange-400"
          style={{ width: sideBarWidth(bin.longLiq, maxLeft) }}
        />
        <div
          className="absolute left-1/2 bottom-[5px] h-2 rounded-r bg-fxos-gold"
          style={{ width: sideBarWidth(bin.shortLiq, maxRight) }}
        />
      </div>
    </div>
  )
}

function CostLiquidationHeatmap({
  heatmap,
}: {
  heatmap: VergexHeatmapResponse | null
}) {
  const data = heatmap?.data
  const bins = (data?.bins || [])
    .filter((bin) => binValue(bin) > 0)
    .slice()
    .sort((a, b) => binPrice(b) - binPrice(a))

  if (!data || bins.length === 0) {
    return (
      <div className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper p-4 text-sm text-fxos-text-muted">
        Cost/liquidation heatmap has not loaded yet.
      </div>
    )
  }

  const maxLeft = Math.max(
    ...bins.map((bin) =>
      Math.max(Math.abs(bin.shortCost || 0), Math.abs(bin.longLiq || 0))
    ),
    1
  )
  const maxRight = Math.max(
    ...bins.map((bin) =>
      Math.max(Math.abs(bin.longCost || 0), Math.abs(bin.shortLiq || 0))
    ),
    1
  )
  const longLiqTotal = bins.reduce((sum, bin) => sum + (bin.longLiq || 0), 0)
  const shortLiqTotal = bins.reduce((sum, bin) => sum + (bin.shortLiq || 0), 0)
  const includedCost = data.cost?.includedPositions || data.costAddrs || 0
  const includedLiq = data.liqAddrs || 0

  return (
    <section className="overflow-hidden rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter shadow-lg">
      <div className="border-b border-[var(--panel-border)] bg-fxos-bg-lighter px-5 py-4">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <div className="text-base font-semibold text-fxos-text">
              Cost / Liquidation Heatmap
              <span className="ml-3 text-sm font-normal text-fxos-text-muted">
                position cost distribution · liquidation clusters
              </span>
            </div>
            <div className="mt-3 flex flex-wrap gap-3 text-sm text-fxos-text-muted">
              <span>{includedCost.toLocaleString()} cost positions</span>
              <span>{includedLiq.toLocaleString()} liquidation prices</span>
              <span>
                mark{' '}
                <span className="font-semibold text-fxos-text">
                  {formatPrice(data.markPrice)}
                </span>
              </span>
            </div>
            {data.liquidation?.reason ? (
              <div className="mt-2 text-sm text-fxos-gold">
                Liquidation prices use latest snapshot; incremental trades can
                lag.
              </div>
            ) : null}
          </div>
          <div className="rounded-full bg-fxos-success/10 px-3 py-1 text-xs font-semibold text-fxos-success">
            live
          </div>
        </div>
      </div>

      <div className="px-5 py-5">
        <div className="mb-4 flex flex-wrap justify-center gap-4 text-sm text-fxos-text-muted">
          <span className="inline-flex items-center gap-1">
            <span className="h-3 w-3 rounded bg-fxos-success/70" />
            Long cost
          </span>
          <span className="inline-flex items-center gap-1">
            <span className="h-3 w-3 rounded bg-fxos-danger/70" />
            Short cost
          </span>
          <span className="inline-flex items-center gap-1 text-orange-300">
            <span className="h-3 w-3 rounded bg-orange-400" />
            Long liquidation
          </span>
          <span className="inline-flex items-center gap-1 text-fxos-gold">
            <span className="h-3 w-3 rounded bg-fxos-gold" />
            Short liquidation
          </span>
        </div>

        <div className="relative rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper px-4 py-5">
          <ChartGridLines />
          <div className="relative z-10 max-h-[720px] space-y-1 overflow-y-auto pr-2">
            {bins.map((bin, index) => (
              <HeatmapChartRow
                key={`${binPrice(bin)}-${index}`}
                bin={bin}
                maxLeft={maxLeft}
                maxRight={maxRight}
                markPrice={data.markPrice}
              />
            ))}
          </div>
          <div className="relative z-10 mt-4 grid grid-cols-[78px_minmax(0,1fr)] items-center gap-3 text-xs text-fxos-text-muted">
            <div />
            <div className="grid grid-cols-5 font-mono">
              <span>{formatMoney(maxLeft)}</span>
              <span className="text-center">{formatMoney(maxLeft / 2)}</span>
              <span className="text-center">$0</span>
              <span className="text-center">{formatMoney(maxRight / 2)}</span>
              <span className="text-right">{formatMoney(maxRight)}</span>
            </div>
          </div>
        </div>

        <div className="mt-4 grid gap-3 md:grid-cols-3">
          <DetailMetricCard
            label="Flush fuel below"
            value={formatMoney(longLiqTotal)}
            note="Long liquidations can force sell into downside breaks."
            tone="red"
          />
          <DetailMetricCard
            label="Squeeze fuel above"
            value={formatMoney(shortLiqTotal)}
            note="Short liquidations can force buy into upside breaks."
            tone="cyan"
          />
          <DetailMetricCard
            label="Bin step"
            value={formatNumber(data.binStep, 4)}
            note={`${bins.length} active price bins returned.`}
            tone="neutral"
          />
        </div>
      </div>
    </section>
  )
}

export function StrategyStudioPage() {
  const { token } = useAuth()
  const { language } = useLanguage()
  const navigate = useNavigate()
  const [strategies, setStrategies] = useState<Strategy[]>([])
  const [selectedStrategy, setSelectedStrategy] = useState<Strategy | null>(
    null
  )
  const [editingConfig, setEditingConfig] = useState<StrategyConfig | null>(
    null
  )
  const [symbols, setSymbols] = useState<MarketSymbol[]>([])
  const [signals, setSignals] = useState<VergexSignalItem[]>([])
  const [detailSignal, setDetailSignal] = useState<VergexSignalItem | null>(
    null
  )
  const [signalLab, setSignalLab] = useState<VergexSignalLabResponse | null>(
    null
  )
  const [heatmap, setHeatmap] = useState<VergexHeatmapResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [symbolsLoading, setSymbolsLoading] = useState(false)
  const [symbolsError, setSymbolsError] = useState('')
  const [signalsLoading, setSignalsLoading] = useState(false)
  const [signalsError, setSignalsError] = useState('')
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailError, setDetailError] = useState('')
  const [detailLiqBand, setDetailLiqBand] = useState('15')
  const [listMode, setListMode] = useState<ListMode>('claw402')
  const [hasChanges, setHasChanges] = useState(false)
  const [activeTab, setActiveTab] = useState<'editor' | 'signal'>('editor')

  const aiConfig = editingConfig?.ai_config || null
  const coinSource = aiConfig?.coin_source
  const indicators = aiConfig?.indicators
  const risk = aiConfig?.risk_control
  const selectedSymbols = coinSource?.static_coins || []
  // Signal Board scope filter (UI-only; does NOT touch strategy config)
  const [scope, setScope] = useState<Scope>('all')
  const activeProfile = profileFromRisk(risk)

  const signalMap = useMemo(() => {
    const map = new Map<string, VergexSignalItem>()
    for (const item of signals) {
      map.set(normalizeSymbol(item.symbol), item)
    }
    return map
  }, [signals])

  const visibleSymbols = useMemo(() => {
    // 'all' defaults to the tradefi pool (crypto is driven by the Claw402
    // board); picking a specific category (including crypto) shows it fully.
    const scoped =
      scope === 'all'
        ? symbols.filter((item) => item.category !== 'crypto')
        : symbols.filter((item) => item.category === scope)
    return [...scoped].sort((a, b) => {
      const aSignal = signalMap.get(normalizeSymbol(a.symbol))
      const bSignal = signalMap.get(normalizeSymbol(b.symbol))
      const aRank = aSignal?.rank || Number.MAX_SAFE_INTEGER
      const bRank = bSignal?.rank || Number.MAX_SAFE_INTEGER
      if (aRank !== bRank) return aRank - bRank
      return (b.volume_24h || 0) - (a.volume_24h || 0)
    })
  }, [scope, signalMap, symbols])

  const visibleSignalItems = useMemo(() => {
    const scoped =
      scope === 'all'
        ? signals
        : signals.filter((item) => item.category === scope)
    return scoped.slice().sort(compareSignalItems)
  }, [scope, signals])

  const selectedSet = useMemo(
    () => new Set(selectedSymbols.map(normalizeSymbol)),
    [selectedSymbols]
  )

  const loadStrategies = useCallback(
    async (preferredStrategyId?: string) => {
      if (!token) return
      setLoading(true)
      try {
        const result = await api.getStrategies()
        setStrategies(result)
        const next =
          (preferredStrategyId
            ? result.find((item) => item.id === preferredStrategyId)
            : null) ||
          result.find((item) => item.is_active) ||
          result[0] ||
          null
        setSelectedStrategy(next)
        setEditingConfig(next ? simplifyConfig(next.config) : null)
        setHasChanges(false)
      } catch (err) {
        notify.error(
          err instanceof Error
            ? err.message
            : t('strategyStudio.failedToLoad', language)
        )
      } finally {
        setLoading(false)
      }
    },
    [token]
  )

  const loadSymbols = useCallback(async () => {
    setSymbolsLoading(true)
    setSymbolsError('')
    try {
      const result = await api.getSymbols('hyperliquid-xyz')
      setSymbols(result.symbols || [])
    } catch (err) {
      setSymbolsError(
        err instanceof Error
          ? err.message
          : t('strategyStudio.symbolListUnavailable', language)
      )
    } finally {
      setSymbolsLoading(false)
    }
  }, [])

  const loadSignals = useCallback(async () => {
    if (!token) return
    setSignalsLoading(true)
    setSignalsError('')
    try {
      const result = await api.getVergexSignalRanking(claw402BoardLimit)
      setSignals(result.items || [])
      setListMode('claw402')
    } catch (err) {
      setSignalsError(
        err instanceof Error
          ? err.message
          : t('strategyStudio.boardUnavailable', language)
      )
    } finally {
      setSignalsLoading(false)
    }
  }, [token])

  const loadSignalDetail = useCallback(
    async (item: VergexSignalItem, bandOverride?: string) => {
      if (!token) return
      const nextBand =
        bandOverride || detailLiqBand || coinSource?.vergex_liq_band || '15'
      const params = {
        marketType: signalMarketType(item),
        symbol: strategySymbolForSignal(item),
        chain: 'mainnet',
        liqBand: nextBand,
      }

      setDetailLiqBand(nextBand)
      setDetailSignal(item)
      setSignalLab(null)
      setHeatmap(null)
      setDetailError('')
      setDetailLoading(true)

      window.requestAnimationFrame(() => {
        document
          .getElementById('claw402-detail-panel')
          ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
      })

      const [labResult, heatmapResult] = await Promise.allSettled([
        api.getVergexSignalLab(params),
        api.getVergexCostLiquidationHeatmap(params),
      ])

      const errors: string[] = []
      if (labResult.status === 'fulfilled') {
        setSignalLab(labResult.value)
      } else {
        errors.push(
          `Signal Lab: ${
            labResult.reason instanceof Error
              ? labResult.reason.message
              : 'unavailable'
          }`
        )
      }

      if (heatmapResult.status === 'fulfilled') {
        setHeatmap(heatmapResult.value)
      } else {
        errors.push(
          `Heatmap: ${
            heatmapResult.reason instanceof Error
              ? heatmapResult.reason.message
              : 'unavailable'
          }`
        )
      }

      setDetailError(errors.join(' · '))
      setDetailLoading(false)
    },
    [coinSource?.vergex_liq_band, detailLiqBand, token]
  )

  const selectDetailBand = useCallback(
    (band: string) => {
      setDetailLiqBand(band)
      if (detailSignal) {
        void loadSignalDetail(detailSignal, band)
      }
    },
    [detailSignal, loadSignalDetail]
  )

  useEffect(() => {
    void loadStrategies()
    void loadSymbols()
    void loadSignals()
  }, [loadStrategies, loadSymbols, loadSignals])

  const patchAI = (patch: Partial<AIStrategyConfig>) => {
    setEditingConfig((prev) => {
      const base = simplifyConfig(prev)
      return {
        ...base,
        language: language as 'zh' | 'en',
        ai_config: {
          ...base.ai_config!,
          ...patch,
        },
      }
    })
    setHasChanges(true)
  }

  const patchCoinSource = (patch: Partial<CoinSourceConfig>) => {
    patchAI({
      coin_source: defaultCoinSource({
        ...coinSource,
        ...patch,
      }),
    })
  }

  const patchIndicators = (patch: Partial<IndicatorConfig>) => {
    patchAI({
      indicators: defaultIndicators({
        ...indicators,
        ...patch,
      }),
    })
  }

  const patchRisk = (patch: Partial<RiskControlConfig>) => {
    patchAI({
      risk_control: defaultRisk({
        ...risk,
        ...patch,
      }),
    })
  }

  const createStrategy = async () => {
    if (!token) return
    try {
      const response = await fetch(
        `${apiUrl('/strategies/default-config')}?lang=${language}`,
        { headers: { Authorization: `Bearer ${token}` } }
      )
      const defaultConfig = response.ok
        ? simplifyConfig(await response.json())
        : simplifyConfig(null)
      defaultConfig.language = language as 'zh' | 'en'
      defaultConfig.ai_config = {
        ...defaultConfig.ai_config!,
        coin_source: defaultCoinSource({
          ...defaultConfig.ai_config?.coin_source,
          static_coins: [],
          hyper_rank_category: 'all',
          vergex_limit: 10,
          vergex_market_type: 'all',
        }),
        indicators: defaultIndicators({
          ...defaultConfig.ai_config?.indicators,
          klines: {
            primary_timeframe: '15m',
            primary_count: 30,
            enable_multi_timeframe: false,
            selected_timeframes: ['15m'],
          },
        }),
        risk_control: defaultRisk({
          ...defaultConfig.ai_config?.risk_control,
          max_positions: 2,
          btc_eth_max_leverage: 10,
          altcoin_max_leverage: 10,
          btc_eth_max_position_value_ratio: 10,
          altcoin_max_position_value_ratio: 10,
          max_margin_usage: 1.0,
          min_confidence: 78,
          min_risk_reward_ratio: 3,
        }),
        custom_prompt:
          'FXOS Autopilot reads the Claw402.ai board each cycle, fetches Signal Lab and cost/liquidation structure for every candidate, confirms with raw OHLCV candles, then trades only when the full-size setup is justified.',
        prompt_sections: undefined,
      }
      const created = await api.createStrategy({
        name: text(
          language,
          'FXOS Claw402 Auto Strategy',
          'FXOS Claw402 Auto Strategy'
        ),
        description: text(
          language,
          text(
            language,
            '单一内置策略：读取 Claw402.ai 看板，获取每个币种详情，然后用原始 K 线交易。',
            'The single built-in strategy: read the Claw402.ai board, fetch per-symbol details, then trade with raw candles.'
          ),
          text(
            language,
            '单一内置策略：读取 Claw402.ai 看板，获取每个币种详情，然后用原始 K 线交易。',
            'The single built-in strategy: read the Claw402.ai board, fetch per-symbol details, then trade with raw candles.'
          )
        ),
        config: defaultConfig,
      })
      await loadStrategies(created.id)
      setHasChanges(false)
    } catch (err) {
      notify.error(
        err instanceof Error
          ? err.message
          : t('strategyStudio.failedToCreate', language)
      )
    }
  }

  const saveStrategy = async (
    activateAfter = false,
    overrideConfig?: StrategyConfig,
    successMessage?: string
  ) => {
    if (!token || !selectedStrategy || (!editingConfig && !overrideConfig))
      return
    setSaving(true)
    try {
      const config = simplifyConfig(overrideConfig || editingConfig)
      config.language = language as 'zh' | 'en'
      const response = await fetch(
        apiUrl(`/strategies/${selectedStrategy.id}`),
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({
            name: selectedStrategy.name,
            description: selectedStrategy.description,
            config,
            is_public: selectedStrategy.is_public,
            config_visible: selectedStrategy.config_visible,
          }),
        }
      )
      if (!response.ok)
        throw new Error(t('strategyStudio.failedToSave', language))
      if (activateAfter) {
        await api.activateStrategy(selectedStrategy.id)
      }
      setHasChanges(false)
      notify.success(
        successMessage ||
          (activateAfter
            ? t('strategyStudio.savedAndActivated', language)
            : t('strategyStudio.strategySaved', language))
      )
      await loadStrategies(selectedStrategy.id)
    } catch (err) {
      notify.error(
        err instanceof Error
          ? err.message
          : t('strategyStudio.failedToSave', language)
      )
    } finally {
      setSaving(false)
    }
  }

  const buildUnifiedClaw402Config = (): StrategyConfig => {
    const base = simplifyConfig(editingConfig)
    base.language = language as 'zh' | 'en'
    const currentCoinSource = base.ai_config?.coin_source
    const pinnedCoins = currentCoinSource?.static_coins || []

    // Pinned universe wins: when the operator pinned coins, launch with their
    // config untouched so the user prompt is built around those symbols
    // (defaultCoinSource already yields source_type='static' when coins exist).
    // Only fall back to the unified Claw402 board template when nothing is
    // pinned — never wipe a user-selected coin list at launch time.
    if (pinnedCoins.length > 0) {
      return base
    }

    base.ai_config = {
      ...base.ai_config!,
      coin_source: defaultCoinSource({
        ...currentCoinSource,
        static_coins: [],
        hyper_rank_category: 'all',
        vergex_limit: 10,
        vergex_market_type: 'all',
        vergex_chain: 'hyperliquid',
      }),
      indicators: defaultIndicators({
        ...base.ai_config?.indicators,
        klines: {
          primary_timeframe: '15m',
          primary_count: 30,
          enable_multi_timeframe: false,
          selected_timeframes: ['15m'],
        },
      }),
      risk_control: defaultRisk({
        ...base.ai_config?.risk_control,
        max_positions: 4,
        btc_eth_max_leverage: 20,
        altcoin_max_leverage: 20,
        // 5× equity notional per position — 4 positions = 20x total account
        // notional (full margin, ~5% liquidation cushion). Aggressive by
        // operator choice; the 0.4 short-signal floor keeps the book balanced.
        btc_eth_max_position_value_ratio: 5,
        altcoin_max_position_value_ratio: 5,
        max_margin_usage: 1.0,
        min_confidence: 78,
        min_risk_reward_ratio: 3,
      }),
      custom_prompt:
        'Run FXOS Autopilot: use the Claw402.ai ranking as the candidate universe, verify each candidate with Signal Lab and cost/liquidation structure, confirm timing with raw OHLCV candles, and only open full-size 10x positions when the setup is strong enough.',
      prompt_sections: undefined,
    }
    return base
  }

  const startUnifiedClaw402Agent = async () => {
    if (!selectedStrategy) return

    setSaving(true)
    try {
      // The shared launcher runs the server-side preflight (fresh wallet and
      // exchange balances) BEFORE ensureStrategy, so a failed launch never
      // mutates or activates the strategy as a side effect.
      const outcome = await launchAutopilot({
        scanIntervalMinutes: 15,
        ensureStrategy: async () => {
          const config = buildUnifiedClaw402Config()
          setEditingConfig(config)
          await api.updateStrategy(selectedStrategy.id, {
            name: selectedStrategy.name,
            description:
              selectedStrategy.description ||
              'Autonomous market selection powered by Claw402.ai Signal Lab, liquidation structure, and raw candles.',
            config,
          })
          await api.activateStrategy(selectedStrategy.id)
          return selectedStrategy.id
        },
      })

      if (!outcome.ok) {
        notify.error(outcome.message)
        const setupTarget =
          outcome.kind === 'error' ? null : outcome.setupTarget
        if (setupTarget) {
          navigate(`${ROUTES.traders}?setup=${setupTarget}`)
        }
        return
      }

      if (outcome.warning) {
        notify.warning(outcome.warning)
      }
      notify.success('FXOS Autopilot started')
      setHasChanges(false)
      await loadStrategies(selectedStrategy.id)
      navigate(buildDashboardPath(outcome.traderId))
    } finally {
      setSaving(false)
    }
  }

  const activateStrategy = async () => {
    if (!selectedStrategy) return
    try {
      await api.activateStrategy(selectedStrategy.id)
      notify.success(t('strategyStudio.strategyActivated', language))
      await loadStrategies(selectedStrategy.id)
    } catch (err) {
      notify.error(
        err instanceof Error
          ? err.message
          : t('strategyStudio.failedToActivate', language)
      )
    }
  }

  const deleteStrategy = async () => {
    if (!selectedStrategy || selectedStrategy.is_active) return
    const ok = await confirmToast(
      t('strategyStudio.confirmDeleteStrategy', language),
      {
        title: t('strategyStudio.confirmDelete', language),
        okText: t('strategyStudio.delete', language),
        cancelText: t('strategyStudio.cancel', language),
      }
    )
    if (!ok) return
    try {
      await api.deleteStrategy(selectedStrategy.id)
      notify.success(t('strategyStudio.strategyDeleted', language))
      await loadStrategies()
    } catch (err) {
      notify.error(
        err instanceof Error
          ? err.message
          : t('strategyStudio.failedToDelete', language)
      )
    }
  }

  const toggleSymbol = (symbol: string) => {
    const normalized = normalizeSymbol(symbol)
    const next = selectedSet.has(normalized)
      ? selectedSymbols.filter((item) => normalizeSymbol(item) !== normalized)
      : [...selectedSymbols, symbol].slice(0, 10)
    const nextLimit =
      next.length > 0
        ? Math.min(next.length, 10)
        : Math.min(Math.max(coinSource?.vergex_limit || 5, 5), 10)
    patchCoinSource({
      static_coins: next,
      vergex_limit: nextLimit,
      vergex_market_type: 'all',
    })
  }

  const setTimeframe = (timeframe: string) => {
    patchIndicators({
      klines: {
        primary_timeframe: timeframe,
        primary_count: indicators?.klines.primary_count || 30,
        enable_multi_timeframe: false,
        selected_timeframes: [timeframe],
      },
    })
  }

  const setBarCount = (count: number) => {
    patchIndicators({
      klines: {
        primary_timeframe: indicators?.klines.primary_timeframe || '15m',
        primary_count: count,
        enable_multi_timeframe: false,
        selected_timeframes: [indicators?.klines.primary_timeframe || '15m'],
      },
    })
  }

  const setLeverage = (leverage: number) => {
    patchRisk({
      btc_eth_max_leverage: leverage,
      altcoin_max_leverage: leverage,
    })
  }

  const applyProfile = (profile: (typeof profileOptions)[number]) => {
    setEditingConfig((prev) => {
      const base = simplifyConfig(prev)
      const currentAI = base.ai_config!
      return {
        ...base,
        language: language as 'zh' | 'en',
        ai_config: {
          ...currentAI,
          coin_source: defaultCoinSource({
            ...currentAI.coin_source,
            vergex_limit: profile.topN,
          }),
          indicators: defaultIndicators({
            ...currentAI.indicators,
            klines: {
              primary_timeframe: profile.timeframe,
              primary_count: profile.bars,
              enable_multi_timeframe: false,
              selected_timeframes: [profile.timeframe],
            },
          }),
          risk_control: defaultRisk({
            ...currentAI.risk_control,
            max_positions: profile.maxPositions,
            btc_eth_max_leverage: profile.leverage,
            altcoin_max_leverage: profile.leverage,
            max_margin_usage: profile.margin,
            min_confidence: profile.confidence,
          }),
          custom_prompt: text(language, profile.promptZh, profile.promptEn),
          prompt_sections: undefined,
        },
      }
    })
    setHasChanges(true)
  }

  if (loading) {
    return (
      <div className="flex min-h-[70vh] items-center justify-center">
        <Loader2 className="h-7 w-7 animate-spin text-fxos-gold" />
      </div>
    )
  }

  return (
    <DeepVoidBackground className="min-h-[calc(100vh-64px)] bg-fxos-bg">
      <div className="border-b border-[var(--panel-border)] bg-fxos-bg/75 px-5 py-4 backdrop-blur">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold text-fxos-text">
              {t('strategyStudio.title', language)}
            </h1>
            <p className="mt-1 text-sm text-fxos-text-muted">
              {t('strategyStudio.subtitle', language)}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => navigate(ROUTES.strategyPrompt)}
              className="inline-flex items-center gap-2 rounded-lg border border-[var(--panel-border)] px-4 py-2 text-sm font-semibold text-fxos-text-muted hover:border-fxos-gold/40 hover:text-fxos-gold transition-colors"
            >
              <Sparkles className="h-4 w-4" />
              {t('promptStudio.title', language)}
            </button>
            <button
              type="button"
              onClick={startUnifiedClaw402Agent}
              disabled={saving || !selectedStrategy}
              className="inline-flex items-center gap-2 rounded-lg bg-fxos-gold px-4 py-2 text-sm font-semibold text-fxos-bg hover:bg-fxos-gold-highlight"
            >
              {saving ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Bot className="h-4 w-4" />
              )}
              {t('strategyStudio.launchAutopilot', language)}
            </button>
          </div>
        </div>
      </div>

      <div className="grid min-h-[calc(100vh-137px)] grid-cols-1 md:grid-cols-[280px_1fr]">
        <aside className="border-r border-[var(--panel-border)] bg-fxos-bg-deeper p-3 md:max-h-[calc(100vh-137px)] md:overflow-y-auto">
          <div className="mb-2 flex items-center justify-between px-2">
            <span className="text-xs font-medium uppercase tracking-wide text-fxos-text-muted">
              {t('strategyStudio.myStrategies', language)}
            </span>
            <span className="rounded bg-fxos-bg-lighter px-1.5 py-0.5 font-mono text-[10px] text-fxos-text-muted">
              {strategies.length}
            </span>
          </div>
          <button
            type="button"
            onClick={createStrategy}
            disabled={saving}
            className="mb-2 inline-flex w-full items-center justify-center gap-2 rounded-lg border border-fxos-gold/30 bg-fxos-gold/10 px-3 py-2 text-sm font-semibold text-fxos-gold transition-colors hover:bg-fxos-gold/15 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Plus className="h-4 w-4" />
            {t('strategyStudio.createStrategy', language)}
          </button>
          <div className="space-y-2">
            {strategies.map((strategy) => (
              <button
                key={strategy.id}
                type="button"
                onClick={() => {
                  setSelectedStrategy(strategy)
                  setEditingConfig(simplifyConfig(strategy.config))
                  setHasChanges(false)
                }}
                className={`w-full rounded-lg border px-3 py-3 text-left transition ${
                  selectedStrategy?.id === strategy.id
                    ? 'border-fxos-gold bg-fxos-gold/10'
                    : 'border-[var(--panel-border)] bg-fxos-bg-lighter hover:border-[var(--panel-border)]'
                }`}
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="line-clamp-2 text-sm font-medium text-fxos-text">
                    {strategy.name}
                  </span>
                  {strategy.is_active ? (
                    <span className="rounded bg-fxos-success/15 px-1.5 py-0.5 text-[10px] text-fxos-success">
                      {t('strategyStudio.active', language)}
                    </span>
                  ) : null}
                </div>
                {strategy.description ? (
                  <div className="mt-1 line-clamp-2 text-xs text-fxos-text-muted">
                    {strategy.description}
                  </div>
                ) : null}
              </button>
            ))}
          </div>
        </aside>

        <main className="overflow-y-auto p-5">
          {selectedStrategy && aiConfig && coinSource && indicators && risk ? (
            <div className="mx-auto max-w-7xl space-y-4">
              <section className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter p-4">
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div className="min-w-0 flex-1">
                    <input
                      value={selectedStrategy.name}
                      onChange={(event) => {
                        setSelectedStrategy({
                          ...selectedStrategy,
                          name: event.target.value,
                        })
                        setHasChanges(true)
                      }}
                      className="w-full bg-transparent text-lg font-semibold text-fxos-text outline-none"
                    />
                    <input
                      value={selectedStrategy.description || ''}
                      onChange={(event) => {
                        setSelectedStrategy({
                          ...selectedStrategy,
                          description: event.target.value,
                        })
                        setHasChanges(true)
                      }}
                      placeholder={t('strategyStudio.addDescription', language)}
                      className="mt-1 w-full bg-transparent text-sm text-fxos-text-muted outline-none placeholder:text-fxos-text-muted/50"
                    />
                    {hasChanges ? (
                      <div className="mt-2 text-xs text-fxos-gold">
                        {t('strategyStudio.unsavedChanges', language)}
                      </div>
                    ) : null}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      onClick={() => saveStrategy(true)}
                      disabled={saving}
                      className="inline-flex items-center gap-2 rounded-lg bg-fxos-success px-3 py-2 text-sm font-semibold text-fxos-bg disabled:cursor-not-allowed disabled:opacity-45"
                    >
                      {saving ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Check className="h-4 w-4" />
                      )}
                      {t('strategyStudio.saveAndUse', language)}
                    </button>
                    <button
                      type="button"
                      onClick={() => saveStrategy()}
                      disabled={saving || !hasChanges}
                      className="inline-flex items-center gap-2 rounded-lg bg-fxos-gold px-3 py-2 text-sm font-semibold text-fxos-bg disabled:cursor-not-allowed disabled:opacity-45"
                    >
                      {saving ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Save className="h-4 w-4" />
                      )}
                      {t('strategyStudio.save', language)}
                    </button>
                    {!selectedStrategy.is_active ? (
                      <button
                        type="button"
                        onClick={activateStrategy}
                        className="inline-flex items-center gap-2 rounded-lg border border-fxos-success/30 bg-fxos-success/10 px-3 py-2 text-sm text-fxos-success hover:bg-fxos-success/15"
                      >
                        <Check className="h-4 w-4" />
                        {t('strategyStudio.activateOnly', language)}
                      </button>
                    ) : null}
                    {!selectedStrategy.is_active ? (
                      <button
                        type="button"
                        onClick={deleteStrategy}
                        className="inline-flex items-center gap-2 rounded-lg border border-fxos-danger/25 bg-fxos-danger/10 px-3 py-2 text-sm text-fxos-danger hover:bg-fxos-danger/15"
                      >
                        <Trash2 className="h-4 w-4" />
                        {t('strategyStudio.delete', language)}
                      </button>
                    ) : null}
                  </div>
                </div>
              </section>

              <div className="flex items-center gap-1 rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper p-1">
                <button
                  type="button"
                  onClick={() => setActiveTab('editor')}
                  className={`flex-1 rounded-md px-4 py-2 text-sm font-semibold transition ${
                    activeTab === 'editor'
                      ? 'bg-fxos-gold/10 text-fxos-gold'
                      : 'text-fxos-text-muted hover:text-fxos-text'
                  }`}
                >
                  {t('strategyStudio.editorTab', language)}
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab('signal')}
                  className={`flex-1 rounded-md px-4 py-2 text-sm font-semibold transition ${
                    activeTab === 'signal'
                      ? 'bg-fxos-gold/10 text-fxos-gold'
                      : 'text-fxos-text-muted hover:text-fxos-text'
                  }`}
                >
                  {t('strategyStudio.signalBoard', language)}
                </button>
              </div>

              {activeTab === 'editor' ? (
                <>
                  <section className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter p-4">
                    <div className="mb-3 text-sm font-semibold text-fxos-text">
                      {t('strategyStudio.tradingStyle', language)}
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {profileOptions.map((profile) => (
                        <button
                          key={profile.value}
                          type="button"
                          onClick={() => applyProfile(profile)}
                          className={`rounded-lg border px-3 py-2 text-sm transition ${
                            activeProfile === profile.value
                              ? 'border-fxos-gold bg-fxos-gold/10 text-fxos-gold'
                              : 'border-[var(--panel-border)] text-fxos-text-muted hover:text-fxos-text'
                          }`}
                        >
                          {text(language, profile.zh, profile.en, profile.id)}
                        </button>
                      ))}
                    </div>
                  </section>

                  <section className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter p-4">
                    <div className="mb-3 text-sm font-semibold text-fxos-text">
                      {t('strategyStudio.coinSource', language)}
                    </div>
                    <div className="grid gap-3 md:grid-cols-2">
                      <button
                        type="button"
                        onClick={() => {
                          patchCoinSource({
                            static_coins: [],
                            vergex_market_type: 'all',
                          })
                          setListMode('claw402')
                          if (signals.length === 0) {
                            void loadSignals()
                          }
                        }}
                        className={`rounded-lg border p-4 text-left transition ${
                          selectedSymbols.length === 0
                            ? 'border-fxos-success bg-fxos-success/10'
                            : 'border-[var(--panel-border)] bg-fxos-bg-deeper hover:border-[var(--panel-border)]'
                        }`}
                      >
                        <div className="flex items-center justify-between gap-3">
                          <div className="text-sm font-semibold text-fxos-text">
                            {t('strategyStudio.followBoard', language)}
                          </div>
                          {selectedSymbols.length === 0 ? (
                            <Check className="h-4 w-4 text-fxos-success" />
                          ) : null}
                        </div>
                        <div className="mt-2 text-xs text-fxos-text-muted">
                          {t('strategyStudio.followBoardHint', language, {
                            n: coinSource.vergex_limit || 5,
                          })}
                        </div>
                      </button>

                      <button
                        type="button"
                        onClick={() => {
                          setListMode('pool')
                          setActiveTab('signal')
                          if (symbols.length === 0) {
                            void loadSymbols()
                          }
                        }}
                        className={`rounded-lg border p-4 text-left transition ${
                          selectedSymbols.length > 0
                            ? 'border-fxos-gold bg-fxos-gold/10'
                            : 'border-[var(--panel-border)] bg-fxos-bg-deeper hover:border-[var(--panel-border)]'
                        }`}
                      >
                        <div className="flex items-center justify-between gap-3">
                          <div className="text-sm font-semibold text-fxos-text">
                            {t('strategyStudio.pinnedUniverse', language)}
                          </div>
                          {selectedSymbols.length > 0 ? (
                            <Check className="h-4 w-4 text-fxos-gold" />
                          ) : null}
                        </div>
                        <div className="mt-2 text-xs text-fxos-text-muted">
                          {selectedSymbols.length > 0
                            ? t(
                                'strategyStudio.pinnedUniverseHintSelected',
                                language,
                                { count: selectedSymbols.length }
                              )
                            : t(
                                'strategyStudio.pinnedUniverseHintDefault',
                                language
                              )}
                        </div>
                      </button>
                    </div>
                    <div className="mt-3 flex flex-wrap items-center gap-3">
                      <span className="text-sm text-fxos-text-muted">
                        {selectedSymbols.length > 0
                          ? t(
                              'strategyStudio.pinnedUniverseHintSelected',
                              language,
                              { count: selectedSymbols.length }
                            )
                          : t('strategyStudio.followBoardHint', language, {
                              n: coinSource.vergex_limit || 5,
                            })}
                      </span>
                      {selectedSymbols.length === 0 ? (
                        <select
                          value={coinSource.vergex_limit || 5}
                          onChange={(event) =>
                            patchCoinSource({
                              vergex_limit: Math.max(
                                Number(event.target.value),
                                5
                              ),
                            })
                          }
                          className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg px-3 py-2 text-sm text-fxos-text"
                        >
                          {topNOptions.map((value) => (
                            <option key={value} value={value}>
                              Top {value}
                            </option>
                          ))}
                        </select>
                      ) : null}
                    </div>
                  </section>
                </>
              ) : (
                <section className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter p-4">
                  <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <div className="flex items-center gap-2 text-sm font-semibold text-fxos-text">
                        <Sparkles className="h-4 w-4 text-fxos-gold" />
                        {t('strategyStudio.signalBoard', language)}
                      </div>
                      <div className="mt-1 text-xs text-fxos-text-muted">
                        {t('strategyStudio.signalBoardDesc', language)}
                      </div>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <button
                        type="button"
                        onClick={() => {
                          if (signals.length === 0) {
                            void loadSignals()
                          } else {
                            setListMode('claw402')
                          }
                        }}
                        disabled={signalsLoading}
                        className={`items-center gap-2 rounded-lg border px-3 py-2 text-xs disabled:opacity-50 ${
                          listMode === 'claw402'
                            ? 'border-fxos-gold bg-fxos-gold/10 text-fxos-gold'
                            : 'border-[var(--panel-border)] text-fxos-text-muted hover:text-fxos-text'
                        }`}
                      >
                        <Sparkles className="h-3.5 w-3.5" />
                        {t(
                          signals.length === 0
                            ? 'strategyStudio.loadBoard'
                            : 'strategyStudio.boardMode',
                          language
                        )}
                      </button>
                      <button
                        type="button"
                        onClick={() => setListMode('pool')}
                        disabled={symbolsLoading}
                        className={`items-center gap-2 rounded-lg border px-3 py-2 text-xs disabled:opacity-50 ${
                          listMode === 'pool'
                            ? 'border-[var(--panel-border)] bg-fxos-bg-deeper text-fxos-text'
                            : 'border-[var(--panel-border)] text-fxos-text-muted hover:text-fxos-text'
                        }`}
                      >
                        <RefreshCw
                          className={`h-3.5 w-3.5 ${symbolsLoading ? 'animate-spin' : ''}`}
                        />
                        {t('strategyStudio.symbolPool', language)}
                      </button>
                      <button
                        type="button"
                        onClick={() => {
                          void loadSignals()
                        }}
                        disabled={signalsLoading}
                        className="inline-flex items-center gap-2 rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper px-3 py-2 text-xs text-fxos-text-muted hover:text-fxos-text disabled:opacity-50"
                      >
                        <RefreshCw
                          className={`h-3.5 w-3.5 ${signalsLoading ? 'animate-spin' : ''}`}
                        />
                        {t('strategyStudio.refresh', language)}
                      </button>
                      {selectedSymbols.length > 0 ? (
                        <button
                          type="button"
                          onClick={() =>
                            patchCoinSource({
                              static_coins: [],
                              vergex_market_type: 'all',
                            })
                          }
                          className="rounded-lg border border-[var(--panel-border)] px-3 py-2 text-xs text-fxos-text-muted hover:text-fxos-text"
                        >
                          {t('strategyStudio.clearSelected', language)}
                        </button>
                      ) : null}
                    </div>
                  </div>

                  <div className="mb-4 flex flex-wrap gap-2">
                    {scopeOptions.map((option) => {
                      const signalCount =
                        option.value === 'all'
                          ? signals.length
                          : signals.filter(
                              (item) => item.category === option.value
                            ).length
                      const poolCount =
                        option.value === 'all'
                          ? symbols.filter((item) => item.category !== 'crypto')
                              .length
                          : symbols.filter(
                              (item) => item.category === option.value
                            ).length
                      const count =
                        listMode === 'claw402' ? signalCount : poolCount
                      return (
                        <button
                          key={option.value}
                          type="button"
                          onClick={() => setScope(option.value)}
                          className={`rounded-lg border px-3 py-2 text-xs transition ${
                            scope === option.value
                              ? 'border-fxos-gold bg-fxos-gold/10 text-fxos-gold'
                              : 'border-[var(--panel-border)] bg-fxos-bg-deeper text-fxos-text-muted hover:text-fxos-text'
                          }`}
                        >
                          {option.en}
                          {count > 0 ? (
                            <span className="ml-2 opacity-70">{count}</span>
                          ) : null}
                        </button>
                      )
                    })}
                  </div>

                  {symbolsError || signalsError ? (
                    <div className="mb-4 rounded-lg border border-fxos-gold/20 bg-fxos-gold/10 px-3 py-2 text-xs text-fxos-gold">
                      {symbolsError || signalsError}
                    </div>
                  ) : null}

                  {listMode === 'claw402' &&
                  signals.length === 0 &&
                  !signalsLoading ? (
                    <button
                      type="button"
                      onClick={() => {
                        void loadSignals()
                      }}
                      className="mb-4 inline-flex items-center gap-2 rounded-lg border border-fxos-gold/30 bg-fxos-gold/10 px-4 py-3 text-sm font-semibold text-fxos-gold hover:bg-fxos-gold/15"
                    >
                      <Sparkles className="h-4 w-4" />
                      {t('strategyStudio.loadSignalBoard', language)}
                    </button>
                  ) : null}

                  <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                    {listMode === 'claw402' && signals.length > 0
                      ? visibleSignalItems.map((item) => {
                          const symbol = normalizeSymbol(item.symbol)
                          const selected = selectedSet.has(symbol)
                          const detailSelected = sameSignalItem(
                            detailSignal,
                            item
                          )
                          const bias = signalBiasInfo(item.bias, language)
                          const BiasIcon = bias.icon
                          return (
                            <div
                              key={`claw402-${item.rank || 0}-${symbol}`}
                              role="button"
                              tabIndex={0}
                              onClick={() => void loadSignalDetail(item)}
                              onKeyDown={(event) => {
                                if (
                                  event.key === 'Enter' ||
                                  event.key === ' '
                                ) {
                                  event.preventDefault()
                                  void loadSignalDetail(item)
                                }
                              }}
                              className={`cursor-pointer rounded-lg border p-3 text-left transition ${
                                detailSelected || selected
                                  ? 'border-fxos-gold bg-fxos-gold/10'
                                  : 'border-[var(--panel-border)] bg-fxos-bg-deeper hover:border-[var(--panel-border)]'
                              }`}
                            >
                              <div className="flex items-center justify-between gap-2">
                                <span className="font-mono text-base font-semibold text-fxos-text">
                                  {symbol}
                                </span>
                                <span className="font-mono text-xs text-fxos-gold">
                                  #{item.rank || '-'}
                                </span>
                              </div>
                              <div className="mt-4 flex items-center justify-between gap-3">
                                <div
                                  className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-1 text-xs font-semibold ${bias.classes}`}
                                >
                                  <BiasIcon className="h-3.5 w-3.5" />
                                  {bias.label}
                                </div>
                                <span className="font-mono text-xs text-fxos-text-muted">
                                  {formatSignalStrength(item)}
                                </span>
                              </div>
                              <div className="mt-4 flex items-center justify-between gap-3 border-t border-[var(--panel-border)] pt-3 text-[11px] uppercase tracking-wide text-fxos-text-muted">
                                <span>
                                  {categoryLabel(item.category, 'en')}
                                </span>
                                <span>{signalMarketType(item)}</span>
                              </div>
                            </div>
                          )
                        })
                      : visibleSymbols.map((item) => {
                          const symbol = normalizeSymbol(item.symbol)
                          const signal = signalMap.get(symbol)
                          const selected = selectedSet.has(symbol)
                          return (
                            <button
                              key={`${item.exchange}-${symbol}`}
                              type="button"
                              onClick={() => toggleSymbol(symbol)}
                              className={`rounded-lg border p-3 text-left transition ${
                                selected
                                  ? 'border-fxos-gold bg-fxos-gold/10'
                                  : 'border-[var(--panel-border)] bg-fxos-bg-deeper hover:border-[var(--panel-border)]'
                              }`}
                            >
                              <div className="flex items-center justify-between gap-2">
                                <span className="font-mono text-sm font-semibold text-fxos-text">
                                  {symbol}
                                </span>
                                <span className="text-[10px] text-fxos-text-muted">
                                  {signal?.rank
                                    ? `#${signal.rank}`
                                    : formatChange(item.change_24h_pct)}
                                </span>
                              </div>
                              <div className="mt-2 flex items-center justify-between gap-2 text-[11px] text-fxos-text-muted">
                                <span>
                                  {categoryLabel(item.category, 'en')}
                                </span>
                                <span>
                                  {signal?.bias ||
                                    (item.mark_price
                                      ? `$${item.mark_price.toFixed(2)}`
                                      : 'ready')}
                                </span>
                              </div>
                            </button>
                          )
                        })}
                  </div>

                  {listMode === 'claw402' ? (
                    <div
                      id="claw402-detail-panel"
                      className="mt-4 scroll-mt-28 space-y-4"
                    >
                      {detailSignal ? (
                        <>
                          <section className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter px-4 py-3">
                            <div className="grid gap-3 lg:grid-cols-[1fr_auto] lg:items-center">
                              <div className="min-w-0">
                                <div className="flex flex-wrap items-center gap-2">
                                  <span className="font-mono text-xl font-semibold text-fxos-text">
                                    {normalizeSymbol(detailSignal.symbol)}
                                  </span>
                                  <span className="rounded-md bg-fxos-gold/10 px-2 py-1 text-xs font-semibold text-fxos-gold">
                                    #{detailSignal.rank || '-'}
                                  </span>
                                  <span className="rounded-md bg-fxos-bg-deeper px-2 py-1 text-xs text-fxos-text-muted">
                                    {categoryLabel(detailSignal.category, 'en')}
                                  </span>
                                </div>
                                <div className="mt-2 flex flex-wrap gap-2 font-mono text-xs text-fxos-text-muted">
                                  <span>{signalMarketType(detailSignal)}</span>
                                  <span>·</span>
                                  <span>
                                    {strategySymbolForSignal(detailSignal)}
                                  </span>
                                  <span>·</span>
                                  <span>mainnet</span>
                                  <span>·</span>
                                  <span>±{detailLiqBand}% band</span>
                                </div>
                              </div>
                              <div className="flex flex-wrap items-center gap-2 lg:justify-end">
                                <button
                                  type="button"
                                  onClick={() =>
                                    void loadSignalDetail(
                                      detailSignal,
                                      detailLiqBand
                                    )
                                  }
                                  disabled={detailLoading}
                                  className="inline-flex items-center gap-2 rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper px-3 py-2 text-xs text-fxos-text-muted hover:text-fxos-text disabled:opacity-50"
                                >
                                  <RefreshCw
                                    className={`h-3.5 w-3.5 ${
                                      detailLoading ? 'animate-spin' : ''
                                    }`}
                                  />
                                  {t('strategyStudio.refresh', language)}
                                </button>
                              </div>
                            </div>
                            {detailLoading ? (
                              <div className="mt-3 inline-flex items-center gap-2 text-xs text-fxos-text-muted">
                                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                {t('strategyStudio.loadingDetail', language)}
                              </div>
                            ) : null}
                            {detailError ? (
                              <div className="mt-3 rounded-md border border-fxos-gold/20 bg-fxos-gold/10 px-3 py-2 text-xs text-fxos-gold">
                                {detailError}
                              </div>
                            ) : null}
                          </section>

                          <CostLiquidationHeatmap heatmap={heatmap} />
                          <SignalLabPanel
                            lab={signalLab}
                            activeBand={detailLiqBand}
                            loading={detailLoading}
                            onBandChange={selectDetailBand}
                            language={language}
                          />
                        </>
                      ) : (
                        <div className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper px-4 py-4 text-sm text-fxos-text-muted">
                          {t('strategyStudio.boardEmptyHint', language)}
                        </div>
                      )}
                    </div>
                  ) : null}

                  {listMode === 'claw402' &&
                  signals.length > 0 &&
                  visibleSignalItems.length === 0 ? (
                    <div className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper px-3 py-3 text-sm text-fxos-text-muted">
                      {t('strategyStudio.noClaw402Markets', language)}
                    </div>
                  ) : null}

                  {listMode === 'pool' &&
                  visibleSymbols.length === 0 &&
                  !symbolsLoading ? (
                    <div className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper px-3 py-3 text-sm text-fxos-text-muted">
                      {t('strategyStudio.noMarkets', language)}
                    </div>
                  ) : null}
                </section>
              )}

              <details className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-deeper p-4">
                <summary className="cursor-pointer text-sm font-semibold text-fxos-text">
                  {t('strategyStudio.advancedSettings', language)}
                </summary>
                <div className="mt-4 grid gap-4 lg:grid-cols-2">
                  <div className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter p-4">
                    <div className="mb-4 flex items-center gap-2 text-sm font-semibold text-fxos-text">
                      <Sparkles className="h-4 w-4 text-fxos-gold" />
                      {t('strategyStudio.rawCandles', language)}
                    </div>
                    <div className="space-y-4">
                      <div>
                        <div className="mb-2 text-xs text-fxos-text-muted">
                          {t('strategyStudio.timeframe', language)}
                        </div>
                        <div className="flex flex-wrap gap-2">
                          {timeframeOptions.map((timeframe) => (
                            <button
                              key={timeframe}
                              type="button"
                              onClick={() => setTimeframe(timeframe)}
                              className={`rounded-lg border px-3 py-2 text-sm ${
                                indicators.klines.primary_timeframe ===
                                timeframe
                                  ? 'border-fxos-gold bg-fxos-gold/10 text-fxos-gold'
                                  : 'border-[var(--panel-border)] text-fxos-text-muted hover:text-fxos-text'
                              }`}
                            >
                              {timeframe}
                            </button>
                          ))}
                        </div>
                      </div>
                      <div>
                        <div className="mb-2 text-xs text-fxos-text-muted">
                          {t('strategyStudio.bars', language)}
                        </div>
                        <div className="flex flex-wrap gap-2">
                          {barCountOptions.map((count) => (
                            <button
                              key={count}
                              type="button"
                              onClick={() => setBarCount(count)}
                              className={`rounded-lg border px-3 py-2 text-sm ${
                                indicators.klines.primary_count === count
                                  ? 'border-fxos-gold bg-fxos-gold/10 text-fxos-gold'
                                  : 'border-[var(--panel-border)] text-fxos-text-muted hover:text-fxos-text'
                              }`}
                            >
                              {count}
                            </button>
                          ))}
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter p-4">
                    <div className="mb-4 flex items-center gap-2 text-sm font-semibold text-fxos-text">
                      <Shield className="h-4 w-4 text-fxos-success" />
                      {t('strategyStudio.tradingParameters', language)}
                    </div>
                    <div className="grid gap-4 sm:grid-cols-3">
                      <label className="space-y-2">
                        <span className="text-xs text-fxos-text-muted">
                          {t('strategyStudio.maxPositions', language)}
                        </span>
                        <select
                          value={risk.max_positions}
                          onChange={(event) =>
                            patchRisk({
                              max_positions: Number(event.target.value),
                            })
                          }
                          className="w-full rounded-lg border border-[var(--panel-border)] bg-fxos-bg px-3 py-2 text-sm text-fxos-text"
                        >
                          {[1, 2, 3, 4, 5].map((value) => (
                            <option key={value} value={value}>
                              {value}
                            </option>
                          ))}
                        </select>
                      </label>
                      <label className="space-y-2">
                        <span className="text-xs text-fxos-text-muted">
                          {t('strategyStudio.leverage', language)}
                        </span>
                        <select
                          value={risk.altcoin_max_leverage}
                          onChange={(event) =>
                            setLeverage(Number(event.target.value))
                          }
                          className="w-full rounded-lg border border-[var(--panel-border)] bg-fxos-bg px-3 py-2 text-sm text-fxos-text"
                        >
                          {[1, 2, 3, 5, 8, 10].map((value) => (
                            <option key={value} value={value}>
                              {value}x
                            </option>
                          ))}
                        </select>
                      </label>
                      <label className="space-y-2">
                        <span className="text-xs text-fxos-text-muted">
                          {t('strategyStudio.entryConfidence', language)}
                        </span>
                        <select
                          value={risk.min_confidence}
                          onChange={(event) =>
                            patchRisk({
                              min_confidence: Number(event.target.value),
                            })
                          }
                          className="w-full rounded-lg border border-[var(--panel-border)] bg-fxos-bg px-3 py-2 text-sm text-fxos-text"
                        >
                          {confidenceOptions.map((value) => (
                            <option key={value} value={value}>
                              {value}%
                            </option>
                          ))}
                        </select>
                      </label>
                    </div>
                  </div>
                </div>

                <div className="mt-4 rounded-lg border border-[var(--panel-border)] bg-fxos-bg-lighter p-4">
                  <div className="mb-2 text-sm font-semibold text-fxos-text">
                    {t('strategyStudio.strategyNote', language)}
                  </div>
                  <textarea
                    value={aiConfig.custom_prompt || ''}
                    onChange={(event) =>
                      patchAI({ custom_prompt: event.target.value })
                    }
                    placeholder={t(
                      'strategyStudio.strategyNotePlaceholder',
                      language
                    )}
                    className="h-28 w-full resize-none rounded-lg border border-[var(--panel-border)] bg-fxos-bg px-3 py-2 text-sm text-fxos-text outline-none placeholder:text-fxos-text-muted/50"
                  />
                </div>
              </details>
            </div>
          ) : (
            <div className="flex h-full items-center justify-center">
              <button
                type="button"
                onClick={createStrategy}
                className="inline-flex items-center gap-2 rounded-lg bg-fxos-gold px-4 py-2 text-sm font-semibold text-fxos-bg hover:bg-fxos-gold-highlight"
              >
                <Plus className="h-4 w-4" />
                {t('strategyStudio.initializeAutopilot', language)}
              </button>
            </div>
          )}
        </main>
      </div>
    </DeepVoidBackground>
  )
}

export default StrategyStudioPage
