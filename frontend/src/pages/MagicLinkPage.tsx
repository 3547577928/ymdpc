import { CheckCircle2, LoaderCircle, XCircle } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { verifyMagicLink } from '../services/api'

export function MagicLinkPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [state, setState] = useState<'loading' | 'success' | 'error'>('loading')
  const [message, setMessage] = useState('正在验证登录链接。')

  useEffect(() => {
    const token = searchParams.get('token')
    if (!token) {
      setState('error')
      setMessage('登录链接不完整，请重新获取。')
      return
    }
    let active = true
    verifyMagicLink(token).then(() => {
      if (!active) return
      setState('success')
      setMessage('登录成功，正在返回首页。')
      window.setTimeout(() => navigate('/', { replace: true }), 700)
    }).catch((reason) => {
      if (!active) return
      setState('error')
      setMessage(reason instanceof Error ? reason.message : '登录链接无效或已过期。')
    })
    return () => { active = false }
  }, [navigate, searchParams])

  return <section className="auth-page"><div className="auth-card magic-link-result">
    {state === 'loading' && <LoaderCircle className="spin" size={32} />}
    {state === 'success' && <CheckCircle2 size={32} />}
    {state === 'error' && <XCircle size={32} />}
    <div className="eyebrow">Quiet Signal / Email sign in</div>
    <h1>{state === 'loading' ? '验证中。' : state === 'success' ? '欢迎回来。' : '链接不可用。'}</h1>
    <p>{message}</p>
    {state === 'error' && <Link className="button button-dark" to="/login">返回登录</Link>}
  </div></section>
}
