import type { Theme } from '../contexts/ThemeContext'

export interface ChartPalette {
  background: string
  textColor: string
  gridColor: string
  borderColor: string
  crosshairLabel: string
}

const DARK_PALETTE: ChartPalette = {
  background: '#0b1220',
  textColor: '#d8e6f5',
  gridColor: 'rgba(45, 212, 191, 0.08)',
  borderColor: 'rgba(45, 212, 191, 0.14)',
  crosshairLabel: '#2dd4bf',
}

const LIGHT_PALETTE: ChartPalette = {
  background: '#ffffff',
  textColor: '#1a2c40',
  gridColor: 'rgba(13, 148, 136, 0.10)',
  borderColor: 'rgba(13, 148, 136, 0.22)',
  crosshairLabel: '#0d9488',
}

// Canvas-based charts (lightweight-charts, TradingView widget) cannot read
// CSS variables, so resolve their palette per theme here.
export function getChartPalette(theme: Theme): ChartPalette {
  return theme === 'dark' ? DARK_PALETTE : LIGHT_PALETTE
}
