import { motion } from 'framer-motion'
import { useState, useEffect } from 'react'
import { useLanguage } from '../../../contexts/LanguageContext'
import { t } from '../../../i18n/translations'
import type { Language } from '../../../i18n/translations'

interface LogEntry {
    id: number
    time: string
    type: string
    msg: string
    color: string
}

const generateLog = (id: number, language: Language): LogEntry => {
    const types = ['EXEC', 'SIGNAL', 'RISK', 'MACRO', 'SYS']
    const pairs = ['AAPL-USDC', 'NVDA-USDC', 'GOLD-USDC', 'EURUSD-USDC', 'OPENAI-IPO']
    const actions = ['BUY', 'SELL', 'HEDGE', 'ROTATE']
    const type = types[Math.floor(Math.random() * types.length)]

    let msg = ''
    let color = ''

    switch (type) {
        case 'EXEC':
            msg = `AGENT-${Math.floor(Math.random() * 99)} ${actions[Math.floor(Math.random() * 4)]} ${pairs[Math.floor(Math.random() * pairs.length)]} @ ${Math.floor(Math.random() * 600)}`
            color = 'text-fxos-success'
            break;
        case 'SIGNAL':
            msg = t('landing.logSignal', language, { z: (Math.random()).toFixed(3) })
            color = 'text-fxos-gold'
            break;
        case 'RISK':
            msg = t('landing.logRisk', language, { pair: pairs[Math.floor(Math.random() * pairs.length)] })
            color = 'text-fxos-danger'
            break;
        case 'MACRO':
            msg = t('landing.logMacro', language, { ms: Math.floor(Math.random() * 10) })
            color = 'text-fxos-text-muted'
            break;
        default:
            msg = t('landing.logSys', language)
            color = 'text-fxos-accent'
    }

    return { id, time: new Date().toLocaleTimeString('en-US', { hour12: false }) + '.' + Math.floor(Math.random() * 999), type, msg, color }
}

export default function LiveFeed() {
    const { language } = useLanguage()
    const [logs, setLogs] = useState<LogEntry[]>([])

    useEffect(() => {
        // Initial population
        const initialLogs = Array.from({ length: 8 }).map((_, i) => generateLog(i, language))
        setLogs(initialLogs)

        const interval = setInterval(() => {
            setLogs(prev => {
                const newLog = generateLog(Date.now(), language)
                return [newLog, ...prev.slice(0, 7)]
            })
        }, 800) // Fast 800ms updates for HFT feel

        return () => clearInterval(interval)
    }, [])

    return (
        <section className="w-full bg-fxos-bg-lighter border-y border-[var(--panel-border)] py-1 overflow-hidden relative">

            <div className="max-w-[1920px] mx-auto px-4 flex flex-col md:flex-row gap-0 md:gap-8 items-stretch h-[240px] md:h-12 text-xs font-mono">

                {/* Left Status Bar (Static) */}
                <div className="hidden md:flex items-center gap-6 text-fxos-text-muted border-r border-[var(--panel-border)] pr-6 shrink-0">
                    <div className="flex items-center gap-2">
                        <div className="w-1.5 h-1.5 bg-fxos-success rounded-full animate-pulse"></div>
                        <span className="font-bold text-fxos-text">{t('landing.feedStable', language)}</span>
                    </div>
                    <div className="flex items-center gap-2">
                        <span className="text-fxos-gold">TPS: 48,291</span>
                    </div>
                </div>

                {/* Right Scrolling Log - Vertical on mobile, Single line ticker on Desktop */}
                <div className="flex-1 overflow-hidden relative font-mono text-[10px] md:text-sm h-full flex items-center">

                    {/* Desktop View: Single Line Fade */}
                    <div className="hidden md:block w-full h-full relative">
                        {logs.slice(0, 1).map((log) => (
                            <motion.div
                                key={log.id}
                                initial={{ opacity: 0, x: -20 }}
                                animate={{ opacity: 1, x: 0 }}
                                className="absolute inset-0 flex items-center gap-4"
                            >
                                <span className="text-fxos-text-muted">[{log.time}]</span>
                                <span className={`font-bold w-10 ${log.type === 'RISK' ? 'text-fxos-danger bg-fxos-danger/10 px-1 rounded' :
                                    log.type === 'SIGNAL' ? 'text-fxos-gold bg-fxos-gold/10 px-1 rounded' :
                                        log.type === 'EXEC' ? 'text-fxos-success' : 'text-fxos-text-muted'
                                    }`}>{log.type}</span>
                                <span className={`${log.color}`}>{log.msg}</span>
                            </motion.div>
                        ))}
                    </div>

                    {/* Mobile View: Vertical Stack */}
                    <div className="md:hidden flex flex-col gap-2 w-full p-4 h-full overflow-hidden">
                        {logs.map((log) => (
                            <div key={log.id} className="flex gap-2 w-full truncate border-b border-[var(--panel-border)] pb-1 last:border-0">
                                <span className="text-fxos-text-muted w-16 shrink-0">{log.time.split('.')[0]}</span>
                                <span className={`font-bold w-8 shrink-0 ${log.type === 'RISK' ? 'text-fxos-danger' :
                                    log.type === 'SIGNAL' ? 'text-fxos-gold' :
                                        'text-fxos-text-muted'
                                    }`}>{log.type}</span>
                                <span className={`${log.color} truncate`}>{log.msg}</span>
                            </div>
                        ))}
                    </div>

                </div>

            </div>
        </section>
    )
}
