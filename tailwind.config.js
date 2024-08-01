/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./cmd/**/templates/**/*.templ"],
  theme: {
    extend: {},
  },
  plugins: [require("daisyui")],
}
