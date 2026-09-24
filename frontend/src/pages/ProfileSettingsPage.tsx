import { ArrowLeft, KeyRound, Save } from 'lucide-react'
import { FormEvent, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { updatePassword, updateProfile } from '../services/api'
import { useAuth } from '../services/auth'
import { usePageMeta } from '../utils/usePageMeta'
import { ImageUploadField } from '../components/ImageUploadField'

export function ProfileSettingsPage() {
  usePageMeta('账号设置')
  const navigate = useNavigate()
  const { user, loading, setUser } = useAuth()
  const [form, setForm] = useState({ nickname: '', avatar: '', bio: '', email: '' })
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [pwdForm, setPwdForm] = useState({ currentPassword: '', newPassword: '' })
  const [pwdError, setPwdError] = useState('')
  const [pwdMessage, setPwdMessage] = useState('')
  const [pwdSaving, setPwdSaving] = useState(false)

  // 登录态由 AuthProvider 提供：未登录跳登录页，探测完成前保持加载态
  useEffect(() => {
    if (loading) return
    if (!user) { navigate('/login?from=/settings/profile'); return }
    setForm({ nickname: user.nickname, avatar: user.avatar, bio: user.bio, email: user.email ?? '' })
  }, [user, loading, navigate])

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setSaving(true)
    setError('')
    try {
      const updated = await updateProfile(form)
      setUser(updated) // 同步全局登录态，导航栏昵称/头像立即更新
      navigate(`/users/${updated.username}`)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '保存资料失败')
    } finally {
      setSaving(false)
    }
  }

  // 提交修改密码，成功后清空表单并提示
  const submitPassword = async (event: FormEvent) => {
    event.preventDefault()
    setPwdSaving(true)
    setPwdError('')
    setPwdMessage('')
    try {
      await updatePassword(pwdForm)
      setPwdForm({ currentPassword: '', newPassword: '' })
      setPwdMessage('密码已更新')
    } catch (reason) {
      setPwdError(reason instanceof Error ? reason.message : '修改密码失败')
    } finally {
      setPwdSaving(false)
    }
  }

  if (loading || !user) return <div className="container page-state"><span className="eyebrow">Settings</span><h1>正在读取资料。</h1></div>
  return <section className="settings-page container"><Link className="back-link" to={`/users/${user.username}`}><ArrowLeft size={15} /> 返回个人主页</Link><div className="eyebrow">Account / Profile</div><h1>编辑资料</h1><form className="settings-form" onSubmit={submit}><label>登录邮箱<input type="email" value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} placeholder="用于邮箱魔法链接登录" autoComplete="email" /></label><label>昵称<input required value={form.nickname} onChange={(event) => setForm({ ...form, nickname: event.target.value })} /></label><div className="settings-field"><span>头像</span><ImageUploadField value={form.avatar} onChange={(avatar) => setForm({ ...form, avatar })} allowUrl={false} buttonLabel="上传头像" previewAlt="头像预览" avatar /></div><label>个人简介<textarea value={form.bio} onChange={(event) => setForm({ ...form, bio: event.target.value })} rows={5} placeholder="介绍一下你自己" /></label>{error && <div className="form-error">{error}</div>}<button className="button button-dark" disabled={saving}><Save size={15} /> {saving ? '保存中' : '保存资料'}</button></form><form className="settings-form" onSubmit={submitPassword}><div className="eyebrow">Account / Security</div><h2><KeyRound size={18} /> 修改密码</h2><label>当前密码<input type="password" required autoComplete="current-password" value={pwdForm.currentPassword} onChange={(event) => setPwdForm({ ...pwdForm, currentPassword: event.target.value })} /></label><label>新密码<input type="password" required minLength={8} autoComplete="new-password" value={pwdForm.newPassword} onChange={(event) => setPwdForm({ ...pwdForm, newPassword: event.target.value })} placeholder="至少 8 个字符" /></label>{pwdError && <div className="form-error">{pwdError}</div>}{pwdMessage && <div className="form-success">{pwdMessage}</div>}<button className="button button-dark" disabled={pwdSaving}><KeyRound size={15} /> {pwdSaving ? '保存中' : '更新密码'}</button></form></section>
}
