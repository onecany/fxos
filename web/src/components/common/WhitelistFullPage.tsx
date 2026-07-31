import { motion } from 'framer-motion'
import { ShieldAlert, ArrowLeft, Send, Lock } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { OFFICIAL_LINKS } from '../../constants/branding'

interface WhitelistFullPageProps {
  onBack?: () => void
}

export function WhitelistFullPage({ onBack }: WhitelistFullPageProps) {
  const navigate = useNavigate()

  const handleBackToLogin = () => {
    if (onBack) {
      onBack()
    } else {
      navigate('/login')
    }
  }

  return (
    <div className="min-h-screen bg-fxos-bg-deeper text-fxos-text font-mono relative overflow-hidden flex items-center justify-center px-4">
      {/* Background Grid */}
      <div className="fixed inset-0 bg-[linear-gradient(to_right,rgba(26,24,19,0.04)_1px,transparent_1px),linear-gradient(to_bottom,rgba(26,24,19,0.04)_1px,transparent_1px)] bg-[size:24px_24px] pointer-events-none"></div>

      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.5 }}
        className="max-w-lg w-full relative z-10"
      >
        <div className="bg-fxos-bg-lighter border border-fxos-danger/30 rounded-lg overflow-hidden relative group">
          {/* Top Bar */}
          <div className="flex items-center justify-between px-4 py-2 bg-fxos-danger/10 border-b border-fxos-danger/30">
            <div className="flex gap-1.5 opacity-50">
              <div className="w-2.5 h-2.5 rounded-full bg-fxos-danger"></div>
              <div className="w-2.5 h-2.5 rounded-full bg-[var(--panel-border)]"></div>
              <div className="w-2.5 h-2.5 rounded-full bg-[var(--panel-border)]"></div>
            </div>
            <div className="text-[10px] text-fxos-danger font-mono tracking-widest animate-pulse">
              ACCESS_DENIED // ERROR_403
            </div>
          </div>

          <div className="p-8 text-center">
            {/* Icon */}
            <div className="relative mx-auto mb-8 w-20 h-20 flex items-center justify-center">
              <div className="relative z-10 p-4 border-2 border-fxos-danger/50 rounded-full bg-fxos-bg-deeper">
                <ShieldAlert className="w-8 h-8 text-fxos-danger" />
              </div>
            </div>

            {/* Title */}
            <h1 className="text-2xl font-bold mb-2 tracking-widest text-fxos-text uppercase">
              <span className="text-fxos-danger">RESTRICTED</span> ACCESS
            </h1>

            <div className="h-[1px] w-full bg-gradient-to-r from-transparent via-fxos-danger/40 to-transparent my-4"></div>

            {/* Description */}
            <p className="text-xs text-fxos-text-muted mb-8 leading-relaxed font-mono px-4">
              <span className="text-fxos-danger">[SYSTEM_MESSAGE]:</span> YOUR
              IDENTIFIER IS NOT ON THE ACTIVE WHITELIST.
              <br />
              <br />
              Platform capacity limits have been reached for the current beta
              phase. Prioritized access is currently reserved for authorized
              operators only.
            </p>

            {/* Info Box */}
            <div className="bg-fxos-danger/8 border border-fxos-danger/25 p-4 rounded mb-8 text-left">
              <div className="flex items-start gap-3">
                <Lock className="w-4 h-4 text-fxos-danger mt-0.5" />
                <div>
                  <h3 className="text-xs font-bold text-fxos-danger uppercase mb-1">
                    Authorization Protocol
                  </h3>
                  <p className="text-[10px] text-fxos-text-muted leading-tight">
                    Access is rolled out in batches. If you believe this is an
                    error, please verify your credentials or contact system
                    administrators.
                  </p>
                </div>
              </div>
            </div>

            {/* Action Buttons */}
            <div className="space-y-3">
              <button
                onClick={handleBackToLogin}
                className="w-full flex items-center justify-center gap-2 py-3 border border-[var(--panel-border)] bg-fxos-bg hover:bg-fxos-bg-deeper hover:border-fxos-danger hover:text-fxos-danger text-fxos-text-muted transition-all text-xs font-bold tracking-widest uppercase group"
              >
                <ArrowLeft className="w-3 h-3 group-hover:-translate-x-1 transition-transform" />
                RETURN TO LOGIN
              </button>

              <div className="grid grid-cols-2 gap-3 mt-4">
                <a
                  href={OFFICIAL_LINKS.telegram}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center justify-center gap-2 py-2 border border-[var(--panel-border)] bg-fxos-bg hover:bg-fxos-bg-deeper text-fxos-text-muted hover:text-fxos-text transition-colors text-[10px] uppercase"
                >
                  <Send className="w-3 h-3" />
                  Support
                </a>
              </div>
            </div>
          </div>

          {/* Footer */}
          <div className="bg-fxos-bg-deeper p-2 text-[9px] text-fxos-text-muted text-center border-t border-[var(--panel-border)] font-mono uppercase">
            ERR_CODE: WLIST_0x403 // SECURITY_LAYER_ACTIVE
          </div>
        </div>
      </motion.div>
    </div>
  )
}
