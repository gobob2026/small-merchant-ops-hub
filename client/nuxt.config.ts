export default defineNuxtConfig({
  devtools: { enabled: true },
  compatibilityDate: "2026-02-15",
  runtimeConfig: {
    public: {
      // dev 默认连本地后端，production 默认同源，均可通过 NUXT_PUBLIC_API_BASE 覆盖
      apiBase:
        process.env.NUXT_PUBLIC_API_BASE ||
        (process.env.NODE_ENV === "production" ? "" : "http://127.0.0.1:8080")
    }
  }
})
