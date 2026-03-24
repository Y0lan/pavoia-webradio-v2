import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 20001,
    host: true,                 // listen on 0.0.0.0
    strictPort: true,
    allowedHosts: [
      'orange.whatbox.ca',      // ← add your public host(s)
      'radio.nicemouth.box.ca',  // (add any reverse-proxied name)
      'nicemouth.box.ca',
      'pavoia.nicemouth.box.ca'

    ],
    proxy: { '/api': 'http://localhost:20000' } // your Node API
    // If you proxy HTTPS→HTTP and want HMR over WSS, also set:
    // hmr: { host: 'orange.whatbox.ca', protocol: 'wss', port: 443 }
  }
})

