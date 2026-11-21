/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        shield: {
          bg: '#0b1220',
          panel: '#111a2e',
          card: '#16213a',
          border: '#243252',
          accent: '#22d3ee',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'Segoe UI', 'Noto Sans Sinhala', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
