import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { Shell } from './components/Shell'
import { AboutPage } from './pages/AboutPage'
import { AdminPage } from './pages/AdminPage'
import { AuthPage } from './pages/AuthPage'
import { FavoritesPage } from './pages/FavoritesPage'
import { HomePage } from './pages/HomePage'
import { MyPostsPage } from './pages/MyPostsPage'
import { NotificationsPage } from './pages/NotificationsPage'
import { PostPage } from './pages/PostPage'
import { PostsPage } from './pages/PostsPage'
import { ProfilePage } from './pages/ProfilePage'
import { ProfileSettingsPage } from './pages/ProfileSettingsPage'
import { WritePage } from './pages/WritePage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Shell><Routes><Route path="/" element={<HomePage />} /><Route path="/posts" element={<PostsPage />} /><Route path="/posts/:slug" element={<PostPage />} /><Route path="/posts/:id/edit" element={<WritePage />} /><Route path="/about" element={<AboutPage />} /><Route path="/write" element={<WritePage />} /><Route path="/me/posts" element={<MyPostsPage />} /><Route path="/me/favorites" element={<FavoritesPage />} /><Route path="/notifications" element={<NotificationsPage />} /><Route path="/users/:username" element={<ProfilePage />} /><Route path="/settings/profile" element={<ProfileSettingsPage />} /><Route path="*" element={<div className="container empty-state post-not-found"><h1>页面不存在</h1></div>} /></Routes></Shell>} path="/*" />
        <Route path="/login" element={<AuthPage />} />
        <Route path="/register" element={<AuthPage />} />
        <Route path="/admin" element={<AdminPage />} />
      </Routes>
    </BrowserRouter>
  )
}
