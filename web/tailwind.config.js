/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Sci-fi neon palette — driven by CSS variables so the whole UI
        // flips between dark and light via [data-theme] on <html>.
        // RGB-triplet pattern keeps tailwind opacity modifiers (/20 etc.)
        // working: they compile to rgb(var(--x-rgb) / 0.2).
        'fxos-gold': {
          DEFAULT: 'rgb(var(--fxos-gold-rgb) / <alpha-value>)',
          dim: 'var(--fxos-gold-dim)',
          glow: 'var(--fxos-gold-glow)',
          highlight: 'var(--fxos-gold-highlight)',
        },
        'fxos-bg': {
          DEFAULT: 'rgb(var(--fxos-bg-rgb) / <alpha-value>)',
          deeper: 'rgb(var(--fxos-bg-deeper-rgb) / <alpha-value>)',
          lighter: 'rgb(var(--fxos-bg-lighter-rgb) / <alpha-value>)',
        },
        'fxos-accent': 'rgb(var(--fxos-accent-rgb) / <alpha-value>)',
        'fxos-text': {
          DEFAULT: 'rgb(var(--fxos-text-rgb) / <alpha-value>)',
          main: 'rgb(var(--fxos-text-main-rgb) / <alpha-value>)',
          muted: 'rgb(var(--fxos-text-muted-rgb) / <alpha-value>)',
        },
        'fxos-success': '#22c55e', // neon green
        'fxos-danger': '#fb7185',  // neon red
      },
      fontFamily: {
        sans: ['IBM Plex Mono', 'ui-monospace', 'Menlo', 'monospace'],
        mono: ['IBM Plex Mono', 'Menlo', 'Monaco', 'Courier New', 'monospace'],
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(circle at center, var(--tw-gradient-stops))',
        'gradient-conic': 'conic-gradient(from 180deg at 50% 50%, var(--tw-gradient-stops))',
        'scanlines': "url(\"data:image/svg+xml,%3Csvg width='4' height='4' viewBox='0 0 4 4' fill='none' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M0 0H4V2H0V0Z' fill='rgba(0,0,0,0.4)'/%3E%3C/svg%3E\")",
        'grid-pattern': "linear-gradient(to right, #1f2937 1px, transparent 1px), linear-gradient(to bottom, #1f2937 1px, transparent 1px)",
      },
      animation: {
        'pulse-slow': 'pulse 4s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'scan': 'scan 8s linear infinite',
        'scan-fast': 'scan 2s linear infinite',
        'float': 'float 6s ease-in-out infinite',
        'glitch': 'glitch 0.3s cubic-bezier(.25, .46, .45, .94) both infinite',
        'shimmer': 'shimmer 2s linear infinite',
      },
      keyframes: {
        scan: {
          '0%': { backgroundPosition: '0 0' },
          '100%': { backgroundPosition: '0 100%' },
        },
        float: {
          '0%, 100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-10px)' },
        },
        glitch: {
          '0%': { transform: 'translate(0)' },
          '20%': { transform: 'translate(-2px, 2px)' },
          '40%': { transform: 'translate(-2px, -2px)' },
          '60%': { transform: 'translate(2px, 2px)' },
          '80%': { transform: 'translate(2px, -2px)' },
          '100%': { transform: 'translate(0)' },
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
      },
      boxShadow: {
        'neon': '0 0 5px var(--fxos-gold), 0 0 20px var(--fxos-gold-dim)',
        'neon-blue': '0 0 5px var(--fxos-accent), 0 0 20px rgba(0, 240, 255, 0.2)',
      },
    },
  },
  plugins: [],
}
