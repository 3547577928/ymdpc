import { ImagePlus, LoaderCircle } from 'lucide-react'
import { useState } from 'react'
import { uploadImage } from '../services/api'

type MarkdownImageUploadButtonProps = {
  disabled?: boolean
  onInsert: (markdown: string) => void
  onError: (message: string) => void
}

export function MarkdownImageUploadButton({ disabled, onInsert, onError }: MarkdownImageUploadButtonProps) {
  const [uploading, setUploading] = useState(false)

  const handleFile = async (file?: File) => {
    if (!file) return
    setUploading(true)
    onError('')
    try {
      const result = await uploadImage(file)
      onInsert(`![${file.name.replace(/\.[^.]+$/, '')}](${result.url})`)
    } catch (reason) {
      onError(reason instanceof Error ? reason.message : '上传正文图片失败')
    } finally {
      setUploading(false)
    }
  }

  return <label className="markdown-import-button">
    <input type="file" accept="image/jpeg,image/png,image/gif,image/webp" disabled={disabled || uploading} onChange={(event) => { void handleFile(event.target.files?.[0]); event.target.value = '' }} />
    {uploading ? <LoaderCircle size={14} className="spin" /> : <ImagePlus size={14} />}
    <span>{uploading ? '上传中' : '插入图片'}</span>
  </label>
}
