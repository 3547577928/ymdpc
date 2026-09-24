// 骨架屏：替代此前「大字标题闪替」的加载态，页面切换不再整片空白
export function SkeletonPostGrid({ count = 6 }: { count?: number }) {
  return (
    <div className="post-grid" aria-hidden="true">
      {Array.from({ length: count }, (_, index) => (
        <div className="skeleton-card" key={index}>
          <div className="skeleton skeleton-cover" />
          <div className="skeleton skeleton-line" style={{ width: '42%' }} />
          <div className="skeleton skeleton-line" style={{ width: '88%' }} />
          <div className="skeleton skeleton-line" style={{ width: '64%' }} />
        </div>
      ))}
    </div>
  )
}

export function SkeletonList({ count = 5 }: { count?: number }) {
  return (
    <div aria-hidden="true">
      {Array.from({ length: count }, (_, index) => (
        <div className="skeleton-row" key={index}>
          <div className="skeleton skeleton-avatar" />
          <div className="skeleton-row-lines">
            <div className="skeleton skeleton-line" style={{ width: '55%' }} />
            <div className="skeleton skeleton-line" style={{ width: '80%' }} />
          </div>
        </div>
      ))}
    </div>
  )
}
