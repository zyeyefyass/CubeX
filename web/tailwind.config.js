/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        windhawk: {
          bg: '#0c0d10',
          panel: '#121317',
          card: '#16171d',
          cardHover: '#1c1e26',
          border: '#27282e',
          borderLight: '#383a44',
          blue: '#0078d4',
          blueHover: '#0067c0',
          blueLight: '#4cc2ff',
          coral: '#f43f5e',
          pill: '#202126',
          pillBorder: '#35363f',
          textMuted: '#9a9da6',
        },
        slate: {
          950: '#0c0d10', // Deep obsidian dark from screenshot
          900: '#121317', // Windhawk container & panel base
          850: '#16171d', // Windhawk card base
          800: '#20222a', // Windhawk pill & component borders
          750: '#27282e', // Table grid lines
          700: '#33353e', // Hover borders
          600: '#484a56',
        },
        cyan: {
          400: '#0078d4', // Map primary cyan to Windhawk Cobalt Blue
          300: '#4cc2ff',
        },
        aurora: {
          cyan: '#0078d4',
          purple: '#0078d4',
          indigo: '#0078d4',
          deep: '#0c0d10',
          card: 'rgba(22, 23, 29, 0.85)',
        },
        matrix: {
          blue: '#16171d',
          cyan: '#0078d4',
          gold: '#f59e0b',
          dark: '#0c0d10',
        }
      }
    },
  },
  plugins: [],
}
