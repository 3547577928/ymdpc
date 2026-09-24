import { Component, type ReactNode } from 'react'
import { Link } from 'react-router-dom'

type ErrorBoundaryState = { hasError: boolean }

// 渲染异常兜底：没有错误边界时，任何组件抛错都会让整站白屏
export class ErrorBoundary extends Component<{ children: ReactNode }, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false }

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true }
  }

  componentDidCatch(error: unknown) {
    console.error('render error:', error)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="container page-state">
          <span className="eyebrow">Error</span>
          <h1>页面出错了。</h1>
          <p>刷新页面通常能恢复；如果反复出现，请联系管理员。</p>
          <p>
            <button className="text-button" onClick={() => this.setState({ hasError: false })}>重试</button>{' '}
            <Link className="text-button" to="/">返回首页</Link>
          </p>
        </div>
      )
    }
    return this.props.children
  }
}
