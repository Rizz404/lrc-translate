import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    // Cloudflare quick tunnels (trycloudflare.com) hand out a new random
    // subdomain each run, so allow the whole suffix instead of one hostname.
    allowedHosts: [".trycloudflare.com"],
    proxy: {
      // 8080 falls inside a Windows Hyper-V/WSL2 dynamic port exclusion
      // range on this dev machine (`netsh interface ipv4 show
      // excludedportrange protocol=tcp` — typically 7726-8325 and/or
      // 50000-50059), which makes the backend's `net.Listen` fail with
      // "bind: An attempt was made to access a socket in a way forbidden by
      // its access permissions" even though nothing else is using the port.
      // 8500 sits outside that range; keep this in sync with backend/.env's
      // PORT.
      "/api": {
        target: "http://localhost:8500",
        changeOrigin: true,
      },
    },
  },
})
