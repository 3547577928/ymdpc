import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { AUTH_EXPIRED_EVENT, getCurrentUser, type AuthUser } from './api'

type AuthContextValue = {
  user?: AuthUser
  /** 首次会话探测完成前为 true，页面据此决定是否等待登录态 */
  loading: boolean
  setUser: (user?: AuthUser) => void
  refresh: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue>({ loading: true, setUser: () => undefined, refresh: async () => undefined })

// AuthProvider 在应用启动时探测一次会话（/auth/me），全局共享登录用户，
// 取代之前每个页面各自请求的做法（一篇文章页曾发出 2-3 次重复的 me 请求）
export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser>()
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async () => {
    try {
      setUser(await getCurrentUser())
    } catch {
      setUser(undefined)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  // 受保护请求返回 401 时全局清理登录态；页面跳转由 Shell 处理
  useEffect(() => {
    const handleAuthExpired = () => setUser(undefined)
    window.addEventListener(AUTH_EXPIRED_EVENT, handleAuthExpired)
    return () => window.removeEventListener(AUTH_EXPIRED_EVENT, handleAuthExpired)
  }, [])

  return <AuthContext.Provider value={{ user, loading, setUser, refresh }}>{children}</AuthContext.Provider>
}

export function useAuth() {
  return useContext(AuthContext)
}
