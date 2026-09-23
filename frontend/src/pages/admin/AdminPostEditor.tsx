import { Eye, PenLine, UploadCloud, X } from 'lucide-react'
import type { Dispatch, FormEvent, SetStateAction } from 'react'
import { useState } from 'react'
import type { PostInput } from '../../services/api'
import type { PostStatus, TagUsage } from '../../types'
import { ImageUploadField } from '../../components/ImageUploadField'
import { MarkdownContent } from '../../components/MarkdownContent'
import { MarkdownImportButton } from '../../components/MarkdownImportButton'
import { CategoryField } from '../../components/CategoryField'

type AdminPostEditorProps = {
  editingID?: number
  editorLoading: boolean
  saving: boolean
  error: string
  draft: PostInput
  categories: TagUsage[]
  setDraft: Dispatch<SetStateAction<PostInput>>
  onClose: () => void
  onSubmit: (event: FormEvent) => void
  onSaveDraft: () => void
}

export function AdminPostEditor({ editingID, editorLoading, saving, error, draft, categories, setDraft, onClose, onSubmit, onSaveDraft }: AdminPostEditorProps) {
  const [mode, setMode] = useState<'write' | 'preview'>('write')
  const [importError, setImportError] = useState('')

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <form className="editor-modal" onSubmit={onSubmit} onClick={(event) => event.stopPropagation()} aria-busy={editorLoading}>
        <div className="modal-head">
          <div>
            <div className="eyebrow">{editingID ? 'Edit post' : 'New post'}</div>
            <h2>{editingID ? '编辑文章' : '写一篇新文章'}</h2>
          </div>
          <button type="button" className="icon-button" onClick={onClose} aria-label="关闭"><X size={17} /></button>
        </div>
        {editorLoading ? <div className="editor-loading">正在读取文章。</div> : <>
          {error && <div className="form-error">{error}</div>}
          <div className="editor-grid">
            <label>标题<input required value={draft.title} onChange={(event) => setDraft({ ...draft, title: event.target.value })} placeholder="输入文章标题" /></label>
            <label>链接标识<input value={draft.slug} onChange={(event) => setDraft({ ...draft, slug: event.target.value })} placeholder="留空则根据标题生成" /></label>
          </div>
          <label>摘要<textarea value={draft.summary} onChange={(event) => setDraft({ ...draft, summary: event.target.value })} placeholder="用一句话说明这篇文章" /></label>
          <div className="editor-grid">
            <label>标签<input value={draft.tags.join(', ')} onChange={(event) => setDraft({ ...draft, tags: event.target.value.split(',').map((tag) => tag.trim()).filter(Boolean) })} placeholder="React, Go" /></label>
            {editingID && <label>状态<select value={draft.status} onChange={(event) => setDraft({ ...draft, status: event.target.value as PostStatus })}><option value="draft">草稿</option><option value="scheduled">待发布</option><option value="published">已发布</option><option value="archived">已归档</option></select></label>}
            <div className="editor-field"><span>分类</span><CategoryField categories={categories} value={draft.categoryId} onChange={(categoryId) => setDraft({ ...draft, categoryId })} onError={setImportError} /></div>
          </div>
          <label>封面图片<ImageUploadField value={draft.coverImage} onChange={(coverImage) => setDraft({ ...draft, coverImage })} /></label>
          <label>计划发布时间<input type="datetime-local" value={draft.scheduledAt ? new Date(new Date(draft.scheduledAt).getTime() - new Date(draft.scheduledAt).getTimezoneOffset() * 60_000).toISOString().slice(0, 16) : ''} onChange={(event) => setDraft({ ...draft, scheduledAt: event.target.value ? new Date(event.target.value).toISOString() : null })} /></label>
          <label className="checkbox-field"><input type="checkbox" checked={draft.featured} onChange={(event) => setDraft({ ...draft, featured: event.target.checked })} />设为精选文章</label>
          <div className="editor-content-tools"><MarkdownImportButton disabled={saving} onImport={(document) => { setDraft((current) => ({ ...current, content: document.content, title: current.title.trim() ? current.title : document.title })); setMode('preview'); setImportError('') }} onError={setImportError} /><div className="editor-mode" role="tablist" aria-label="正文模式"><button type="button" className={mode === 'write' ? 'is-active' : ''} onClick={() => setMode('write')}><PenLine size={14} /> 编辑</button><button type="button" className={mode === 'preview' ? 'is-active' : ''} onClick={() => setMode('preview')}><Eye size={14} /> 预览</button></div></div>
          {importError && <div className="form-error">{importError}</div>}
          {mode === 'write' ? <label>正文<textarea required className="editor-textarea" value={draft.content} onChange={(event) => setDraft({ ...draft, content: event.target.value })} placeholder="支持 Markdown" /></label> : <div className="editor-preview"><MarkdownContent className="article-content" content={draft.content || '*这里会显示 Markdown 预览。*'} /></div>}
          <div className="editor-actions">
            {!editingID && <button type="button" className="button button-light" onClick={onSaveDraft} disabled={saving}>保存草稿</button>}
            <button type="submit" className="button button-dark" disabled={saving}><UploadCloud size={16} /> {saving ? '保存中' : editingID ? '保存修改' : '发布文章'}</button>
          </div>
        </>}
      </form>
    </div>
  )
}
