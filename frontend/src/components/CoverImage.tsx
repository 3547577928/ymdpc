import { useEffect, useState } from 'react'

export function CoverImage({ src, title, className, loading }: { src: string; title: string; className?: string; loading?: 'eager' | 'lazy' }) {
  const [failed, setFailed] = useState(false)

  useEffect(() => setFailed(false), [src])

  if (!src || failed) {
    return <div className={`${className ?? ''} cover-placeholder`} role="img" aria-label={`${title}封面`}><div className="cover-art" aria-hidden="true"><i /><i /><i /></div><strong>QS</strong><span>Quiet Signal</span></div>
  }

  return <img className={className} src={src} alt={`${title}封面`} loading={loading} onError={() => setFailed(true)} />
}
