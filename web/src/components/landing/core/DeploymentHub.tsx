import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Terminal, Copy, Check, ChevronRight, Server, Command, Shield } from 'lucide-react'
import { useLanguage } from '../../../contexts/LanguageContext'
import { t } from '../../../i18n/translations'

export default function DeploymentHub() {
    const { language } = useLanguage()
    const [copied, setCopied] = useState(false)
    const installCmd = "curl -fsSL https://raw.githubusercontent.com/onecany/fxos/main/scripts/install.sh | bash"

    const handleCopy = () => {
        navigator.clipboard.writeText(installCmd)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
    }

    return (
        <section className="py-24 bg-fxos-bg relative overflow-hidden border-t border-[var(--panel-border)]">
            {/* Background Grids */}
            <div className="absolute inset-0 bg-[linear-gradient(to_right,#1a181310_1px,transparent_1px),linear-gradient(to_bottom,#1a181310_1px,transparent_1px)] bg-[size:24px_24px]"></div>

            <div className="max-w-7xl mx-auto px-6 relative z-10">
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-16 items-center">

                    {/* Left Column: Context */}
                    <div className="space-y-8">
                        <div className="flex items-center gap-2 text-fxos-gold font-mono text-xs tracking-[0.2em] uppercase">
                            <Server className="w-4 h-4" /> {t('landing.deployEyebrow', language)}
                        </div>

                        <h2 className="text-4xl md:text-6xl font-black text-fxos-text leading-tight">
                            {t('landing.deployTitle1', language)} <span className="text-fxos-gold">{t('landing.deployTitle2', language)}</span>
                        </h2>

                        <p className="text-fxos-text-muted text-lg leading-relaxed font-light">
                            {t('landing.deployDesc', language)}
                        </p>

                        {/* the first five minutes, in plain words */}
                        <ol className="space-y-2 pt-2 font-mono text-sm text-fxos-text-muted">
                            {[
                                t('landing.step1', language),
                                t('landing.step2', language),
                                t('landing.step3', language),
                            ].map((step, i) => (
                                <li key={i} className="flex gap-3">
                                    <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded border border-fxos-gold/30 bg-fxos-gold/10 text-[11px] font-bold text-fxos-gold">
                                        {i + 1}
                                    </span>
                                    <span>{step}</span>
                                </li>
                            ))}
                        </ol>

                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-4">
                            {[
                                { icon: Command, label: t('landing.featureInstallLabel', language), desc: t('landing.featureInstallDesc', language) },
                                { icon: Shield, label: t('landing.featureKeysLabel', language), desc: t('landing.featureKeysDesc', language) }
                            ].map((item, i) => (
                                <div key={i} className="flex gap-4 items-start p-4 rounded bg-fxos-bg-lighter border border-[var(--panel-border)] hover:border-fxos-gold/30 transition-colors group">
                                    <div className="p-2 rounded bg-fxos-bg-deeper border border-[var(--panel-border)] text-fxos-gold group-hover:bg-fxos-gold/10 transition-colors">
                                        <item.icon className="w-5 h-5" />
                                    </div>
                                    <div>
                                        <h4 className="text-fxos-text font-bold font-mono text-sm mb-1">{item.label}</h4>
                                        <p className="text-fxos-text-muted text-xs">{item.desc}</p>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>

                    {/* Right Column: Terminal */}
                    <motion.div
                        initial={{ opacity: 0, x: 50 }}
                        whileInView={{ opacity: 1, x: 0 }}
                        viewport={{ once: true }}
                        className="relative"
                    >
                        {/* Glow effect */}
                        <div className="absolute -inset-1 bg-fxos-gold/10 rounded-xl blur-xl opacity-50"></div>

                        <div className="relative rounded-xl overflow-hidden bg-fxos-bg-lighter border border-[var(--panel-border)] shadow-lg">
                            {/* Terminal Header */}
                            <div className="flex items-center justify-between px-4 py-3 bg-fxos-bg-deeper border-b border-[var(--panel-border)]">
                                <div className="flex gap-2">
                                    <div className="w-3 h-3 rounded-full bg-fxos-danger/80"></div>
                                    <div className="w-3 h-3 rounded-full bg-fxos-gold/80"></div>
                                    <div className="w-3 h-3 rounded-full bg-fxos-success/80"></div>
                                </div>
                                <div className="text-[10px] font-mono text-fxos-text-muted flex items-center gap-1.5">
                                    <Terminal className="w-3 h-3" />
                                    root@fxos-os:~
                                </div>
                            </div>

                            {/* Terminal Content */}
                            <div className="p-8 font-mono text-sm md:text-base bg-fxos-bg-lighter min-h-[200px] flex flex-col justify-center">
                                <div className="mb-2 text-fxos-text-muted text-xs tracking-wide"># Initialize FXOS Core Protocol</div>
                                <div
                                    className="group relative flex items-start gap-3 p-4 rounded-lg bg-fxos-bg-deeper border border-[var(--panel-border)] hover:border-fxos-gold/50 cursor-pointer transition-all hover:bg-fxos-bg"
                                    onClick={handleCopy}
                                >
                                    <span className="text-fxos-gold mt-1"><ChevronRight className="w-4 h-4" /></span>
                                    <code className="text-fxos-text flex-1 break-all">
                                        {installCmd}
                                    </code>

                                    <div className="absolute right-4 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity">
                                        <AnimatePresence mode='wait'>
                                            {copied ? (
                                                <motion.div
                                                    initial={{ scale: 0.5, opacity: 0 }}
                                                    animate={{ scale: 1, opacity: 1 }}
                                                    exit={{ scale: 0.5, opacity: 0 }}
                                                    className="flex items-center gap-1 text-fxos-success bg-fxos-success/10 px-2 py-1 rounded text-xs font-bold"
                                                >
                                                    <Check className="w-3 h-3" />
                                                </motion.div>
                                            ) : (
                                                <div className="text-fxos-text-muted bg-fxos-bg-deeper p-1.5 rounded hover:text-fxos-text hover:bg-fxos-bg">
                                                    <Copy className="w-4 h-4" />
                                                </div>
                                            )}
                                        </AnimatePresence>
                                    </div>
                                </div>
                                <div className="mt-4 flex gap-2">
                                    <div className="w-2 h-4 bg-fxos-gold animate-pulse"></div>
                                </div>
                            </div>
                        </div>
                    </motion.div>
                </div>
            </div>
        </section>
    )
}
