import { useEffect } from 'react'

const SITE_NAME = 'Quiet Signal'
const DEFAULT_TITLE = 'Quiet Signal · 写作社区'
const DEFAULT_DESCRIPTION = '一个安静的写作社区：写下界面的细节、系统的取舍，以及还没有答案的问题。'

function setMeta(selector: string, attr: string, value: string) {
  document.querySelector(`meta[${selector}]`)?.setAttribute(attr, value)
}

// usePageMeta 按路由设置 <title> 与 description/OG：纯 CSR SPA 所有路由曾共享
// index.html 里同一个静态 <title>，搜索引擎与分享卡片对每篇文章只能拿到站点名片
export function usePageMeta(title?: string, description?: string) {
  useEffect(() => {
    const fullTitle = title ? `${title} · ${SITE_NAME}` : DEFAULT_TITLE
    const desc = description?.trim() || DEFAULT_DESCRIPTION
    document.title = fullTitle
    setMeta('name="description"', 'content', desc)
    setMeta('property="og:title"', 'content', fullTitle)
    setMeta('property="og:description"', 'content', desc)
    return () => {
      document.title = DEFAULT_TITLE
      setMeta('name="description"', 'content', DEFAULT_DESCRIPTION)
      setMeta('property="og:title"', 'content', DEFAULT_TITLE)
      setMeta('property="og:description"', 'content', DEFAULT_DESCRIPTION)
    }
  }, [title, description])
}
