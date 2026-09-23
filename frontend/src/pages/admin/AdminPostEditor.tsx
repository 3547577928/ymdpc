import { UploadCloud, X } from 'lucide-react'
import type { Dispatch, FormEvent, SetStateAction } from 'react'
import type { PostInput } from '../../services/api'
import type { PostStatus, TagUsage } from '../../types'

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
            {editingID && <label>状态<select value={draft.status} onChange={(event) => setDraft({ ...draft, status: event.target.value as PostStatus })}><option value="draft">草稿</option><option value="published">已发布</option><option value="archived">已归档</option></select></label>}
            <label>分类<select value={draft.categoryId ?? ''} onChange={(event) => setDraft({ ...draft, categoryId: event.target.value ? Number(event.target.value) : null })}><option value="">不设分类</option>{categories.map((category) => <option key={category.id} value={category.id}>{category.name}</option>)}</select></label>
          </div>
          <label>封面地址<input value={draft.coverImage} onChange={(event) => setDraft({ ...draft, coverImage: event.target.value })} placeholder="https://..." /></label>
          <label className="checkbox-field"><input type="checkbox" checked={draft.featured} onChange={(event) => setDraft({ ...draft, featured: event.target.checked })} />设为精选文章</label>
          <label>正文<textarea required className="editor-textarea" value={draft.content} onChange={(event) => setDraft({ ...draft, content: event.target.value })} placeholder="支持 Markdown" /></label>
          <div className="editor-actions">
            {!editingID && <button type="button" className="button button-light" onClick={onSaveDraft} disabled={saving}>保存草稿</button>}
            <button type="submit" className="button button-dark" disabled={saving}><UploadCloud size={16} /> {saving ? '保存中' : editingID ? '保存修改' : '发布文章'}</button>
          </div>
        </>}
      </form>
    </div>
  )
}
