import { useState, useEffect, useCallback } from 'react'
import {
  Shield,
  TrendingUp,
  AlertTriangle,
  Activity,
  Box,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import type { GridRiskInfo } from '../../types'
import { gridRisk, ts } from '../../i18n/strategy-translations'
import { apiUrl } from '../../lib/api/helpers'

interface GridRiskPanelProps {
  traderId: string
  language?: string
  refreshInterval?: number // ms, default 5000
}

export function GridRiskPanel({
  traderId,
  language = 'en',
  refreshInterval = 5000,
}: GridRiskPanelProps) {
  const [riskInfo, setRiskInfo] = useState<GridRiskInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [expanded, setExpanded] = useState(false)

  const fetchRiskInfo = useCallback(async () => {
    try {
      const token = localStorage.getItem('auth_token')
      const response = await fetch(apiUrl(`/traders/${traderId}/grid-risk`), {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`)
      }

      const data = await response.json()
      setRiskInfo(data)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [traderId])

  useEffect(() => {
    fetchRiskInfo()
    const interval = setInterval(fetchRiskInfo, refreshInterval)
    return () => clearInterval(interval)
  }, [fetchRiskInfo, refreshInterval])

  const getRegimeColor = (regime: string) => {
    switch (regime) {
      case 'narrow':
        return 'var(--fxos-success)'
      case 'standard':
        return 'var(--fxos-gold)'
      case 'wide':
        return 'var(--fxos-gold)'
      case 'volatile':
        return 'var(--fxos-danger)'
      case 'trending':
        return 'var(--fxos-gold)'
      default:
        return 'var(--text-secondary)'
    }
  }

  const getBreakoutColor = (level: string) => {
    switch (level) {
      case 'none':
        return 'var(--fxos-success)'
      case 'short':
        return 'var(--fxos-gold)'
      case 'mid':
        return 'var(--fxos-gold)'
      case 'long':
        return 'var(--fxos-danger)'
      default:
        return 'var(--text-secondary)'
    }
  }

  const getPositionColor = (percent: number) => {
    if (percent < 50) return 'var(--fxos-success)'
    if (percent < 80) return 'var(--fxos-gold)'
    return 'var(--fxos-danger)'
  }

  const formatPrice = (price: number) => {
    if (price === 0) return '-'
    if (price >= 1000)
      return price.toLocaleString('en-US', {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      })
    if (price >= 1) return price.toFixed(4)
    return price.toFixed(6)
  }

  const formatUSD = (value: number) => {
    return `$${value.toLocaleString('en-US', { minimumFractionDigits: 0, maximumFractionDigits: 0 })}`
  }

  const cardStyle = {
    background: 'var(--panel-bg)',
    border: '1px solid var(--panel-border)',
  }

  if (loading) {
    return (
      <div
        className="p-3 text-center text-xs"
        style={{ color: 'var(--text-secondary)' }}
      >
        {ts(gridRisk.loading, language)}
      </div>
    )
  }

  if (error) {
    return (
      <div
        className="p-3 text-center text-xs"
        style={{ color: 'var(--fxos-danger)' }}
      >
        {ts(gridRisk.error, language)}: {error}
      </div>
    )
  }

  if (!riskInfo) {
    return (
      <div
        className="p-3 text-center text-xs"
        style={{ color: 'var(--text-secondary)' }}
      >
        {ts(gridRisk.noData, language)}
      </div>
    )
  }

  return (
    <div className="rounded-lg" style={cardStyle}>
      {/* Collapsible Header */}
      <div
        className="flex items-center justify-between p-3 cursor-pointer hover:bg-fxos-bg-lighter transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center gap-2">
          <Shield className="w-4 h-4" style={{ color: 'var(--fxos-gold)' }} />
          <span
            className="font-medium text-sm"
            style={{ color: 'var(--text-primary)' }}
          >
            {ts(gridRisk.gridRisk, language)}
          </span>
        </div>
        <div className="flex items-center gap-3">
          {/* Summary badges when collapsed */}
          <div className="flex items-center gap-2 text-xs">
            <span
              className="px-2 py-0.5 rounded"
              style={{
                background: getRegimeColor(riskInfo.regime_level) + '20',
                color: getRegimeColor(riskInfo.regime_level),
              }}
            >
              {ts(
                gridRisk[
                  (riskInfo.regime_level || 'standard') as keyof typeof gridRisk
                ],
                language
              )}
            </span>
            <span
              className="font-mono"
              style={{ color: 'var(--text-primary)' }}
            >
              {riskInfo.effective_leverage.toFixed(1)}x
            </span>
            <span
              className="font-mono"
              style={{ color: getPositionColor(riskInfo.position_percent) }}
            >
              {riskInfo.position_percent.toFixed(0)}%
            </span>
          </div>
          {expanded ? (
            <ChevronUp
              className="w-4 h-4"
              style={{ color: 'var(--text-secondary)' }}
            />
          ) : (
            <ChevronDown
              className="w-4 h-4"
              style={{ color: 'var(--text-secondary)' }}
            />
          )}
        </div>
      </div>

      {/* Expanded Content */}
      {expanded && (
        <div className="px-3 pb-3 space-y-3">
          {/* Row 1: Leverage & Position */}
          <div className="grid grid-cols-2 gap-3">
            {/* Leverage */}
            <div
              className="p-2 rounded"
              style={{ background: 'var(--fxos-bg-lighter)' }}
            >
              <div className="flex items-center gap-1 mb-2">
                <TrendingUp
                  className="w-3 h-3"
                  style={{ color: 'var(--fxos-gold)' }}
                />
                <span
                  className="text-xs font-medium"
                  style={{ color: 'var(--text-secondary)' }}
                >
                  {ts(gridRisk.leverageInfo, language)}
                </span>
              </div>
              <div className="grid grid-cols-3 gap-1 text-xs">
                <div>
                  <div style={{ color: 'var(--text-secondary)' }}>
                    {ts(gridRisk.currentLeverage, language)}
                  </div>
                  <div
                    className="font-mono"
                    style={{ color: 'var(--text-primary)' }}
                  >
                    {riskInfo.current_leverage}x
                  </div>
                </div>
                <div>
                  <div style={{ color: 'var(--text-secondary)' }}>
                    {ts(gridRisk.effectiveLeverage, language)}
                  </div>
                  <div
                    className="font-mono"
                    style={{ color: 'var(--fxos-gold)' }}
                  >
                    {riskInfo.effective_leverage.toFixed(2)}x
                  </div>
                </div>
                <div>
                  <div style={{ color: 'var(--text-secondary)' }}>
                    {ts(gridRisk.recommendedLeverage, language)}
                  </div>
                  <div
                    className="font-mono"
                    style={{
                      color:
                        riskInfo.current_leverage >
                        riskInfo.recommended_leverage
                          ? 'var(--fxos-danger)'
                          : 'var(--fxos-success)',
                    }}
                  >
                    {riskInfo.recommended_leverage}x
                  </div>
                </div>
              </div>
            </div>

            {/* Position */}
            <div
              className="p-2 rounded"
              style={{ background: 'var(--fxos-bg-lighter)' }}
            >
              <div className="flex items-center gap-1 mb-2">
                <Activity
                  className="w-3 h-3"
                  style={{ color: 'var(--fxos-gold)' }}
                />
                <span
                  className="text-xs font-medium"
                  style={{ color: 'var(--text-secondary)' }}
                >
                  {ts(gridRisk.positionInfo, language)}
                </span>
              </div>
              <div className="grid grid-cols-3 gap-1 text-xs">
                <div>
                  <div style={{ color: 'var(--text-secondary)' }}>
                    {ts(gridRisk.currentPosition, language)}
                  </div>
                  <div
                    className="font-mono"
                    style={{ color: 'var(--text-primary)' }}
                  >
                    {formatUSD(riskInfo.current_position)}
                  </div>
                </div>
                <div>
                  <div style={{ color: 'var(--text-secondary)' }}>
                    {ts(gridRisk.maxPosition, language)}
                  </div>
                  <div
                    className="font-mono"
                    style={{ color: 'var(--text-primary)' }}
                  >
                    {formatUSD(riskInfo.max_position)}
                  </div>
                </div>
                <div>
                  <div style={{ color: 'var(--text-secondary)' }}>
                    {ts(gridRisk.positionPercent, language)}
                  </div>
                  <div
                    className="font-mono"
                    style={{
                      color: getPositionColor(riskInfo.position_percent),
                    }}
                  >
                    {riskInfo.position_percent.toFixed(1)}%
                  </div>
                </div>
              </div>
              {/* Mini progress bar */}
              <div
                className="h-1 mt-2 rounded-full overflow-hidden"
                style={{ background: 'var(--fxos-bg-lighter)' }}
              >
                <div
                  className="h-full rounded-full"
                  style={{
                    width: `${Math.min(riskInfo.position_percent, 100)}%`,
                    background: getPositionColor(riskInfo.position_percent),
                  }}
                />
              </div>
            </div>
          </div>

          {/* Row 2: Market State & Liquidation */}
          <div className="grid grid-cols-2 gap-3">
            {/* Market State */}
            <div
              className="p-2 rounded"
              style={{ background: 'var(--fxos-bg-lighter)' }}
            >
              <div className="flex items-center gap-1 mb-2">
                <Shield
                  className="w-3 h-3"
                  style={{ color: 'var(--fxos-gold)' }}
                />
                <span
                  className="text-xs font-medium"
                  style={{ color: 'var(--text-secondary)' }}
                >
                  {ts(gridRisk.marketState, language)}
                </span>
              </div>
              <div className="grid grid-cols-2 gap-2 text-xs">
                <div>
                  <div style={{ color: 'var(--text-secondary)' }}>
                    {ts(gridRisk.regimeLevel, language)}
                  </div>
                  <div
                    className="font-medium"
                    style={{ color: getRegimeColor(riskInfo.regime_level) }}
                  >
                    {ts(
                      gridRisk[
                        (riskInfo.regime_level ||
                          'standard') as keyof typeof gridRisk
                      ],
                      language
                    )}
                  </div>
                </div>
                <div>
                  <div style={{ color: 'var(--text-secondary)' }}>
                    {ts(gridRisk.currentPrice, language)}
                  </div>
                  <div
                    className="font-mono"
                    style={{ color: 'var(--text-primary)' }}
                  >
                    {formatPrice(riskInfo.current_price)}
                  </div>
                </div>
                <div>
                  <div style={{ color: 'var(--text-secondary)' }}>
                    {ts(gridRisk.breakoutLevel, language)}
                  </div>
                  <div
                    className="font-medium"
                    style={{ color: getBreakoutColor(riskInfo.breakout_level) }}
                  >
                    {ts(
                      gridRisk[
                        (riskInfo.breakout_level ||
                          'none') as keyof typeof gridRisk
                      ],
                      language
                    )}
                  </div>
                </div>
                <div>
                  <div style={{ color: 'var(--text-secondary)' }}>
                    {ts(gridRisk.breakoutDirection, language)}
                  </div>
                  <div
                    className="font-medium"
                    style={{
                      color:
                        riskInfo.breakout_direction === 'up'
                          ? 'var(--fxos-success)'
                          : riskInfo.breakout_direction === 'down'
                            ? 'var(--fxos-danger)'
                            : 'var(--text-secondary)',
                    }}
                  >
                    {riskInfo.breakout_direction
                      ? ts(
                          gridRisk[
                            riskInfo.breakout_direction as keyof typeof gridRisk
                          ],
                          language
                        )
                      : '-'}
                  </div>
                </div>
              </div>
            </div>

            {/* Liquidation */}
            <div
              className="p-2 rounded"
              style={{ background: 'var(--fxos-bg-lighter)' }}
            >
              <div className="flex items-center gap-1 mb-2">
                <AlertTriangle
                  className="w-3 h-3"
                  style={{ color: 'var(--fxos-danger)' }}
                />
                <span
                  className="text-xs font-medium"
                  style={{ color: 'var(--text-secondary)' }}
                >
                  {ts(gridRisk.liquidationInfo, language)}
                </span>
              </div>
              <div className="grid grid-cols-2 gap-2 text-xs">
                <div>
                  <div style={{ color: 'var(--text-secondary)' }}>
                    {ts(gridRisk.liquidationPrice, language)}
                  </div>
                  <div
                    className="font-mono"
                    style={{ color: 'var(--fxos-danger)' }}
                  >
                    {riskInfo.liquidation_price > 0
                      ? formatPrice(riskInfo.liquidation_price)
                      : '-'}
                  </div>
                </div>
                <div>
                  <div style={{ color: 'var(--text-secondary)' }}>
                    {ts(gridRisk.liquidationDistance, language)}
                  </div>
                  <div
                    className="font-mono"
                    style={{ color: 'var(--fxos-danger)' }}
                  >
                    {riskInfo.liquidation_distance > 0
                      ? `${riskInfo.liquidation_distance.toFixed(1)}%`
                      : '-'}
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Row 3: Box State */}
          <div
            className="p-2 rounded"
            style={{ background: 'var(--fxos-bg-lighter)' }}
          >
            <div className="flex items-center gap-1 mb-2">
              <Box className="w-3 h-3" style={{ color: 'var(--fxos-gold)' }} />
              <span
                className="text-xs font-medium"
                style={{ color: 'var(--text-secondary)' }}
              >
                {ts(gridRisk.boxState, language)}
              </span>
            </div>
            <div className="grid grid-cols-3 gap-2 text-xs">
              <div className="flex justify-between">
                <span style={{ color: 'var(--text-secondary)' }}>
                  {ts(gridRisk.shortBox, language)}
                </span>
                <span
                  className="font-mono"
                  style={{ color: 'var(--text-primary)' }}
                >
                  {formatPrice(riskInfo.short_box_lower)} -{' '}
                  {formatPrice(riskInfo.short_box_upper)}
                </span>
              </div>
              <div className="flex justify-between">
                <span style={{ color: 'var(--text-secondary)' }}>
                  {ts(gridRisk.midBox, language)}
                </span>
                <span
                  className="font-mono"
                  style={{ color: 'var(--text-primary)' }}
                >
                  {formatPrice(riskInfo.mid_box_lower)} -{' '}
                  {formatPrice(riskInfo.mid_box_upper)}
                </span>
              </div>
              <div className="flex justify-between">
                <span style={{ color: 'var(--text-secondary)' }}>
                  {ts(gridRisk.longBox, language)}
                </span>
                <span
                  className="font-mono"
                  style={{ color: 'var(--text-primary)' }}
                >
                  {formatPrice(riskInfo.long_box_lower)} -{' '}
                  {formatPrice(riskInfo.long_box_upper)}
                </span>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
