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
