<template>
  <div class="merchant-ops-page">
    <h1 class="page-title">商家运营台</h1>
    <p class="page-subtitle">会员、订单、活动、归因、复购跟进统一操作与查看</p>

    <ElRow :gutter="16" class="mb-4">
      <ElCol :xs="24" :sm="12" :lg="6">
        <ElCard class="metric-card">
          <p class="metric-label">会员数</p>
          <h3>{{ summary.memberCount }}</h3>
        </ElCard>
      </ElCol>
      <ElCol :xs="24" :sm="12" :lg="6">
        <ElCard class="metric-card">
          <p class="metric-label">订单数</p>
          <h3>{{ summary.orderCount }}</h3>
        </ElCard>
      </ElCol>
      <ElCol :xs="24" :sm="12" :lg="6">
        <ElCard class="metric-card">
          <p class="metric-label">支付订单</p>
          <h3>{{ summary.paidOrderCount }}</h3>
        </ElCard>
      </ElCol>
      <ElCol :xs="24" :sm="12" :lg="6">
        <ElCard class="metric-card">
          <p class="metric-label">复购率</p>
          <h3>{{ summary.repurchaseRate.toFixed(2) }}%</h3>
        </ElCard>
      </ElCol>
    </ElRow>

    <ElRow :gutter="16" class="mb-4">
      <ElCol :xs="24" :sm="12" :lg="6">
        <ElCard class="metric-card">
          <p class="metric-label">活跃活动</p>
          <h3>{{ summary.activeCampaignCount }}</h3>
        </ElCard>
      </ElCol>
      <ElCol :xs="24" :sm="12" :lg="6">
        <ElCard class="metric-card">
          <p class="metric-label">GMV</p>
          <h3>{{ formatAmount(summary.revenueCents) }}</h3>
        </ElCard>
      </ElCol>
      <ElCol :xs="24" :sm="12" :lg="12">
        <ElCard class="metric-card">
          <p class="metric-label">渠道分布</p>
          <ElSpace wrap>
            <ElTag
              v-for="item in summary.channelBreakdown"
              :key="item.channel"
              type="success"
              effect="light"
            >
              {{ item.channel }}: {{ item.memberCount }}
            </ElTag>
          </ElSpace>
        </ElCard>
      </ElCol>
    </ElRow>

    <ElRow :gutter="16" class="mb-4">
      <ElCol :xs="24" :lg="8">
        <ElCard>
          <template #header>新增会员</template>
          <ElForm label-width="90px" @submit.prevent>
            <ElFormItem label="姓名">
              <ElInput v-model.trim="memberForm.name" maxlength="80" placeholder="请输入会员姓名" />
            </ElFormItem>
            <ElFormItem label="手机号">
              <ElInput v-model.trim="memberForm.phone" maxlength="20" placeholder="请输入手机号" />
            </ElFormItem>
            <ElFormItem label="渠道">
              <ElSelect v-model="memberForm.channel" class="w-full">
                <ElOption label="wechat" value="wechat" />
                <ElOption label="douyin" value="douyin" />
                <ElOption label="store" value="store" />
              </ElSelect>
            </ElFormItem>
            <ElButton type="primary" :loading="loading" @click="submitMember">创建会员</ElButton>
          </ElForm>
        </ElCard>
      </ElCol>

      <ElCol :xs="24" :lg="8">
        <ElCard>
          <template #header>新增订单</template>
          <ElForm label-width="90px" @submit.prevent>
            <ElFormItem label="会员">
              <ElSelect v-model.number="orderForm.memberId" class="w-full" placeholder="请选择会员">
                <ElOption
                  v-for="member in members"
                  :key="member.id"
                  :label="`${member.name} (${member.phone})`"
                  :value="member.id"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="金额(CNY)">
              <ElInputNumber v-model="orderForm.amountYuan" :min="0.01" :precision="2" :step="1" />
            </ElFormItem>
            <ElFormItem label="来源">
              <ElSelect v-model="orderForm.source" class="w-full">
                <ElOption label="wechat" value="wechat" />
                <ElOption label="douyin" value="douyin" />
                <ElOption label="offline" value="offline" />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="状态">
              <ElSelect v-model="orderForm.status" class="w-full">
                <ElOption label="paid" value="paid" />
                <ElOption label="pending" value="pending" />
                <ElOption label="refunded" value="refunded" />
              </ElSelect>
            </ElFormItem>
            <ElButton type="primary" :loading="loading" @click="submitOrder">创建订单</ElButton>
          </ElForm>
        </ElCard>
      </ElCol>

      <ElCol :xs="24" :lg="8">
        <ElCard>
          <template #header>
            <span>新增活动（仅超级管理员）</span>
          </template>
          <ElForm label-width="90px" @submit.prevent>
            <ElFormItem label="名称">
              <ElInput
                v-model.trim="campaignForm.name"
                maxlength="120"
                placeholder="请输入活动名称"
              />
            </ElFormItem>
            <ElFormItem label="渠道">
              <ElSelect v-model="campaignForm.channel" class="w-full">
                <ElOption label="wechat" value="wechat" />
                <ElOption label="douyin" value="douyin" />
                <ElOption label="store" value="store" />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="折扣(%)">
              <ElInputNumber
                v-model="campaignForm.discountPct"
                :min="1"
                :max="100"
                :precision="1"
                :step="0.5"
              />
            </ElFormItem>
            <ElFormItem label="状态">
              <ElSelect v-model="campaignForm.status" class="w-full">
                <ElOption label="active" value="active" />
                <ElOption label="draft" value="draft" />
                <ElOption label="closed" value="closed" />
              </ElSelect>
            </ElFormItem>
            <ElButton type="primary" :loading="loading" v-roles="'R_SUPER'" @click="submitCampaign">
              创建活动
            </ElButton>
            <ElTag v-roles="['R_ADMIN']" type="info" effect="light">R_ADMIN 仅可查看活动</ElTag>
          </ElForm>
        </ElCard>
      </ElCol>
    </ElRow>

    <ElRow :gutter="16" class="mb-4">
      <ElCol :xs="24" :lg="12">
        <ElCard>
          <template #header>会员列表</template>
          <ElTable :data="members" size="small">
            <ElTableColumn prop="name" label="姓名" min-width="100" />
            <ElTableColumn prop="phone" label="手机号" min-width="120" />
            <ElTableColumn prop="channel" label="渠道" min-width="90" />
            <ElTableColumn label="创建时间" min-width="150">
              <template #default="{ row }">
                {{ formatTime(row.createdAt) }}
              </template>
            </ElTableColumn>
          </ElTable>
        </ElCard>
      </ElCol>

      <ElCol :xs="24" :lg="12">
        <ElCard>
          <template #header>订单列表</template>
          <ElTable :data="orders" size="small">
            <ElTableColumn prop="orderNo" label="订单号" min-width="180" />
            <ElTableColumn prop="memberName" label="会员" min-width="100" />
            <ElTableColumn label="金额" min-width="100">
              <template #default="{ row }">
                {{ formatAmount(row.amountCents) }}
              </template>
            </ElTableColumn>
            <ElTableColumn prop="status" label="状态" min-width="90" />
            <ElTableColumn prop="source" label="来源" min-width="90" />
          </ElTable>
        </ElCard>
      </ElCol>
    </ElRow>

    <ElRow :gutter="16" class="mb-4">
      <ElCol :xs="24" :lg="12">
        <ElCard>
          <template #header>活动列表</template>
          <ElTable :data="campaigns" size="small">
            <ElTableColumn prop="name" label="活动名" min-width="140" />
            <ElTableColumn prop="channel" label="渠道" min-width="90" />
            <ElTableColumn prop="discountPct" label="折扣(%)" min-width="100" />
            <ElTableColumn prop="status" label="状态" min-width="90" />
            <ElTableColumn label="创建时间" min-width="150">
              <template #default="{ row }">
                {{ formatTime(row.createdAt) }}
              </template>
            </ElTableColumn>
          </ElTable>
        </ElCard>
      </ElCol>

      <ElCol :xs="24" :lg="12">
        <ElCard>
          <template #header>
            <div class="followup-head">
              <span>复购跟进名单</span>
              <ElSpace>
                <span class="text-gray-500">窗口(天)</span>
                <ElInputNumber v-model="followupDays" :min="1" :max="365" :step="1" size="small" />
                <ElButton size="small" @click="refreshFollowups">刷新</ElButton>
              </ElSpace>
            </div>
          </template>
          <ElTable :data="followups.items" size="small">
            <ElTableColumn prop="memberName" label="会员" min-width="100" />
            <ElTableColumn prop="phone" label="手机号" min-width="120" />
            <ElTableColumn prop="channel" label="渠道" min-width="90" />
            <ElTableColumn prop="paidOrderCount" label="支付单数" min-width="90" />
            <ElTableColumn label="最近支付" min-width="150">
              <template #default="{ row }">
                {{ row.lastPaidAt ? formatTime(row.lastPaidAt) : '-' }}
              </template>
            </ElTableColumn>
            <ElTableColumn prop="daysSinceLastPay" label="距今天数" min-width="90" />
          </ElTable>
        </ElCard>
      </ElCol>
    </ElRow>

    <ElCard>
      <template #header>
        <div class="report-head">
          <span>活动归因报表</span>
          <ElSpace wrap>
            <ElSelect v-model="reportQuery.status" clearable placeholder="状态" style="width: 110px">
              <ElOption label="active" value="active" />
              <ElOption label="draft" value="draft" />
              <ElOption label="closed" value="closed" />
            </ElSelect>
            <ElSelect
              v-model="reportQuery.channel"
              clearable
              placeholder="渠道"
              style="width: 110px"
            >
              <ElOption label="wechat" value="wechat" />
              <ElOption label="douyin" value="douyin" />
              <ElOption label="store" value="store" />
            </ElSelect>
            <ElInput
              v-model.trim="reportQuery.q"
              clearable
              placeholder="活动关键词"
              style="width: 180px"
            />
            <ElButton size="small" @click="refreshAttribution">查询</ElButton>
            <ElButton size="small" type="success" v-roles="'R_SUPER'" @click="exportAttributionCsv">
              导出 CSV
            </ElButton>
          </ElSpace>
        </div>
      </template>
      <ElTable :data="attribution.rows" size="small">
        <ElTableColumn prop="campaignName" label="活动名" min-width="150" />
        <ElTableColumn prop="channel" label="渠道" min-width="90" />
        <ElTableColumn prop="status" label="状态" min-width="90" />
        <ElTableColumn prop="targetMemberCount" label="目标会员" min-width="100" />
        <ElTableColumn prop="paidOrderCount" label="支付单数" min-width="100" />
        <ElTableColumn prop="convertedMemberCount" label="转化会员" min-width="100" />
        <ElTableColumn prop="repurchaseConvertedCount" label="复购转化" min-width="100" />
        <ElTableColumn label="营收" min-width="120">
          <template #default="{ row }">
            {{ formatAmount(row.revenueCents) }}
          </template>
        </ElTableColumn>
        <ElTableColumn label="转化率" min-width="90">
          <template #default="{ row }">
            {{ row.conversionRate.toFixed(2) }}%
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { ElMessage } from 'element-plus'
  import {
    fetchCampaignAttribution,
    fetchCreateMerchantCampaign,
    fetchCreateMerchantMember,
    fetchCreateMerchantOrder,
    fetchMerchantCampaigns,
    fetchMerchantFollowups,
    fetchMerchantMembers,
    fetchMerchantOrders,
    fetchMerchantSummary,
    getCampaignAttributionExportUrl
  } from '@/api/merchant-ops'

  defineOptions({ name: 'MerchantOpsHub' })

  const loading = ref(false)
  const followupDays = ref(30)

  const members = ref<Api.MerchantOps.Member[]>([])
  const orders = ref<Api.MerchantOps.Order[]>([])
  const campaigns = ref<Api.MerchantOps.Campaign[]>([])
  const followups = ref<Api.MerchantOps.FollowupPayload>({
    daysWindow: 30,
    items: []
  })
  const attribution = ref<Api.MerchantOps.CampaignAttributionPayload>({
    rows: []
  })
  const summary = ref<Api.MerchantOps.Summary>({
    memberCount: 0,
    orderCount: 0,
    paidOrderCount: 0,
    revenueCents: 0,
    repurchaseCount: 0,
    repurchaseRate: 0,
    activeCampaignCount: 0,
    channelBreakdown: []
  })

  const reportQuery = reactive<Api.MerchantOps.CampaignAttributionQueryParams>({
    status: undefined,
    channel: undefined,
    q: '',
    limit: 100
  })

  const memberForm = reactive<Api.MerchantOps.CreateMemberParams>({
    name: '',
    phone: '',
    channel: 'wechat'
  })

  const orderForm = reactive({
    memberId: 0,
    amountYuan: undefined as number | undefined,
    source: 'wechat',
    status: 'paid' as Api.MerchantOps.CreateOrderParams['status']
  })

  const campaignForm = reactive<Api.MerchantOps.CreateCampaignParams>({
    name: '',
    channel: 'wechat',
    discountPct: 10,
    status: 'active'
  })

  async function refreshFollowups() {
    followups.value = await fetchMerchantFollowups({ days: followupDays.value, limit: 50 })
  }

  async function refreshAttribution() {
    attribution.value = await fetchCampaignAttribution(reportQuery)
  }

  async function refreshAll() {
    loading.value = true
    try {
      const [summaryData, memberData, orderData, campaignData, followupData, attributionData] =
        await Promise.all([
          fetchMerchantSummary(),
          fetchMerchantMembers(),
          fetchMerchantOrders(),
          fetchMerchantCampaigns(),
          fetchMerchantFollowups({ days: followupDays.value, limit: 50 }),
          fetchCampaignAttribution(reportQuery)
        ])
      summary.value = summaryData
      members.value = memberData
      orders.value = orderData
      campaigns.value = campaignData
      followups.value = followupData
      attribution.value = attributionData
      if (members.value.length > 0 && orderForm.memberId === 0) {
        orderForm.memberId = members.value[0].id
      }
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : '加载数据失败')
    } finally {
      loading.value = false
    }
  }

  async function submitMember() {
    if (!memberForm.name || !memberForm.phone || !memberForm.channel) {
      ElMessage.warning('请填写完整会员信息')
      return
    }

    loading.value = true
    try {
      await fetchCreateMerchantMember(memberForm)
      ElMessage.success('会员创建成功')
      memberForm.name = ''
      memberForm.phone = ''
      await refreshAll()
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : '会员创建失败')
    } finally {
      loading.value = false
    }
  }

  async function submitOrder() {
    if (!orderForm.memberId) {
      ElMessage.warning('请选择会员')
      return
    }
    const amountYuan = orderForm.amountYuan
    if (!amountYuan || amountYuan <= 0) {
      ElMessage.warning('请输入有效订单金额')
      return
    }

    loading.value = true
    try {
      await fetchCreateMerchantOrder({
        memberId: orderForm.memberId,
        amountCents: Math.round(amountYuan * 100),
        source: orderForm.source,
        status: orderForm.status
      })
      ElMessage.success('订单创建成功')
      orderForm.amountYuan = undefined
      await refreshAll()
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : '订单创建失败')
    } finally {
      loading.value = false
    }
  }

  async function submitCampaign() {
    if (!campaignForm.name || !campaignForm.channel || !campaignForm.discountPct) {
      ElMessage.warning('请填写完整活动信息')
      return
    }

    loading.value = true
    try {
      await fetchCreateMerchantCampaign(campaignForm)
      ElMessage.success('活动创建成功')
      campaignForm.name = ''
      campaignForm.discountPct = 10
      await refreshAll()
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : '活动创建失败')
    } finally {
      loading.value = false
    }
  }

  function exportAttributionCsv() {
    const url = getCampaignAttributionExportUrl(reportQuery)
    window.open(url, '_blank')
  }

  function formatAmount(cents: number) {
    return new Intl.NumberFormat('zh-CN', {
      style: 'currency',
      currency: 'CNY'
    }).format(cents / 100)
  }

  function formatTime(value: string) {
    return new Date(value).toLocaleString('zh-CN')
  }

  onMounted(() => {
    refreshAll()
  })
</script>

<style scoped lang="scss">
  .merchant-ops-page {
    .page-title {
      margin: 0;
      font-size: 24px;
      font-weight: 700;
    }

    .page-subtitle {
      margin: 8px 0 16px;
      color: var(--art-text-gray-600);
    }

    .metric-card {
      h3 {
        margin: 8px 0 0;
        font-size: 24px;
      }

      .metric-label {
        margin: 0;
        color: var(--art-text-gray-600);
      }
    }

    .followup-head,
    .report-head {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 12px;
    }
  }
</style>
