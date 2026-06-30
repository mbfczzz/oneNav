/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        // Primary blue (#3b82f6 = Tailwind blue-500) + teal drag accents from the original.
        primary: {
          DEFAULT: '#3b82f6',
          50: '#eff6ff',
          100: '#dbeafe',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
        },
        teal: {
          ghost: '#f4fbfe',
          chosen: '#f8fcff',
          accent: '#2d95bd',
        },
      },
      boxShadow: {
        card: '0 10px 24px rgba(45,149,189,0.14)',
        'card-hover': '0 14px 30px rgba(45,149,189,0.20)',
      },
    },
  },
  plugins: [],
}
