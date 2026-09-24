// formatter 构造昂贵，列表页每行都会调用，提为模块级常量复用
const dateFormatter = new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })

export function formatDate(value: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value.replaceAll('-', '.')
  return dateFormatter.format(date).replaceAll('/', '.')
}
