/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // "Night" theme — battery-friendly dark palette
        night: {
          50: "#D2E8DE",   // primary text
          200: "#A8C4B8",  // secondary text
          400: "#7C918A",  // muted text
          500: "#5C716A",  // faint labels
          700: "#2A3A34",  // strong border
          800: "#1E2A26",  // border
          850: "#1A2420",  // card surface
          900: "#111816",  // content background
          950: "#0B100E",  // sidebar background
        },
        mint: {
          DEFAULT: "#4ADE80", // primary action
          soft: "#A7F3D0",    // highlighted text
          deep: "#166B4A",    // user chat bubble
          ink: "#052E16",     // text on mint
        },
      },
    },
  },
  plugins: [],
}
