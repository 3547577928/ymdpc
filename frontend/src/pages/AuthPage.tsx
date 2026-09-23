import { ArrowRight, KeyRound, LockKeyhole, Mail, UserRound } from 'lucide-react'
import { FormEvent, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { login, register, requestEmailCode, verifyEmailCode } from '../services/api'

export function AuthPage() {
  const location = useLocation()
  const navigate = useNavigate()
  const isRegister = location.pathname === '/register'
  const [form, setForm] = useState({ username: '', nickname: '', password: '' })
  const [email, setEmail] = useState('')
  const [emailCode, setEmailCode] = useState('')
  const [codeSent, setCodeSent] = useState(false)
  const [mode, setMode] = useState<'password' | 'email'>('password')
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [saving, setSaving] = useState(false)

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    setMessage('')
    setSaving(true)
    try {
      if (isRegister) {
        await register(form.username, form.password, form.nickname)
        navigate('/', { replace: true })
      } else if (mode === 'email' && !codeSent) {
        await requestEmailCode(email)
        setCodeSent(true)
        setMessage('验证码已发送，请在下方输入邮件中的 6 位数字。')
      } else if (mode === 'email') {
        await verifyEmailCode(email, emailCode)
        navigate('/', { replace: true })
      } else {
        await login(form.username, form.password)
        navigate('/', { replace: true })
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
      <button className="button button-dark" type="submit" disabled={saving}>{saving ? '处理中' : isRegister ? '创建账号' : mode === 'email' ? codeSent ? '验证并登录' : '发送验证码' : '登录'} <ArrowRight size={16} /></button>
      {!isRegister && mode === 'email' && codeSent && <div className="auth-code-actions"><button type="button" onClick={() => { setCodeSent(false); setEmailCode(''); setMessage(''); setError('') }}>修改邮箱</button><button type="button" disabled={saving} onClick={async () => { setSaving(true); setError(''); try { await requestEmailCode(email); setEmailCode(''); setMessage('新的验证码已发送。') } catch (reason) { setError(reason instanceof Error ? reason.message : '发送失败') } finally { setSaving(false) } }}>重新发送</button></div>}
      <div className="auth-switch">{isRegister ? '已经有账号？' : '还没有账号？'} <Link to={isRegister ? '/login' : '/register'}>{isRegister ? '直接登录' : '注册账号'}</Link></div>
    </form>
  </section>
}
