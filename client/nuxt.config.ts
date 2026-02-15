export default defineNuxtConfig({
  devtools: { enabled: true },
  compatibilityDate: "2026-02-15",
  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || "http://127.0.0.1:8080"
    }
  }
})
