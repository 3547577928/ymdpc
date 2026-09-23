import { ImagePlus, LoaderCircle } from 'lucide-react'
import { useState } from 'react'
import { uploadImage } from '../services/api'

type ImageUploadFieldProps = {
  value: string
  onChange: (value: string) => void
  allowUrl?: boolean
  buttonLabel?: string
  previewAlt?: string
  avatar?: boolean
}

export function ImageUploadField({ value, onChange, allowUrl = true, buttonLabel = '上传', previewAlt = '图片预览', avatar = false }: ImageUploadFieldProps) {
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')

  const handleChange = async (file?: File) => {
    if (!file) return
    setError('')
    setUploading(true)
    try {
      const result = await uploadImage(file)
      onChange(result.url)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '上传图片失败')
    } finally {
      setUploading(false)
    }
  }

  return <div className="image-upload-field">
    <div className={`image-upload-row ${allowUrl ? '' : 'file-only'}`}>{allowUrl && <input value={value} onChange={(event) => onChange(event.target.value)} placeholder="图片地址或上传图片" />}<label className="image-upload-button"><input type="file" accept="image/jpeg,image/png,image/gif,image/webp" onChange={(event) => { void handleChange(event.target.files?.[0]); event.target.value = '' }} disabled={uploading} />{uploading ? <LoaderCircle size={14} className="spin" /> : <ImagePlus size={14} />}<span>{uploading ? '上传中' : buttonLabel}</span></label>{!allowUrl && value && <button type="button" className="image-remove-button" onClick={() => onChange('')}>移除</button>}</div>
    {value && <img className={`image-upload-preview ${avatar ? 'is-avatar' : ''}`} src={value} alt={previewAlt} />}
    {error && <small className="image-upload-error">{error}</small>}
  </div>
}
