<template>
  <div class="merchant-ops-page">
    <h1 class="page-title">商家运营台</h1>
    <p class="page-subtitle">会员、订单、复购数据统一操作与查看</p>

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
      <ElCol :xs="24" :lg="12">
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

      <ElCol :xs="24" :lg="12">
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

    <ElCard>
      <template #header>渠道分布</template>
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
  </div>
</template>

<script setup lang="ts">
  import { ElMessage } from 'element-plus'
  import {
    fetchCreateMerchantMember,
    fetchCreateMerchantOrder,
    fetchMerchantMembers,
    fetchMerchantOrders,
    fetchMerchantSummary
  } from '@/api/merchant-ops'

  defineOptions({ name: 'MerchantOpsHub' })

  const loading = ref(false)

  const members = ref<Api.MerchantOps.Member[]>([])
  const orders = ref<Api.MerchantOps.Order[]>([])
  const summary = ref<Api.MerchantOps.Summary>({
    memberCount: 0,
    orderCount: 0,
    paidOrderCount: 0,
    revenueCents: 0,
    repurchaseCount: 0,
    repurchaseRate: 0,
    channelBreakdown: []
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

  async function refreshAll() {
    loading.value = true
    try {
      const [summaryData, memberData, orderData] = await Promise.all([
        fetchMerchantSummary(),
        fetchMerchantMembers(),
        fetchMerchantOrders()
      ])
      summary.value = summaryData
      members.value = memberData
      orders.value = orderData
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
  }
</style>
