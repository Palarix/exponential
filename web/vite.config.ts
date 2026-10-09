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
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor-markdown': ['react-markdown', 'remark-gfm', 'remark-breaks'],
          'vendor-dnd': ['@dnd-kit/core', '@dnd-kit/sortable', '@dnd-kit/utilities'],
          'vendor-editor': ['@tiptap/react', '@tiptap/starter-kit', 'tiptap-markdown', '@tiptap/extension-task-list', '@tiptap/extension-task-item'],
          'vendor-charts': ['recharts'],
        },
      },
    },
  },
})
