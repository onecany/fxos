import React from 'react'

interface DeepVoidBackgroundProps extends React.HTMLAttributes<HTMLDivElement> {
  children?: React.ReactNode
  className?: string
  disableAnimation?: boolean
}

export function DeepVoidBackground({
  children,
  className = '',
  disableAnimation = false,
  ...props
}: DeepVoidBackgroundProps) {
  return (
    <div
      className={`relative w-full min-h-screen bg-fxos-bg text-fxos-text overflow-hidden flex flex-col ${className}`}
      {...props}
    >
      {disableAnimation ? (
        <>
          <div className="absolute inset-0 pointer-events-none z-0 bg-fxos-bg"></div>
          <div className="absolute inset-0 pointer-events-none z-0 opacity-[0.08] bg-[linear-gradient(to_right,rgba(45,212,191,0.12)_1px,transparent_1px),linear-gradient(to_bottom,rgba(45,212,191,0.12)_1px,transparent_1px)] bg-[size:32px_32px]"></div>
        </>
      ) : (
        <>
          <div className="absolute inset-0 pointer-events-none z-0 bg-[radial-gradient(circle_at_20%_20%,rgba(45,212,191,0.07),transparent_20%),radial-gradient(circle_at_80%_10%,rgba(125,211,252,0.05),transparent_24%),radial-gradient(circle_at_50%_80%,rgba(255,255,255,0.03),transparent_28%),linear-gradient(180deg,rgba(255,255,255,0.02),transparent_30%),var(--fxos-bg)] bg-cover"></div>
          <div className="absolute inset-0 pointer-events-none z-0 opacity-70 bg-[linear-gradient(90deg,rgba(255,255,255,0.02)_1px,transparent_1px),linear-gradient(rgba(255,255,255,0.02)_1px,transparent_1px)] bg-[size:40px_40px]"></div>
          <div className="absolute inset-0 pointer-events-none z-0">
            <div
              className="absolute inset-x-0 bottom-0 h-[55vh] bg-[linear-gradient(to_right,rgba(45,212,191,0.10)_1px,transparent_1px),linear-gradient(to_bottom,rgba(45,212,191,0.10)_1px,transparent_1px)] bg-[size:40px_40px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)] opacity-50"
              style={{
                transform:
                  'perspective(500px) rotateX(62deg) translateY(100px) scale(2)',
              }}
            ></div>
          </div>
        </>
      )}

      <div className="relative z-10 flex-1 flex flex-col h-full w-full">
        {children}
      </div>
    </div>
  )
}
