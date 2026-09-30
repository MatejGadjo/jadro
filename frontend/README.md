# Jadro frontend

React 19 + Vite SPA for [Jadro](../README.md). It talks to the Go API over `/api/*`.

```bash
cp .env.example .env     # VITE_API_BASE_URL=http://localhost:8080 for local dev
npm install
npm run dev              # http://localhost:5173
npm test                 # Vitest
npm run lint             # ESLint
npm run build            # type-check + production build to dist/
ANALYZE=1 npm run build  # also writes stats.html (bundle report)
```

Structure: `src/api/` (axios + zod schemas), `src/hooks/` (TanStack Query), `src/pages/`, `src/components/`, `src/lib/` (pure helpers, unit-tested). Conventions and product constraints are in [AGENTS.md](AGENTS.md).

In Docker the app is built with an empty `VITE_API_BASE_URL` (same origin) and served by nginx. See the root [README](../README.md).
