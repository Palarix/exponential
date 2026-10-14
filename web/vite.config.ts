/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8080',
    },
  },
  test: {
    // Vitest blanks CSS imports by default; fonts.test.ts reads index.css?raw.
    css: { include: [/index\.css/] },
  },
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            { name: 'vendor-markdown', test: /node_modules[\\/](react-markdown|remark-|micromark|mdast-|hast-|unified|unist-)/ },
            { name: 'vendor-dnd', test: /node_modules[\\/]@dnd-kit[\\/]/ },
            { name: 'vendor-editor', test: /node_modules[\\/](@tiptap[\\/]|tiptap-markdown|prosemirror-|markdown-it)/ },
            { name: 'vendor-charts', test: /node_modules[\\/](recharts|d3-|victory-vendor)/ },
          ],
        },
      },
    },
  },
})
