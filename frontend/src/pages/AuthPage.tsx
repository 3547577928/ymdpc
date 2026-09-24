import { ArrowRight, KeyRound, LockKeyhole, Mail, UserRound } from 'lucide-react'
import { FormEvent, useEffect, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { login, register, requestEmailCode, verifyEmailCode } from '../services/api'
import { useAuth } from '../services/auth'
import { usePageMeta } from '../utils/usePageMeta'

export function AuthPage() {
  const location = useLocation()
  usePageMeta(location.pathname === '/register' ? '注册' : '登录')
  const navigate = useNavigate()
  const isRegister = location.pathname === '/register'
  const [form, setForm] = useState({ username: '', nickname: '', password: '' })
  const [email, setEmail] = useState('')
  const [emailCode, setEmailCode] = useState('')
  const [codeSent, setCodeSent] = useState(false)
  const [cooldown, setCooldown] = useState(0)
  const [mode, setMode] = useState<'password' | 'email'>('password')
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [saving, setSaving] = useState(false)
  const { setUser } = useAuth()
  // 登录成功后回跳来源页（如从文章页点「登录后参与」进来）；只允许站内路径，防开放跳转
  const from = new URLSearchParams(location.search).get('from')
  const safeFrom = from && from.startsWith('/') && !from.startsWith('//') ? from : '/'

  // 重发倒计时：与后端按邮箱 60s 限流配合，避免用户连点被 429
  useEffect(() => {
    if (cooldown <= 0) return
    const timer = window.setTimeout(() => setCooldown((value) => value - 1), 1000)
    return () => window.clearTimeout(timer)
  }, [cooldown])

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    setMessage('')
    setSaving(true)
    try {
      if (isRegister) {
        setUser(await register(form.username, form.password, form.nickname))
        navigate(safeFrom, { replace: true })
      } else if (mode === 'email' && !codeSent) {
        await requestEmailCode(email)
        setCodeSent(true)
        setCooldown(60)
        setMessage('验证码已发送，请在下方输入邮件中的 6 位数字。')
      } else if (mode === 'email') {
        setUser(await verifyEmailCode(email, emailCode))
        navigate(safeFrom, { replace: true })
      } else {
        setUser(await login(form.username, form.password))
        navigate(safeFrom, { replace: true })
      }
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
      {!isRegister && <div className="auth-mode-switch" role="tablist" aria-label="登录方式"><button type="button" className={mode === 'password' ? 'active' : ''} onClick={() => { setMode('password'); setError(''); setMessage('') }}>密码登录</button><button type="button" className={mode === 'email' ? 'active' : ''} onClick={() => { setMode('email'); setError(''); setMessage('') }}>邮箱登录</button></div>}
      {(isRegister || mode === 'password') && <>
        <label><span>用户名</span><div className="input-with-icon"><UserRound size={16} /><input required minLength={3} value={form.username} onChange={(event) => setForm({ ...form, username: event.target.value })} autoComplete="username" /></div></label>
        {isRegister && <label><span>昵称</span><input required value={form.nickname} onChange={(event) => setForm({ ...form, nickname: event.target.value })} /></label>}
        <label><span>密码</span><div className="input-with-icon"><LockKeyhole size={16} /><input required minLength={8} type="password" value={form.password} onChange={(event) => setForm({ ...form, password: event.target.value })} autoComplete={isRegister ? 'new-password' : 'current-password'} /></div></label>
      </>}
      {!isRegister && mode === 'email' && <>
        <label><span>邮箱</span><div className="input-with-icon"><Mail size={16} /><input required type="email" value={email} onChange={(event) => setEmail(event.target.value)} autoComplete="email" placeholder="name@example.com" readOnly={codeSent} /></div></label>
        {codeSent && <label><span>验证码</span><div className="input-with-icon"><KeyRound size={16} /><input required inputMode="numeric" pattern="[0-9]{6}" minLength={6} maxLength={6} value={emailCode} onChange={(event) => setEmailCode(event.target.value.replace(/\D/g, '').slice(0, 6))} autoComplete="one-time-code" placeholder="6 位数字" autoFocus /></div></label>}
      </>}
      {error && <div className="form-error">{error}</div>}
      {message && <div className="form-success">{message}</div>}
      <button className="button button-dark" type="submit" disabled={saving || (!isRegister && mode === 'email' && !codeSent && cooldown > 0)}>{saving ? '处理中' : isRegister ? '创建账号' : mode === 'email' ? codeSent ? '验证并登录' : cooldown > 0 ? `重新发送（${cooldown}s）` : '发送验证码' : '登录'} <ArrowRight size={16} /></button>
      {!isRegister && mode === 'email' && codeSent && <div className="auth-code-actions"><button type="button" onClick={() => { setCodeSent(false); setEmailCode(''); setMessage(''); setError('') }}>修改邮箱</button><button type="button" disabled={saving || cooldown > 0} onClick={async () => { setSaving(true); setError(''); try { await requestEmailCode(email); setEmailCode(''); setCooldown(60); setMessage('新的验证码已发送。') } catch (reason) { setError(reason instanceof Error ? reason.message : '发送失败') } finally { setSaving(false) } }}>{cooldown > 0 ? `重新发送（${cooldown}s）` : '重新发送'}</button></div>}
      <div className="auth-switch">{isRegister ? '已经有账号？' : '还没有账号？'} <Link to={isRegister ? '/login' : '/register'}>{isRegister ? '直接登录' : '注册账号'}</Link></div>
    </form>
  </section>
}
