import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
    plugins: [react()],
    resolve: {
        alias: {
            '@': path.resolve(__dirname, './src'),
        },
    },
    build: {
        outDir: 'dist',
    },
    server: {
        port: parseInt(process.env.WAILS_VITE_PORT || process.env.VITE_PORT || '5173'),
        strictPort: true,
    },
})
