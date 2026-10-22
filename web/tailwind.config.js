/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        slate: {
          950: '#11171f', // Soothing warm dark charcoal (replaces harsh #020617 pitch black)
          900: '#151d27', // Card & container base
          850: '#1a2430',
          800: '#222e3d', // Card borders & dividers
          750: '#2a394a',
          700: '#334458', // Muted borders
        },
        cyan: {
          400: '#38bdf8', // Soften harsh neon #22d3ee to calm sky-cyan
          300: '#7dd3fc',
        },
        aurora: {
          cyan: '#2dd4bf',
          purple: '#8b5cf6',
          indigo: '#6366f1',
          deep: '#11171f',
          card: 'rgba(22, 29, 39, 0.85)',
        },
        matrix: {
          blue: '#162232',
          cyan: '#38bdf8',
          gold: '#f59e0b',
          dark: '#11171f',
        }
      }
    },
  },
  plugins: [],
}
