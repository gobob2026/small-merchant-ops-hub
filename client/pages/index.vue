<template>
  <main class="page">
    <section class="hero">
      <h1>Small Merchant Ops Hub</h1>
      <p>Members, orders, campaigns, and repurchase follow-up in one workflow.</p>
      <p class="api-base">API: {{ apiBase }}</p>
    </section>

    <p v-if="errorMessage" class="error">{{ errorMessage }}</p>

    <section class="grid metrics">
      <article class="metric">
        <h3>Members</h3>
        <strong>{{ summary.memberCount }}</strong>
      </article>
      <article class="metric">
        <h3>Orders</h3>
        <strong>{{ summary.orderCount }}</strong>
      </article>
      <article class="metric">
        <h3>Paid Revenue</h3>
        <strong>{{ formatAmount(summary.revenueCents) }}</strong>
      </article>
      <article class="metric">
        <h3>Repurchase Rate</h3>
        <strong>{{ summary.repurchaseRate.toFixed(2) }}%</strong>
      </article>
      <article class="metric">
        <h3>Active Campaigns</h3>
        <strong>{{ summary.activeCampaignCount }}</strong>
      </article>
    </section>

    <section class="grid three-cols">
      <article class="panel">
        <h2>Create Member</h2>
        <form class="form" @submit.prevent="submitMember">
          <label>
            Name
            <input v-model.trim="memberForm.name" type="text" maxlength="80" required />
          </label>
          <label>
            Phone
            <input v-model.trim="memberForm.phone" type="text" maxlength="20" required />
          </label>
          <label>
            Channel
            <select v-model="memberForm.channel">
              <option value="wechat">wechat</option>
              <option value="douyin">douyin</option>
              <option value="store">store</option>
            </select>
          </label>
          <button type="submit" :disabled="loading">Create Member</button>
        </form>
      </article>

      <article class="panel">
        <h2>Create Order</h2>
        <form class="form" @submit.prevent="submitOrder">
          <label>
            Member
            <select v-model.number="orderForm.memberId" required>
              <option :value="0">Select member</option>
              <option v-for="member in members" :key="member.id" :value="member.id">
                {{ member.name }} ({{ member.phone }})
              </option>
            </select>
          </label>
          <label>
            Amount (CNY)
            <input v-model="orderForm.amountYuan" type="number" min="0.01" step="0.01" required />
          </label>
          <label>
            Source
            <select v-model="orderForm.source">
              <option value="wechat">wechat</option>
              <option value="douyin">douyin</option>
              <option value="offline">offline</option>
            </select>
          </label>
          <label>
            Status
            <select v-model="orderForm.status">
              <option value="paid">paid</option>
              <option value="pending">pending</option>
              <option value="refunded">refunded</option>
            </select>
          </label>
          <button type="submit" :disabled="loading || members.length === 0">Create Order</button>
        </form>
      </article>

      <article class="panel">
        <h2>Create Campaign</h2>
        <form class="form" @submit.prevent="submitCampaign">
          <label>
            Name
            <input v-model.trim="campaignForm.name" type="text" maxlength="120" required />
          </label>
          <label>
            Channel
            <select v-model="campaignForm.channel">
              <option value="wechat">wechat</option>
              <option value="douyin">douyin</option>
              <option value="store">store</option>
            </select>
          </label>
          <label>
            Discount (%)
            <input
              v-model.number="campaignForm.discountPct"
              type="number"
              min="1"
              max="100"
              step="0.5"
              required
            />
          </label>
          <label>
            Status
            <select v-model="campaignForm.status">
              <option value="active">active</option>
              <option value="draft">draft</option>
              <option value="closed">closed</option>
            </select>
          </label>
          <button type="submit" :disabled="loading">Create Campaign</button>
        </form>
      </article>
    </section>

    <section class="grid two-cols">
      <article class="panel">
        <h2>Members</h2>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Phone</th>
              <th>Channel</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="member in members" :key="member.id">
              <td>{{ member.name }}</td>
              <td>{{ member.phone }}</td>
              <td>{{ member.channel }}</td>
              <td>{{ formatTime(member.createdAt) }}</td>
            </tr>
          </tbody>
        </table>
      </article>

      <article class="panel">
        <h2>Recent Orders</h2>
        <table>
          <thead>
            <tr>
              <th>Order</th>
              <th>Member</th>
              <th>Amount</th>
              <th>Status</th>
              <th>Source</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="order in orders" :key="order.id">
              <td>{{ order.orderNo }}</td>
              <td>{{ order.memberName }}</td>
              <td>{{ formatAmount(order.amountCents) }}</td>
              <td>{{ order.status }}</td>
              <td>{{ order.source }}</td>
            </tr>
          </tbody>
        </table>
      </article>
    </section>

    <section class="grid two-cols">
      <article class="panel">
        <h2>Campaigns</h2>
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Channel</th>
              <th>Discount</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="campaign in campaigns" :key="campaign.id">
              <td>{{ campaign.name }}</td>
              <td>{{ campaign.channel }}</td>
              <td>{{ campaign.discountPct }}%</td>
              <td>{{ campaign.status }}</td>
            </tr>
          </tbody>
        </table>
      </article>

      <article class="panel">
        <div class="row-head">
          <h2>Repurchase Follow-ups</h2>
          <div class="inline-control">
            <span>Days</span>
            <input v-model.number="followupDays" type="number" min="1" max="365" />
            <button type="button" @click="refreshFollowups" :disabled="loading">Refresh</button>
          </div>
        </div>
        <table>
          <thead>
            <tr>
              <th>Member</th>
              <th>Channel</th>
              <th>Paid Orders</th>
              <th>Days Since Last</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in followups.items" :key="item.memberId">
              <td>{{ item.memberName }}</td>
              <td>{{ item.channel }}</td>
              <td>{{ item.paidOrderCount }}</td>
              <td>{{ item.daysSinceLastPay }}</td>
            </tr>
          </tbody>
        </table>
      </article>
    </section>

    <section class="grid">
      <article class="panel">
        <div class="row-head">
          <h2>Campaign Attribution Report</h2>
          <div class="inline-control">
            <input
              v-model.trim="reportQuery.q"
              type="text"
              placeholder="Campaign keyword"
              style="width: 160px"
            />
            <select v-model="reportQuery.status">
              <option value="">all status</option>
              <option value="active">active</option>
              <option value="draft">draft</option>
              <option value="closed">closed</option>
            </select>
            <select v-model="reportQuery.channel">
              <option value="">all channel</option>
              <option value="wechat">wechat</option>
              <option value="douyin">douyin</option>
              <option value="store">store</option>
            </select>
            <button type="button" @click="refreshAttribution" :disabled="loading">Query</button>
            <a class="link-btn" :href="attributionExportUrl" target="_blank" rel="noopener">CSV</a>
          </div>
        </div>
        <table>
          <thead>
            <tr>
              <th>Campaign</th>
              <th>Channel</th>
              <th>Status</th>
              <th>Target</th>
              <th>Paid Orders</th>
              <th>Converted</th>
              <th>Repurchase</th>
              <th>Revenue</th>
              <th>Rate</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="row in attribution.rows" :key="row.campaignId">
              <td>{{ row.campaignName }}</td>
              <td>{{ row.channel }}</td>
              <td>{{ row.status }}</td>
              <td>{{ row.targetMemberCount }}</td>
              <td>{{ row.paidOrderCount }}</td>
              <td>{{ row.convertedMemberCount }}</td>
              <td>{{ row.repurchaseConvertedCount }}</td>
              <td>{{ formatAmount(row.revenueCents) }}</td>
              <td>{{ row.conversionRate.toFixed(2) }}%</td>
            </tr>
          </tbody>
        </table>
      </article>
    </section>
  </main>
</template>

<script setup lang="ts">
type ApiEnvelope<T> = {
  code: number
  msg: string
  data: T
}

type Member = {
  id: number
  name: string
  phone: string
  channel: string
  createdAt: string
}

type Order = {
  id: number
  orderNo: string
  memberId: number
  memberName: string
  amountCents: number
  status: string
  source: string
  createdAt: string
}

type Campaign = {
  id: number
  name: string
  channel: string
  discountPct: number
  status: "draft" | "active" | "closed"
  createdAt: string
}

type FollowupItem = {
  memberId: number
  memberName: string
  channel: string
  paidOrderCount: number
  daysSinceLastPay: number
}

type FollowupPayload = {
  daysWindow: number
  items: FollowupItem[]
}

type CampaignAttributionRow = {
  campaignId: number
  campaignName: string
  channel: string
  status: string
  targetMemberCount: number
  paidOrderCount: number
  convertedMemberCount: number
  repurchaseConvertedCount: number
  revenueCents: number
  conversionRate: number
}

type CampaignAttributionPayload = {
  rows: CampaignAttributionRow[]
}

type ChannelBreakdown = {
  channel: string
  memberCount: number
}

type Summary = {
  memberCount: number
  orderCount: number
  paidOrderCount: number
  revenueCents: number
  repurchaseCount: number
  repurchaseRate: number
  activeCampaignCount: number
  channelBreakdown: ChannelBreakdown[]
}

const runtimeConfig = useRuntimeConfig()
const apiBase = runtimeConfig.public.apiBase as string

const loading = ref(false)
const errorMessage = ref("")

const members = ref<Member[]>([])
const orders = ref<Order[]>([])
const campaigns = ref<Campaign[]>([])
const followups = ref<FollowupPayload>({
  daysWindow: 30,
  items: []
})
const followupDays = ref(30)
const attribution = ref<CampaignAttributionPayload>({
  rows: []
})

const summary = ref<Summary>({
  memberCount: 0,
  orderCount: 0,
  paidOrderCount: 0,
  revenueCents: 0,
  repurchaseCount: 0,
  repurchaseRate: 0,
  activeCampaignCount: 0,
  channelBreakdown: []
})

const memberForm = reactive({
  name: "",
  phone: "",
  channel: "wechat"
})

const orderForm = reactive({
  memberId: 0,
  amountYuan: "",
  source: "wechat",
  status: "paid"
})

const campaignForm = reactive({
  name: "",
  channel: "wechat",
  discountPct: 10,
  status: "active" as "draft" | "active" | "closed"
})

const reportQuery = reactive({
  q: "",
  status: "",
  channel: ""
})

async function requestApi<T>(path: string, options: Record<string, unknown> = {}) {
  const payload = await $fetch<ApiEnvelope<T>>(path, {
    baseURL: apiBase,
    ...options
  })
  if (payload.code !== 200) {
    throw new Error(payload.msg || "Request failed")
  }
  return payload.data
}

async function loadMembers() {
  members.value = await requestApi<Member[]>("/api/v1/members")
  if (members.value.length > 0 && orderForm.memberId === 0) {
    orderForm.memberId = members.value[0].id
  }
}

async function loadOrders() {
  orders.value = await requestApi<Order[]>("/api/v1/orders")
}

async function loadCampaigns() {
  campaigns.value = await requestApi<Campaign[]>("/api/v1/campaigns")
}

async function loadSummary() {
  summary.value = await requestApi<Summary>("/api/v1/summary")
}

async function loadFollowups() {
  followups.value = await requestApi<FollowupPayload>(
    `/api/v1/followups?days=${encodeURIComponent(followupDays.value)}&limit=50`
  )
}

async function loadAttribution() {
  const query = new URLSearchParams()
  if (reportQuery.q) query.set("q", reportQuery.q)
  if (reportQuery.status) query.set("status", reportQuery.status)
  if (reportQuery.channel) query.set("channel", reportQuery.channel)
  query.set("limit", "100")
  attribution.value = await requestApi<CampaignAttributionPayload>(
    `/api/v1/reports/campaign-attribution?${query.toString()}`
  )
}

async function refreshAll() {
  loading.value = true
  errorMessage.value = ""
  try {
    await Promise.all([
      loadMembers(),
      loadOrders(),
      loadCampaigns(),
      loadSummary(),
      loadFollowups(),
      loadAttribution()
    ])
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : "Request failed"
  } finally {
    loading.value = false
  }
}

async function refreshFollowups() {
  loading.value = true
  errorMessage.value = ""
  try {
    await loadFollowups()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : "Request failed"
  } finally {
    loading.value = false
  }
}

async function refreshAttribution() {
  loading.value = true
  errorMessage.value = ""
  try {
    await loadAttribution()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : "Request failed"
  } finally {
    loading.value = false
  }
}

async function submitMember() {
  loading.value = true
  errorMessage.value = ""
  try {
    await requestApi<Member>("/api/v1/members", {
      method: "POST",
      body: memberForm
    })
    memberForm.name = ""
    memberForm.phone = ""
    await refreshAll()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : "Create member failed"
  } finally {
    loading.value = false
  }
}

async function submitOrder() {
  loading.value = true
  errorMessage.value = ""
  try {
    if (!orderForm.memberId) {
      throw new Error("Please select a member")
    }

    const amountYuan = Number(orderForm.amountYuan)
    if (!Number.isFinite(amountYuan) || amountYuan <= 0) {
      throw new Error("Please provide a valid amount")
    }

    await requestApi<Order>("/api/v1/orders", {
      method: "POST",
      body: {
        memberId: orderForm.memberId,
        amountCents: Math.round(amountYuan * 100),
        source: orderForm.source,
        status: orderForm.status
      }
    })
    orderForm.amountYuan = ""
    await refreshAll()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : "Create order failed"
  } finally {
    loading.value = false
  }
}

async function submitCampaign() {
  loading.value = true
  errorMessage.value = ""
  try {
    if (!campaignForm.name.trim()) {
      throw new Error("Please enter a campaign name")
    }
    if (!campaignForm.discountPct || campaignForm.discountPct <= 0 || campaignForm.discountPct > 100) {
      throw new Error("Please provide a valid discount")
    }

    await requestApi<Campaign>("/api/v1/campaigns", {
      method: "POST",
      body: campaignForm
    })
    campaignForm.name = ""
    campaignForm.discountPct = 10
    await refreshAll()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : "Create campaign failed"
  } finally {
    loading.value = false
  }
}

function formatAmount(cents: number) {
  return new Intl.NumberFormat("zh-CN", {
    style: "currency",
    currency: "CNY"
  }).format(cents / 100)
}

function formatTime(value: string) {
  return new Date(value).toLocaleString("zh-CN")
}

const attributionExportUrl = computed(() => {
  const query = new URLSearchParams()
  if (reportQuery.q) query.set("q", reportQuery.q)
  if (reportQuery.status) query.set("status", reportQuery.status)
  if (reportQuery.channel) query.set("channel", reportQuery.channel)
  query.set("limit", "100")
  return `${apiBase}/api/v1/reports/campaign-attribution/export?${query.toString()}`
})

onMounted(() => {
  refreshAll()
})
</script>

<style scoped>
.page {
  max-width: 1260px;
  margin: 0 auto;
  padding: 24px;
  font-family: "Segoe UI", -apple-system, BlinkMacSystemFont, "PingFang SC", sans-serif;
  color: #1f2937;
}

.hero h1 {
  margin: 0;
  font-size: 32px;
}

.hero p {
  margin: 10px 0 0;
  color: #4b5563;
}

.api-base {
  font-size: 13px;
}

.error {
  margin-top: 16px;
  padding: 10px 12px;
  border-radius: 10px;
  background: #fef2f2;
  color: #b91c1c;
}

.grid {
  margin-top: 18px;
  display: grid;
  gap: 14px;
}

.two-cols {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.three-cols {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.metrics {
  grid-template-columns: repeat(5, minmax(0, 1fr));
}

.panel,
.metric {
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  background: #ffffff;
  padding: 14px;
}

.metric h3 {
  margin: 0;
  color: #6b7280;
  font-size: 13px;
}

.metric strong {
  margin-top: 10px;
  display: block;
  font-size: 22px;
}

.form {
  display: grid;
  gap: 10px;
}

label {
  display: grid;
  gap: 6px;
  font-size: 13px;
}

input,
select,
button {
  border: 1px solid #d1d5db;
  border-radius: 10px;
  padding: 8px 10px;
  font: inherit;
}

button {
  cursor: pointer;
  background: #111827;
  color: #ffffff;
  border: none;
}

button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

th,
td {
  text-align: left;
  border-bottom: 1px solid #e5e7eb;
  padding: 8px 4px;
}

th {
  color: #6b7280;
  font-weight: 600;
}

.row-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.inline-control {
  display: flex;
  align-items: center;
  gap: 8px;
}

.inline-control input {
  width: 88px;
}

.link-btn {
  border: 1px solid #111827;
  color: #111827;
  border-radius: 10px;
  padding: 7px 12px;
  text-decoration: none;
  font-size: 13px;
}

@media (max-width: 1100px) {
  .three-cols {
    grid-template-columns: 1fr;
  }
  .metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 960px) {
  .two-cols,
  .metrics {
    grid-template-columns: 1fr;
  }
}
</style>
