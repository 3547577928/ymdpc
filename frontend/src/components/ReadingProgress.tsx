import { useEffect, useState } from 'react'

// 阅读进度条独立维护滚动状态（rAF 节流）：进度状态挂在 PostPage 顶层时，
// 每次滚动事件都会重渲染整篇文章（含 ReactMarkdown 子树），长文滚动明显掉帧
export function ReadingProgress() {
  const [progress, setProgress] = useState(0)

  useEffect(() => {
    let frame = 0
    const update = () => {
      frame = 0
      const available = document.documentElement.scrollHeight - window.innerHeight
      setProgress(available > 0 ? Math.min(100, Math.max(0, (window.scrollY / available) * 100)) : 0)
    }
    const schedule = () => { if (!frame) frame = requestAnimationFrame(update) }
    update()
    window.addEventListener('scroll', schedule, { passive: true })
    window.addEventListener('resize', schedule)
    return () => {
      if (frame) cancelAnimationFrame(frame)
      window.removeEventListener('scroll', schedule)
      window.removeEventListener('resize', schedule)
    }
  }, [])

  return <div className="reading-progress" aria-hidden="true"><span style={{ width: `${progress}%` }} /></div>
}
