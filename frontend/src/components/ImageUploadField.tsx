import { ImagePlus, LoaderCircle } from 'lucide-react'
import { useState } from 'react'
import { uploadImage } from '../services/api'

export function ImageUploadField({ value, onChange }: { value: string; onChange: (value: string) => void }) {
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
    <div className="image-upload-row"><input value={value} onChange={(event) => onChange(event.target.value)} placeholder="图片地址或上传图片" /><label className="image-upload-button"><input type="file" accept="image/jpeg,image/png,image/gif,image/webp" onChange={(event) => void handleChange(event.target.files?.[0])} disabled={uploading} />{uploading ? <LoaderCircle size={14} className="spin" /> : <ImagePlus size={14} />}<span>{uploading ? '上传中' : '上传'}</span></label></div>
    {value && <img className="image-upload-preview" src={value} alt="封面预览" />}
    {error && <small className="image-upload-error">{error}</small>}
  </div>
}
