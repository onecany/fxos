import { motion } from 'framer-motion'
import { TrendingUp, Layers, Zap, Hexagon, Crosshair } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../../contexts/AuthContext'
import { useLanguage } from '../../../contexts/LanguageContext'
import { t } from '../../../i18n/translations'

const traderPresets = [
  {
    name: 'ALPHA-1',
    class: 'US_STOCKS',
    descKey: 'landing.agent1Desc',
    apy: '142%',
    winRate: '68%',
    riskKey: 'landing.riskHigh',
    color: 'text-fxos-gold',
    border: 'border-fxos-gold/50',
    bg_glow: 'shadow-sm',
    icon: Zap,
  },
  {
    name: 'BETA-X',
    class: 'MACRO_FX',
    descKey: 'landing.agent2Desc',
    apy: '89%',
    winRate: '55%',
    riskKey: 'landing.riskMed',
    color: 'text-fxos-accent',
    border: 'border-fxos-accent/30',
    bg_glow: 'shadow-sm',
    icon: TrendingUp,
  },
  {
    name: 'GAMMA-RAY',
    class: 'PRE_IPO',
    descKey: 'landing.agent3Desc',
    apy: '24%',
    winRate: '99%',
    riskKey: 'landing.riskLow',
    color: 'text-fxos-text',
    border: 'border-fxos-gold/20',
    bg_glow: 'shadow-sm',
    icon: Layers,
  },
]

export default function AgentGrid() {
  const { language } = useLanguage()
  const { user } = useAuth()
  const navigate = useNavigate()

  const handleInitialize = () => {
    if (user) {
      navigate('/strategy')
    } else {
      navigate('/login')
    }
  }

  return (
    <section
      id="market-scanner"
      className="py-16 md:py-24 bg-fxos-bg relative overflow-hidden"
    >
      {/* Background Details */}
      <div className="absolute top-0 right-0 p-10 opacity-10 pointer-events-none">
        <Hexagon className="w-64 h-64 text-fxos-text-muted" strokeWidth={0.5} />
      </div>

      <div className="max-w-7xl mx-auto px-6 relative z-10">
        <div className="flex flex-col md:flex-row justify-between items-end mb-10 md:mb-16 gap-6">
          <div>
            <div className="flex items-center gap-2 text-fxos-gold font-mono text-xs mb-2 tracking-widest uppercase">
              <Crosshair className="w-4 h-4" />{' '}
              {t('landing.assetClassSelect', language)}
            </div>
            <h2 className="text-4xl md:text-5xl font-black text-fxos-text uppercase tracking-tighter">
              {t('landing.proTraders1', language)}{' '}
              <span className="text-fxos-gold">
                {t('landing.proTraders2', language)}
              </span>
            </h2>
          </div>
          <div className="font-mono text-right text-xs text-fxos-text-muted max-w-xs">
            {t('landing.agentTagline', language)}
          </div>
        </div>

        {/* Grid Container - Removing scroll tracking for stability test */}
        <div className="flex flex-row md:grid md:grid-cols-3 gap-4 md:gap-8 overflow-x-auto md:overflow-visible pb-12 md:pb-0 snap-x snap-mandatory -mx-6 px-6 md:mx-0 md:px-0 scrollbar-hide">
          {traderPresets.map((preset, i) => {
            const Icon = preset.icon

            return (
              <motion.div
                key={i}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                transition={{ delay: i * 0.1 }}
                className={`group relative bg-fxos-bg-lighter backdrop-blur-xl border ${preset.border} overflow-hidden transition-all duration-300 min-w-[85vw] md:min-w-0 snap-center shrink-0 rounded-xl md:rounded-none`}
              >
                {/* Top "Hinge" decoration */}
                <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-transparent via-fxos-text/10 to-transparent"></div>

                <div className="p-8 relative z-10">
                  {/* Header */}
                  <div className="flex justify-between items-start mb-6">
                    <div className="p-3 bg-fxos-bg-deeper rounded border border-[var(--panel-border)]">
                      <Icon className={`w-8 h-8 ${preset.color}`} />
                    </div>
                    <div className="text-right">
                      <div className="text-[10px] font-mono text-fxos-text-muted uppercase">
                        {t('landing.classLabel', language)}
                      </div>
                      <div
                        className={`font-bold font-mono tracking-wider ${preset.color}`}
                      >
                        {preset.class}
                      </div>
                    </div>
                  </div>

                  {/* Name & Desc */}
                  <h3 className="text-3xl font-bold text-fxos-text mb-2 tracking-tight group-hover:text-fxos-accent transition-colors">
                    {preset.name}
                  </h3>
                  <p className="text-fxos-text-muted text-sm mb-8 leading-relaxed h-10">
                    {t(preset.descKey, language)}
                  </p>

                  {/* Stats Grid */}
                  <div className="grid grid-cols-3 gap-px bg-[var(--panel-border)] border border-[var(--panel-border)] rounded overflow-hidden mb-8">
                    <div className="bg-fxos-bg-deeper p-3 text-center group-hover:bg-fxos-bg transition-colors">
                      <div className="text-[10px] text-fxos-text-muted uppercase font-mono mb-1">
                        APY
                      </div>
                      <div className="text-fxos-success font-bold">
                        {preset.apy}
                      </div>
                    </div>
                    <div className="bg-fxos-bg-deeper p-3 text-center group-hover:bg-fxos-bg transition-colors">
                      <div className="text-[10px] text-fxos-text-muted uppercase font-mono mb-1">
                        {t('landing.winLabel', language)}
                      </div>
                      <div className="text-fxos-text font-bold">
                        {preset.winRate}
                      </div>
                    </div>
                    <div className="bg-fxos-bg-deeper p-3 text-center group-hover:bg-fxos-bg transition-colors">
                      <div className="text-[10px] text-fxos-text-muted uppercase font-mono mb-1">
                        {t('landing.riskLabel', language)}
                      </div>
                      <div className={`${preset.color} font-bold`}>
                        {t(preset.riskKey, language)}
                      </div>
                    </div>
                  </div>

                  {/* Action Btn */}
                  <button
                    onClick={handleInitialize}
                    className={`w-full py-4 text-xs font-bold font-mono uppercase tracking-[0.2em] border border-[var(--panel-border)] hover:border-${preset.color === 'text-fxos-gold' ? 'fxos-gold' : 'fxos-text'} hover:bg-fxos-text/5 transition-all flex items-center justify-center gap-2 group-hover:text-fxos-text cursor-pointer text-fxos-text`}
                  >
                    <span className={preset.color}>[</span>{' '}
                    {t('landing.initialize', language)}{' '}
                    <span className={preset.color}>]</span>
                  </button>
                </div>

                {/* Decorative Background Elements */}
                <div className="absolute -right-10 -bottom-10 w-40 h-40 bg-gradient-to-br from-fxos-text/5 to-transparent rounded-full blur-2xl group-hover:opacity-50 transition-opacity opacity-20"></div>
              </motion.div>
            )
          })}
        </div>
      </div>
    </section>
  )
}
