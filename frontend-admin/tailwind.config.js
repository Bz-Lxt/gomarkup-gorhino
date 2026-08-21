/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js}'],
  theme: {
    extend: {
      colors: {
        bg: '#07090c',
        panel: '#10141b',
        inset: '#0b0e13',
        line: '#243044',
        ink: '#e8edf5',
        muted: '#8b9bb4',
        amber: '#f6a53b',
        phosphor: '#7cffb2',
        cyan: '#5ec8ff',
        danger: '#ff5a5a',
        warn: '#ffd166',
      },
      fontFamily: {
        ui: ['"Noto Sans SC"', 'Barlow', 'sans-serif'],
        display: ['Oxanium', 'monospace'],
        mono: ['"IBM Plex Mono"', 'ui-monospace', 'monospace'],
      },
    },
  },
  plugins: [],
}
