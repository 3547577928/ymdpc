import { Plus } from 'lucide-react'
import { useEffect, useState } from 'react'
import { createCategory } from '../services/api'
import type { Category } from '../types'

type CategoryFieldProps = {
  categories: Category[]
  value?: number | null
  onChange: (categoryId: number | null) => void
  onError: (message: string) => void
}

export function CategoryField({ categories, value, onChange, onError }: CategoryFieldProps) {
  const [items, setItems] = useState(categories)
  const [name, setName] = useState('')
  const [creating, setCreating] = useState(false)

  useEffect(() => {
    setItems(categories)
  }, [categories])

  const addCategory = async () => {
    const trimmed = name.trim()
    if (!trimmed || creating) return
    setCreating(true)
    onError('')
    try {
      const category = await createCategory(trimmed)
      setItems((current) => [...current.filter((item) => item.id !== category.id), category].sort((a, b) => a.name.localeCompare(b.name, 'zh-CN')))
      onChange(category.id)
      setName('')
    } catch (reason) {
      onError(reason instanceof Error ? reason.message : '创建分类失败')
    } finally {
      setCreating(false)
    }
  }

  return <div className="category-field">
    <select value={value ?? ''} onChange={(event) => onChange(event.target.value ? Number(event.target.value) : null)}>
      <option value="">不设分类</option>
      {items.map((category) => <option key={category.id} value={category.id}>{category.name}</option>)}
    </select>
    <div className="category-create-row">
      <input maxLength={80} value={name} onChange={(event) => setName(event.target.value)} onKeyDown={(event) => { if (event.key === 'Enter') { event.preventDefault(); void addCategory() } }} placeholder="新建分类" />
      <button type="button" className="icon-button" onClick={() => void addCategory()} disabled={!name.trim() || creating} aria-label="创建并选择分类"><Plus size={15} /></button>
    </div>
  </div>
}
