import { useEffect, useState } from 'react'
import { httpClient } from '../../lib/httpClient'
import { apiUrl } from '../../lib/api/helpers'

interface ChartWithOrdersSimpleProps {
  symbol: string
  interval?: string
  traderID?: string
  height?: number
}

export function ChartWithOrdersSimple({
  symbol = 'BTCUSDT',
  interval = '5m',
  traderID,
  height = 500,
}: ChartWithOrdersSimpleProps) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [klineCount, setKlineCount] = useState(0)
  const [orderCount, setOrderCount] = useState(0)

  useEffect(() => {
    const loadData = async () => {
      console.log('[ChartSimple] Loading data for', symbol, interval, 'trader:', traderID)
      setLoading(true)
      setError(null)

      try {
        // Fetch kline data from our own service
        const limit = 100
        const klineUrl = apiUrl(`/klines?symbol=${symbol}&interval=${interval}&limit=${limit}`)

        console.log('[ChartSimple] Fetching klines from our service:', klineUrl)
        const klineResult = await httpClient.request(klineUrl, { silent: true })

        if (!klineResult.success || !klineResult.data) {
          throw new Error('Failed to fetch klines from our service')
        }

        console.log('[ChartSimple] Received klines:', klineResult.data.length)
        setKlineCount(klineResult.data.length)

        // Test fetching order data
        if (traderID) {
          const tradesUrl = apiUrl(`/trades?trader_id=${traderID}&symbol=${symbol}&limit=100`)
          console.log('[ChartSimple] Fetching trades from:', tradesUrl)
          const tradesResult = await httpClient.request(tradesUrl, { silent: true })

          if (tradesResult.success && tradesResult.data) {
            console.log('[ChartSimple] Received trades:', tradesResult.data.length)
            setOrderCount(tradesResult.data.length)
          } else {
            console.warn('[ChartSimple] Failed to fetch trades:', tradesResult.message || 'Unknown error', tradesResult)
          }
        }

        setLoading(false)
      } catch (err: any) {
        console.error('[ChartSimple] Error:', err)
        setError(err.message || 'Failed to load data')
        setLoading(false)
      }
    }

    loadData()
  }, [symbol, interval, traderID])

  return (
    <div className="relative rounded-xl overflow-hidden bg-fxos-bg-deeper" style={{ minHeight: height }}>
      {/* Title bar */}
      <div className="flex items-center justify-between p-4 border-b border-[rgba(45,212,191,0.16)]">
        <div className="flex items-center gap-3">
          <span className="text-xl">📈</span>
          <h3 className="text-lg font-bold text-fxos-text">
            {symbol} {interval} (Test Mode)
          </h3>
        </div>
        {loading && (
          <div className="text-sm text-fxos-text-muted">
            Loading...
          </div>
        )}
      </div>

      {/* Test info */}
      <div className="p-8 space-y-4">
        {error ? (
          <div className="text-center">
            <div className="text-2xl mb-2">⚠️</div>
            <div className="text-fxos-danger">{error}</div>
          </div>
        ) : (
          <>
            <div className="p-4 rounded bg-fxos-bg-lighter border border-[rgba(45,212,191,0.12)]">
              <div className="text-sm mb-2 text-fxos-info">Binance Kline Data</div>
              <div className="text-2xl font-bold text-fxos-success">
                {klineCount} klines
              </div>
            </div>

            {traderID && (
              <div className="p-4 rounded bg-fxos-bg-lighter border border-[rgba(45,212,191,0.12)]">
                <div className="text-sm mb-2 text-fxos-info">Historical Order Data</div>
                <div className="text-2xl font-bold text-fxos-danger">
                  {orderCount} orders
                </div>
              </div>
            )}

            <div className="p-4 rounded bg-fxos-bg-lighter border border-[rgba(45,212,191,0.12)]">
              <div className="text-sm mb-2 text-fxos-info">Status</div>
              <div className="text-lg text-fxos-text">
                ✅ Data fetched successfully, chart component in development
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
