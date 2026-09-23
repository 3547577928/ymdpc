import { AlertTriangle, Check, X } from 'lucide-react'
import type { FormEvent, ReactNode } from 'react'

type DialogBaseProps = {
  open: boolean
  title: string
  message?: ReactNode
  onCancel: () => void
}

type ConfirmDialogProps = DialogBaseProps & {
  confirmLabel?: string
  onConfirm: () => void | Promise<void>
}

type PromptDialogProps = DialogBaseProps & {
  value: string
  placeholder?: string
  submitLabel?: string
  onChange: (value: string) => void
  onSubmit: () => void | Promise<void>
}

function DialogFrame({ title, message, onCancel, children, tone = 'warning' }: DialogBaseProps & { children: ReactNode; tone?: 'warning' | 'success' }) {
  return <div className="modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.currentTarget === event.target) onCancel() }}>
    <section className="dialog-modal" role="dialog" aria-modal="true" aria-labelledby="dialog-title" onMouseDown={(event) => event.stopPropagation()}>
      <div className="dialog-head">
        <div className={`dialog-icon dialog-icon-${tone}`}>{tone === 'warning' ? <AlertTriangle size={18} /> : <Check size={18} />}</div>
        <button className="icon-button" onClick={onCancel} aria-label="关闭"><X size={17} /></button>
      </div>
      <h2 id="dialog-title">{title}</h2>
      {message && <p className="dialog-message">{message}</p>}
      {children}
    </section>
  </div>
}

export function ConfirmDialog({ open, title, message, confirmLabel = '确认', onConfirm, onCancel }: ConfirmDialogProps) {
  if (!open) return null
  return <DialogFrame open={open} title={title} message={message} onCancel={onCancel}>
    <div className="dialog-actions">
      <button className="button button-light" onClick={onCancel}>取消</button>
      <button className="button button-dark" onClick={() => void onConfirm()}>{confirmLabel}</button>
    </div>
  </DialogFrame>
}

export function PromptDialog({ open, title, message, value, placeholder, submitLabel = '提交', onChange, onSubmit, onCancel }: PromptDialogProps) {
  if (!open) return null
  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    void onSubmit()
  }
  return <DialogFrame open={open} title={title} message={message} onCancel={onCancel}>
    <form onSubmit={handleSubmit}>
      <textarea className="dialog-textarea" value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} rows={4} autoFocus />
      <div className="dialog-actions">
        <button type="button" className="button button-light" onClick={onCancel}>取消</button>
        <button type="submit" className="button button-dark" disabled={!value.trim()}>{submitLabel}</button>
      </div>
    </form>
  </DialogFrame>
}

export function NoticeDialog({ open, title, message, onCancel }: DialogBaseProps) {
  if (!open) return null
  return <DialogFrame open={open} title={title} message={message} onCancel={onCancel} tone="success">
    <div className="dialog-actions"><button className="button button-dark" onClick={onCancel}>知道了</button></div>
  </DialogFrame>
}
