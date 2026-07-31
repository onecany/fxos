import { motion, AnimatePresence } from 'framer-motion'
import { LogIn, UserPlus, X, AlertTriangle, Terminal } from 'lucide-react'
import { Link } from 'react-router-dom'
import { DeepVoidBackground } from '../common/DeepVoidBackground'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'

interface LoginRequiredOverlayProps {
  isOpen: boolean
  onClose: () => void
  featureName?: string
}

export function LoginRequiredOverlay({
  isOpen,
  onClose,
  featureName,
}: LoginRequiredOverlayProps) {
  const { language } = useLanguage()

  const tr = (key: string, params?: Record<string, string | number>) =>
    t(`loginRequired.${key}`, language, params)

  const subtitle = featureName
    ? tr('subtitleWithFeature', { featureName })
    : tr('subtitleDefault')

  const benefits = [tr('benefit1'), tr('benefit2'), tr('benefit4')]

  return (
    <AnimatePresence>
      {isOpen && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 z-50"
        >
          <DeepVoidBackground
            className="w-full h-full bg-fxos-bg/95 backdrop-blur-md flex items-center justify-center p-4 text-fxos-text"
            disableAnimation
            onClick={onClose}
          >
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: 10 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.95, y: 10 }}
              transition={{ type: 'spring', damping: 20, stiffness: 300 }}
              className="relative max-w-md w-full overflow-hidden bg-fxos-bg-lighter border border-fxos-gold/30 shadow-lg rounded-sm group font-mono"
              onClick={(e) => e.stopPropagation()}
            >
              {/* Terminal Window Header */}
              <div className="flex items-center justify-between px-3 py-2 bg-fxos-bg-deeper border-b border-fxos-gold/20">
                <div className="flex items-center gap-2">
                  <Terminal size={12} className="text-fxos-gold" />
                  <span className="text-[10px] text-fxos-text-muted uppercase tracking-wider">
                    auth_protocol.exe
                  </span>
                </div>
                <button
                  onClick={onClose}
                  className="text-fxos-text-muted hover:text-fxos-danger transition-colors"
                >
                  <X size={14} />
                </button>
              </div>

              {/* Main Content */}
              <div className="p-8 relative">
                {/* Background Grid */}
                <div className="absolute inset-0 bg-[linear-gradient(to_right,#80808008_1px,transparent_1px),linear-gradient(to_bottom,#80808008_1px,transparent_1px)] bg-[size:14px_14px] pointer-events-none"></div>

                <div className="relative z-10">
                  {/* Flashing Access Denied */}
                  <div className="flex justify-center mb-6">
                    <div className="relative">
                      <div className="bg-fxos-bg-lighter border border-fxos-danger/50 text-fxos-danger px-4 py-2 flex items-center gap-3">
                        <AlertTriangle size={18} className="animate-pulse" />
                        <span className="font-bold tracking-widest text-sm uppercase">
                          {tr('accessDenied')}
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Terminal Text */}
                  <div className="space-y-4 mb-8">
                    <div className="text-center">
                      <h2 className="text-xl font-bold text-fxos-text uppercase tracking-wider mb-2">
                        {tr('title')}
                      </h2>
                      <p className="text-fxos-gold text-xs uppercase tracking-widest border-b border-fxos-gold/20 pb-4 inline-block">
                        {subtitle}
                      </p>
                    </div>

                    <div className="bg-fxos-bg-deeper border-l-2 border-fxos-gold/20 p-3 my-4">
                      <p className="text-xs text-fxos-text-muted leading-relaxed font-mono">
                        <span className="text-fxos-success mr-2">$</span>
                        {tr('description')}
                      </p>
                    </div>

                    <div className="grid grid-cols-2 gap-2">
                      {benefits.map((benefit, i) => (
                        <div
                          key={i}
                          className="flex items-center gap-2 text-[10px] text-fxos-text-muted uppercase tracking-wide"
                        >
                          <span className="text-fxos-gold">✓</span> {benefit}
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Action Buttons */}
                  <div className="space-y-3">
                    <Link
                      to="/login"
                      className="flex items-center justify-center gap-2 w-full py-3 bg-fxos-gold text-fxos-bg font-bold text-xs uppercase tracking-widest hover:bg-fxos-gold-highlight transition-all group"
                    >
                      <LogIn size={14} />
                      <span>{tr('loginButton')}</span>
                      <span className="opacity-0 group-hover:opacity-100 transition-opacity -ml-2 group-hover:ml-0">
                        -&gt;
                      </span>
                    </Link>

                    <Link
                      to="/register"
                      className="flex items-center justify-center gap-2 w-full py-3 bg-transparent border border-fxos-gold/20 text-fxos-text-muted hover:text-fxos-text hover:border-fxos-gold font-bold text-xs uppercase tracking-widest transition-all hover:bg-fxos-gold/10"
                    >
                      <UserPlus size={14} />
                      <span>{tr('registerButton')}</span>
                    </Link>
                  </div>

                  <div className="mt-4 text-center">
                    <button
                      onClick={onClose}
                      className="text-[10px] text-fxos-text-muted hover:text-fxos-danger uppercase tracking-widest hover:underline decoration-fxos-danger/30"
                    >
                      [ {tr('abort')} ]
                    </button>
                  </div>
                </div>
              </div>

              {/* Corner Accents */}
              <div className="absolute top-0 right-0 w-2 h-2 border-t border-r border-fxos-gold"></div>
              <div className="absolute bottom-0 left-0 w-2 h-2 border-b border-l border-fxos-gold"></div>
            </motion.div>
          </DeepVoidBackground>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
