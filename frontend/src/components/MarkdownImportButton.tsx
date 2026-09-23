import { FileUp, LoaderCircle } from 'lucide-react'
import { useState } from 'react'
import { readMarkdownFile, type ImportedMarkdown } from '../utils/markdownImport'

type MarkdownImportButtonProps = {
  disabled?: boolean
  onImport: (document: ImportedMarkdown) => void
  onError: (message: string) => void
}

export function MarkdownImportButton({ disabled, onImport, onError }: MarkdownImportButtonProps) {
  const [reading, setReading] = useState(false)

  const handleFile = async (file?: File) => {
    if (!file) return
    setReading(true)
    onError('')
    try {
      onImport(await readMarkdownFile(file))
    } catch (reason) {
      onError(reason instanceof Error ? reason.message : '读取 Markdown 文件失败')
    } finally {
      setReading(false)
    }
  }

  return <label className="markdown-import-button">
    <input type="file" accept=".md,.markdown,text/markdown" disabled={disabled || reading} onChange={(event) => { void handleFile(event.target.files?.[0]); event.target.value = '' }} />
    {reading ? <LoaderCircle size={14} className="spin" /> : <FileUp size={14} />}
    <span>{reading ? '读取中' : '导入 Markdown'}</span>
  </label>
}
