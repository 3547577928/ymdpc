import type { FormEvent } from 'react'

type AdminLoginProps = {
  username: string
  password: string
  error: string
  onUsernameChange: (username: string) => void
  onPasswordChange: (password: string) => void
  onSubmit: (event: FormEvent) => void
}

export function AdminLogin({ username, password, error, onUsernameChange, onPasswordChange, onSubmit }: AdminLoginProps) {
  return <section className="admin-login-page">
    <form className="admin-login-card" onSubmit={onSubmit}>
      <div className="admin-logo">QS <span>CONTROL</span></div>
      <div className="eyebrow">Private workspace</div>
      <h1>登录管理后台</h1>
      <p>管理文章、草稿和公开内容。</p>
      <label>用户名<input autoComplete="username" value={username} onChange={(event) => onUsernameChange(event.target.value)} /></label>
      <label>密码<input type="password" autoComplete="current-password" value={password} onChange={(event) => onPasswordChange(event.target.value)} /></label>
      {error && <div className="form-error">{error}</div>}
      <button className="button button-dark" type="submit">登录后台</button>
    </form>
  </section>
}
