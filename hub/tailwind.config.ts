import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./hooks/**/*.{ts,tsx}",
    "./stores/**/*.{ts,tsx}",
  ],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        surface: {
          DEFAULT: "#FFFFFF",
          secondary: "#F2F2F7",
          tertiary: "#E5E5EA",
        },
        content: {
          primary: "#0A0A0B",
          secondary: "#636366",
          tertiary: "#8E8E93",
        },
        accent: {
          DEFAULT: "#007AFF",
          hover: "#0055CC",
          pressed: "#003399",
          tint: "#E8F0FE",
        },
        status: {
          success: "#34C759",
          warning: "#FF9500",
          error: "#FF3B30",
          info: "#5AC8FA",
        },
      },
      fontFamily: {
        sans: ["Inter", "system-ui", "sans-serif"],
        mono: ["JetBrains Mono", "monospace"],
      },
      fontSize: {
        "headline-1": ["32px", { lineHeight: "40px", fontWeight: "700" }],
        "headline-2": ["24px", { lineHeight: "32px", fontWeight: "600" }],
        "headline-3": ["20px", { lineHeight: "28px", fontWeight: "600" }],
        "headline-4": ["18px", { lineHeight: "24px", fontWeight: "600" }],
        body: ["15px", { lineHeight: "22px", fontWeight: "400" }],
        "body-small": ["13px", { lineHeight: "18px", fontWeight: "400" }],
        caption: ["12px", { lineHeight: "16px", fontWeight: "400" }],
      },
      borderRadius: {
        sm: "6px",
        md: "10px",
        lg: "16px",
      },
      spacing: {
        "4px": "4px",
        "8px": "8px",
        "12px": "12px",
        "16px": "16px",
        "24px": "24px",
        "32px": "32px",
        "48px": "48px",
        "64px": "64px",
      },
    },
  },
  plugins: [],
};

export default config;
