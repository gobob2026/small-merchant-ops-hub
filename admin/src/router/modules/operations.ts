import { AppRouteRecord } from '@/types/router'

export const operationsRoutes: AppRouteRecord = {
  path: '/operations',
  name: 'Operations',
  component: '/index/index',
  meta: {
    title: '商家运营',
    icon: 'ri:store-2-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: 'hub',
      name: 'MerchantOpsHub',
      component: '/operations/hub',
      meta: {
        title: '运营台',
        icon: 'ri:line-chart-line',
        keepAlive: false,
        roles: ['R_SUPER', 'R_ADMIN']
      }
    }
  ]
}
