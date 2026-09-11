import nextCoreWebVitals from "eslint-config-next/core-web-vitals";

/** @type {import("eslint").Linter.Config[]} */
const eslintConfig = [
  ...nextCoreWebVitals,
  {
    rules: {
      // React 19 / Hooks v7 — barra endurecida tras `frontend-eslint-warnings-cleanup` (openspec).
      "react-hooks/set-state-in-effect": "error",
      "react-hooks/immutability": "error",
      "react-hooks/purity": "error",
    },
  },
  {
    ignores: [
      "node_modules/**",
      ".next/**",
      "out/**",
      "build/**",
      "coverage/**",
      "next-env.d.ts",
    ],
  },
];

export default eslintConfig;
