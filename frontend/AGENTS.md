# Jadro frontend: guide for AI coding agents

## Commands
- `npm run dev` — start Vite dev server
- `npm run build` — type-check + build to `dist/`
- `npm run lint` — run ESLint
- `npm run typecheck` — TypeScript only
- `npm test` — Vitest unit tests (`src/**/__tests__/`)
- `npm run preview` — preview production build locally

## Tech stack
- Vite + React + TypeScript (strict)
- Tailwind CSS + shadcn/ui
- TanStack Router + TanStack Query
- axios + zod
- lucide-react + date-fns
- clsx + tailwind-merge

## MVP product constraints (locked)
- Feed is cluster-based, not article-based (articles only on cluster detail).
- No login, no personalization, no infinite scroll.
- No client-side clustering or ranking logic (backend is authoritative).
- Trust > CTR, readability > density, consistency > experiments.

## API contract
- Backend base URL set via `VITE_API_BASE_URL` (default: `http://localhost:8080`). No Vite proxy.
- `GET /api/feed?topic=&page=` — used by `useFeed`; returns `FeedResponse`.
- `GET /api/clusters/:id` — used by `useCluster`; returns `ClusterResponse`.
- Zod schemas in `src/api/schemas.ts`; TypeScript types inferred in `src/api/types.ts`.
- Query key factory in `src/api/queryKeys.ts`.

## Architecture rules
- All API calls go through `src/api/` and validate with zod.
- API functions return typed data only, never raw axios responses.
- Pages do not fetch data directly; they use hooks in `src/hooks/`.
- Hooks use TanStack Query for remote data (explicit staleTime/gcTime/retry).
- Routing uses TanStack Router only.
- Route loaders do not fetch remote data.

## Navigation
- Topic-first navigation is required.
- Prefer routes like `/politika`, `/ekonomija`, `/sport` and map internally to topic params.
- Separate topic pages will be added later; do not add them now.

## UI and design
- Presentational components only; business logic stays in hooks.
- Tailwind CSS only; use `clsx` + `tailwind-merge` for conditional classes.
- Icons from lucide-react only; dates via date-fns only.
- No heavy animations.
- Layout: header + centered main feed column + footer.
- No topic sidebar in this UI.

## Intentionally absent
- No login, accounts, or personalization.
- No infinite scroll.
- No topic sidebar.
- No client-side clustering or ranking (backend is authoritative).
- No ads or ad slots of any kind.
- Do not add any of the above without explicit instruction.

## Code quality
- Type everything; avoid `any`.
- Centralize constants in `src/lib/constants.ts`.
- Keep files small and focused; split when they grow.
- Do not add new libraries without explicit approval.
