import { ArrowUpRight, Code2, MessagesSquare, PenLine, Users } from 'lucide-react'
import { Link } from 'react-router-dom'
import { usePageMeta } from '../utils/usePageMeta'

// 构建信息由构建流程注入，见 vite.config.ts，用于确认线上运行的是哪一个版本
const buildMeta = [
  { label: '版本号', value: __APP_VERSION__ },
  { label: '打包时间', value: new Date(__BUILD_TIME__).toLocaleString('zh-CN', { hour12: false }) },
  { label: '运行平台', value: __BUILD_PLATFORM__ },
]

export function AboutPage() {
  usePageMeta('关于')
  return (
    <section className="container about-page">
      <div className="about-header"><div><div className="eyebrow">About the community</div><h1>这里是<br /><em>Quiet Signal。</em></h1></div></div>
      <div className="about-grid">
        <div className="about-lead"><p>Quiet Signal 是一个安静的写作社区：写下界面的细节、系统的取舍，也记录那些还没有答案的问题。</p><p>社区里的每篇文章都可以评论、点赞和收藏，也可以关注喜欢的作者；首页和文章页对访客开放阅读。</p></div>
        <div className="about-facts"><div><Users size={16} /><span>面向所有读者开放</span></div><div><MessagesSquare size={16} /><span>文章 · 评论 · 关注</span></div><div><Code2 size={16} /><span>React · Go · SQLite</span></div><div><PenLine size={16} /><span>写作优先，少一点噪音</span></div></div>
      </div>
      <div className="about-section"><div className="eyebrow">Community notes</div><h2>社区在意的事情</h2><div className="principles"><div><span>01</span><h3>清晰胜过聪明</h3><p>让下一个接手的人能够快速理解，比写出一个技巧性很强的实现更重要。</p></div><div><span>02</span><h3>保持完整闭环</h3><p>从数据到界面，从异常状态到部署，尽量把一个问题真正做完。</p></div><div><span>03</span><h3>给注意力留白</h3><p>减少不必要的通知、装饰和流程，把时间还给真正重要的事情。</p></div></div></div>
      <div className="about-cta"><div><div className="eyebrow">Join in</div><h2>想写点什么，或者看看别人在写什么？</h2></div><Link className="button button-dark" to="/posts">进入社区文章 <ArrowUpRight size={16} /></Link></div>
      <div className="about-build"><div className="eyebrow">Build info</div><h2>当前运行的版本</h2><div className="build-info-grid">{buildMeta.map((item) => <div key={item.label}><span>{item.label}</span><strong>{item.value}</strong></div>)}</div></div>
    </section>
  )
}