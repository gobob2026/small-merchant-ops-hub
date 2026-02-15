/**
 * HTTP 请求封装模块
 * 基于 Axios 封装的 HTTP 请求工具，提供统一的请求/响应处理
 *
 * ## 主要功能
 *
 * - 请求/响应拦截器（自动添加 Token、统一错误处理）
 * - 401 未授权自动刷新令牌，刷新失败自动登出（带防抖机制）
 * - 请求失败自动重试（可配置）
 * - 统一的成功/错误消息提示
 * - 支持 GET/POST/PUT/DELETE 等常用方法
 *
 * @module utils/http
 * @author Art Design Pro Team
 */

import axios, {
  AxiosHeaders,
  AxiosRequestConfig,
  AxiosResponse,
  InternalAxiosRequestConfig
} from 'axios'
import { useUserStore } from '@/store/modules/user'
import { ApiStatus } from './status'
import { HttpError, handleError, showError, showSuccess } from './error'
import { $t } from '@/locales'
import { BaseResponse } from '@/types'

/** 请求配置常量 */
const REQUEST_TIMEOUT = 15000
const LOGOUT_DELAY = 500
const MAX_RETRIES = 0
const RETRY_DELAY = 1000
const UNAUTHORIZED_DEBOUNCE_TIME = 3000

/** 401防抖状态 */
let isUnauthorizedErrorShown = false
let unauthorizedTimer: NodeJS.Timeout | null = null
let refreshingAccessTokenPromise: Promise<string> | null = null

/** 扩展 AxiosRequestConfig */
interface ExtendedAxiosRequestConfig extends AxiosRequestConfig {
  showErrorMessage?: boolean
  showSuccessMessage?: boolean
}

interface RetryAxiosRequestConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
}

const { VITE_API_URL, VITE_WITH_CREDENTIALS } = import.meta.env

const jsonTransformResponse = [
  (data: string, headers: Record<string, string>) => {
    const contentType = headers['content-type']
    if (contentType?.includes('application/json')) {
      try {
        return JSON.parse(data)
      } catch {
        return data
      }
    }
    return data
  }
]

/** Axios实例 */
const axiosInstance = axios.create({
  timeout: REQUEST_TIMEOUT,
  baseURL: VITE_API_URL,
  withCredentials: VITE_WITH_CREDENTIALS === 'true',
  validateStatus: (status) => status >= 200 && status < 300,
  transformResponse: jsonTransformResponse
})

/** 刷新令牌请求实例（无拦截器，避免递归） */
const refreshAxiosInstance = axios.create({
  timeout: REQUEST_TIMEOUT,
  baseURL: VITE_API_URL,
  withCredentials: VITE_WITH_CREDENTIALS === 'true',
  validateStatus: (status) => status >= 200 && status < 300,
  transformResponse: jsonTransformResponse
})

/** 请求拦截器 */
axiosInstance.interceptors.request.use(
  (request: InternalAxiosRequestConfig) => {
    const { accessToken } = useUserStore()
    if (accessToken) request.headers.set('Authorization', accessToken)

    if (request.data && !(request.data instanceof FormData) && !request.headers['Content-Type']) {
      request.headers.set('Content-Type', 'application/json')
      request.data = JSON.stringify(request.data)
    }

    return request
  },
  (error) => {
    showError(createHttpError($t('httpMsg.requestConfigError'), ApiStatus.error))
    return Promise.reject(error)
  }
)

/** 响应拦截器 */
axiosInstance.interceptors.response.use(
  async (response: AxiosResponse<BaseResponse>) => {
    const { code, msg } = response.data
    if (code === ApiStatus.success) return response
    if (code === ApiStatus.unauthorized) {
      return retryWithRefreshedToken(response, msg)
    }
    throw createHttpError(msg || $t('httpMsg.requestFailed'), code)
  },
  async (error) => {
    if (error.response?.status === ApiStatus.unauthorized) {
      return retryWithRefreshedToken(error.response)
    }
    return Promise.reject(handleError(error))
  }
)

/** 统一创建HttpError */
function createHttpError(message: string, code: number) {
  return new HttpError(message, code)
}

/** 刷新 access token（并发单飞） */
async function refreshAccessToken(): Promise<string> {
  if (!refreshingAccessTokenPromise) {
    refreshingAccessTokenPromise = (async () => {
      const userStore = useUserStore()
      if (!userStore.refreshToken) {
        throw createHttpError($t('httpMsg.unauthorized'), ApiStatus.unauthorized)
      }

      const res = await refreshAxiosInstance.post<BaseResponse<Api.Auth.LoginResponse>>(
        '/api/auth/refresh',
        {
          refreshToken: userStore.refreshToken
        }
      )

      const { code, msg, data } = res.data
      if (
        code !== ApiStatus.success ||
        !data ||
        typeof data.token !== 'string' ||
        data.token === '' ||
        typeof data.refreshToken !== 'string' ||
        data.refreshToken === ''
      ) {
        throw createHttpError(msg || $t('httpMsg.unauthorized'), code || ApiStatus.unauthorized)
      }

      userStore.setToken(data.token, data.refreshToken)
      return data.token
    })().finally(() => {
      refreshingAccessTokenPromise = null
    })
  }

  return refreshingAccessTokenPromise
}

function isRefreshRequest(url?: string) {
  return typeof url === 'string' && url.includes('/api/auth/refresh')
}

function setAuthorizationHeader(config: RetryAxiosRequestConfig, accessToken: string) {
  if (!config.headers) {
    config.headers = new AxiosHeaders()
  }
  const headers = config.headers as {
    set?: (name: string, value: string) => void
    Authorization?: string
  }
  if (typeof headers.set === 'function') {
    headers.set('Authorization', accessToken)
    return
  }
  headers.Authorization = accessToken
}

/** 401 时尝试刷新令牌并重试一次 */
async function retryWithRefreshedToken(
  response: AxiosResponse<BaseResponse>,
  message?: string
): Promise<AxiosResponse<BaseResponse>> {
  const requestConfig = response.config as RetryAxiosRequestConfig
  if (!requestConfig || requestConfig._retry || isRefreshRequest(requestConfig.url)) {
    handleUnauthorizedError(message)
  }

  const userStore = useUserStore()
  if (!userStore.refreshToken) {
    handleUnauthorizedError(message)
  }

  let newAccessToken = ''
  try {
    newAccessToken = await refreshAccessToken()
  } catch {
    handleUnauthorizedError(message)
  }

  requestConfig._retry = true
  setAuthorizationHeader(requestConfig, newAccessToken)
  return axiosInstance.request<BaseResponse>(requestConfig)
}

/** 处理401错误（带防抖） */
function handleUnauthorizedError(message?: string): never {
  const error = createHttpError(message || $t('httpMsg.unauthorized'), ApiStatus.unauthorized)

  if (!isUnauthorizedErrorShown) {
    isUnauthorizedErrorShown = true
    logOut()

    unauthorizedTimer = setTimeout(resetUnauthorizedError, UNAUTHORIZED_DEBOUNCE_TIME)

    showError(error, true)
    throw error
  }

  throw error
}

/** 重置401防抖状态 */
function resetUnauthorizedError() {
  isUnauthorizedErrorShown = false
  if (unauthorizedTimer) clearTimeout(unauthorizedTimer)
  unauthorizedTimer = null
}

/** 退出登录函数 */
function logOut() {
  setTimeout(() => {
    useUserStore().logOut()
  }, LOGOUT_DELAY)
}

/** 是否需要重试 */
function shouldRetry(statusCode: number) {
  return [
    ApiStatus.requestTimeout,
    ApiStatus.internalServerError,
    ApiStatus.badGateway,
    ApiStatus.serviceUnavailable,
    ApiStatus.gatewayTimeout
  ].includes(statusCode)
}

/** 请求重试逻辑 */
async function retryRequest<T>(
  config: ExtendedAxiosRequestConfig,
  retries: number = MAX_RETRIES
): Promise<T> {
  try {
    return await request<T>(config)
  } catch (error) {
    if (retries > 0 && error instanceof HttpError && shouldRetry(error.code)) {
      await delay(RETRY_DELAY)
      return retryRequest<T>(config, retries - 1)
    }
    throw error
  }
}

/** 延迟函数 */
function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/** 请求函数 */
async function request<T = any>(config: ExtendedAxiosRequestConfig): Promise<T> {
  // POST | PUT 参数自动填充
  if (
    ['POST', 'PUT'].includes(config.method?.toUpperCase() || '') &&
    config.params &&
    !config.data
  ) {
    config.data = config.params
    config.params = undefined
  }

  try {
    const res = await axiosInstance.request<BaseResponse<T>>(config)

    // 显示成功消息
    if (config.showSuccessMessage && res.data.msg) {
      showSuccess(res.data.msg)
    }

    return res.data.data as T
  } catch (error) {
    if (error instanceof HttpError && error.code !== ApiStatus.unauthorized) {
      const showMsg = config.showErrorMessage !== false
      showError(error, showMsg)
    }
    return Promise.reject(error)
  }
}

/** API方法集合 */
const api = {
  get<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>({ ...config, method: 'GET' })
  },
  post<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>({ ...config, method: 'POST' })
  },
  put<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>({ ...config, method: 'PUT' })
  },
  del<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>({ ...config, method: 'DELETE' })
  },
  request<T>(config: ExtendedAxiosRequestConfig) {
    return retryRequest<T>(config)
  }
}

export default api
