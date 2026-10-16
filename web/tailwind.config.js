/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        aurora: {
          cyan: '#00f5d4',
          purple: '#7b2cbf',
          indigo: '#3a86ff',
          deep: '#0b0914',
          card: 'rgba(23, 16, 42, 0.75)',
        },
        matrix: {
          blue: '#002d50',
          cyan: '#00d2ff',
          gold: '#fbbf24',
          dark: '#001529',
        }
      }
    },
  },
  plugins: [],
}
