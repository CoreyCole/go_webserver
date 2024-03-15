/*eslint-env node*/

/**
 * @type {import('tailwindcss').Config}
 */
module.exports = {
  darkMode: "class",
  content: [
    "./public/**/*.html",
    "./webserver/**/*.{go,templ,html}",
    "./react/**/*.{js,jsx,ts,tsx}",
  ],
  safelist: [],
  plugins: [require("daisyui")],
  daisyui: {
    themes: ["dark"],
  },
}
