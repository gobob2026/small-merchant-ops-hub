import request from '@/utils/http'

export function fetchMerchantSummary() {
  return request.get<Api.MerchantOps.Summary>({
    url: '/api/v1/summary'
  })
}

export function fetchMerchantMembers() {
  return request.get<Api.MerchantOps.Member[]>({
    url: '/api/v1/members'
  })
}

export function fetchMerchantOrders() {
  return request.get<Api.MerchantOps.Order[]>({
    url: '/api/v1/orders'
  })
}

export function fetchCreateMerchantMember(params: Api.MerchantOps.CreateMemberParams) {
  return request.post<Api.MerchantOps.Member>({
    url: '/api/v1/members',
    params
  })
}

export function fetchCreateMerchantOrder(params: Api.MerchantOps.CreateOrderParams) {
  return request.post<Api.MerchantOps.Order>({
    url: '/api/v1/orders',
    params
  })
}

export function fetchMerchantCampaigns() {
  return request.get<Api.MerchantOps.Campaign[]>({
    url: '/api/v1/campaigns'
  })
}

export function fetchCreateMerchantCampaign(params: Api.MerchantOps.CreateCampaignParams) {
  return request.post<Api.MerchantOps.Campaign>({
    url: '/api/v1/campaigns',
    params
  })
}

export function fetchMerchantFollowups(params?: Api.MerchantOps.FollowupQueryParams) {
  return request.get<Api.MerchantOps.FollowupPayload>({
    url: '/api/v1/followups',
    params
  })
}
