/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          orange: '#f97316',
          orangeHover: '#ea580c',
          green: '#22c55e',
          greenHover: '#16a34a',
        }
      }
    },
  },
  plugins: [],
}
