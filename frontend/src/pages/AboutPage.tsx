import { ArrowUpRight, Code2, Figma, Github, Mail, MapPin, Sparkles } from 'lucide-react'

export function AboutPage() {
  return (
    <section className="container about-page">
      <div className="about-header"><div><div className="eyebrow">A little about me</div><h1>你好，我是<br /><em>Yiming。</em></h1></div><div className="about-avatar"><span>YM</span><div className="avatar-line avatar-line-one" /><div className="avatar-line avatar-line-two" /></div></div>
      <div className="about-grid">
        <div className="about-lead"><p>我是一名独立开发者和产品设计师，喜欢把模糊的问题拆成清晰的系统，也喜欢把系统重新写成让人愿意使用的界面。</p><p>目前主要使用 React、Go 和 SQLite 做一些小而完整的产品。这个博客是工作台，也是一个持续更新的个人档案。</p><div className="about-links"><a href="mailto:hello@quietsig.dev"><Mail size={16} /> hello@quietsig.dev</a><a href="https://github.com" target="_blank" rel="noreferrer"><Github size={16} /> github.com/yiming</a></div></div>
        <div className="about-facts"><div><MapPin size={16} /><span>Shanghai / China</span></div><div><Sparkles size={16} /><span>专注于工具型产品</span></div><div><Code2 size={16} /><span>React · Go · SQL</span></div><div><Figma size={16} /><span>设计和工程并行</span></div></div>
      </div>
      <div className="about-section"><div className="eyebrow">Working notes</div><h2>我在意的事情</h2><div className="principles"><div><span>01</span><h3>清晰胜过聪明</h3><p>让下一个接手的人能够快速理解，比写出一个技巧性很强的实现更重要。</p></div><div><span>02</span><h3>保持完整闭环</h3><p>从数据到界面，从异常状态到部署，尽量把一个问题真正做完。</p></div><div><span>03</span><h3>给注意力留白</h3><p>减少不必要的通知、装饰和流程，把时间还给真正重要的事情。</p></div></div></div>
      <div className="about-cta"><div><div className="eyebrow">Say hello</div><h2>有一个问题，或者只是想聊聊？</h2></div><a className="button button-dark" href="mailto:hello@quietsig.dev">发一封邮件 <ArrowUpRight size={16} /></a></div>
    </section>
  )
}
