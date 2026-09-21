import { ArrowRight, LockKeyhole, UserRound } from 'lucide-react'
import { FormEvent, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { login, register } from '../services/api'

export function AuthPage() {
  const location = useLocation()
  const navigate = useNavigate()
  const isRegister = location.pathname === '/register'
  const [form, setForm] = useState({ username: '', nickname: '', password: '' })
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    setSaving(true)
    try {
      if (isRegister) await register(form.username, form.password, form.nickname)
      else await login(form.username, form.password)
      navigate('/', { replace: true })
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '操作失败')
    } finally {
      setSaving(false)
    }
  }

  return <section className="auth-page">
    <form className="auth-card" onSubmit={submit}>
      <div className="auth-mark">QS <span>COMMUNITY</span></div>
      <div className="eyebrow">Quiet Signal / {isRegister ? 'Join' : 'Sign in'}</div>
      <h1>{isRegister ? '加入讨论。' : '回到这里。'}</h1>
      <p>{isRegister ? '注册一个账号，写下你的判断，也参与别人的讨论。' : '登录后可以发文章、评论、点赞和关注作者。'}</p>
      <label><span>用户名</span><div className="input-with-icon"><UserRound size={16} /><input required minLength={3} value={form.username} onChange={(event) => setForm({ ...form, username: event.target.value })} autoComplete="username" /></div></label>
      {isRegister && <label><span>昵称</span><input required value={form.nickname} onChange={(event) => setForm({ ...form, nickname: event.target.value })} /></label>}
      <label><span>密码</span><div className="input-with-icon"><LockKeyhole size={16} /><input required minLength={8} type="password" value={form.password} onChange={(event) => setForm({ ...form, password: event.target.value })} autoComplete={isRegister ? 'new-password' : 'current-password'} /></div></label>
      {error && <div className="form-error">{error}</div>}
      <button className="button button-dark" type="submit" disabled={saving}>{saving ? '处理中' : isRegister ? '创建账号' : '登录'} <ArrowRight size={16} /></button>
      <div className="auth-switch">{isRegister ? '已经有账号？' : '还没有账号？'} <Link to={isRegister ? '/login' : '/register'}>{isRegister ? '直接登录' : '注册账号'}</Link></div>
    </form>
  </section>
}
