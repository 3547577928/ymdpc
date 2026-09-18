import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { Shell } from './components/Shell'
import { AboutPage } from './pages/AboutPage'
import { AdminPage } from './pages/AdminPage'
import { HomePage } from './pages/HomePage'
import { PostPage } from './pages/PostPage'
import { PostsPage } from './pages/PostsPage'

export default function App() {
  return <BrowserRouter><Routes><Route element={<Shell><Routes><Route path="/" element={<HomePage />} /><Route path="/posts" element={<PostsPage />} /><Route path="/posts/:slug" element={<PostPage />} /><Route path="/about" element={<AboutPage />} /></Routes></Shell>} path="/*" /><Route path="/admin" element={<AdminPage />} /></Routes></BrowserRouter>
}
