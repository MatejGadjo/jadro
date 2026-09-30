import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'
import { visualizer } from 'rollup-plugin-visualizer'

// https://vite.dev/config/
export default defineConfig({
  // `ANALYZE=1 npm run build` writes and opens stats.html (bundle report).
  plugins: [react(), tailwindcss(), process.env.ANALYZE ? visualizer({ open: true }) : null],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
