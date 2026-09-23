import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { Shell } from './components/Shell'
import { AboutPage } from './pages/AboutPage'
import { AdminPage } from './pages/AdminPage'
import { AuthPage } from './pages/AuthPage'
import { MagicLinkPage } from './pages/MagicLinkPage'
import { FavoritesPage } from './pages/FavoritesPage'
import { HomePage } from './pages/HomePage'
import { MyPostsPage } from './pages/MyPostsPage'
import { NotificationsPage } from './pages/NotificationsPage'
import { PostPage } from './pages/PostPage'
import { PostsPage } from './pages/PostsPage'
import { ProfilePage } from './pages/ProfilePage'
import { ProfileSettingsPage } from './pages/ProfileSettingsPage'
import { WritePage } from './pages/WritePage'

// 社区内容页统一挂在 Shell 布局（页头导航 + 页脚）下；
// 登录、注册、后台没有导航需求，使用各自的独立布局
const shellPages = [
  { path: '/', element: <HomePage /> },
  { path: '/posts', element: <PostsPage /> },
  { path: '/posts/:slug', element: <PostPage /> },
  { path: '/posts/:id/edit', element: <WritePage /> },
  { path: '/write', element: <WritePage /> },
  { path: '/about', element: <AboutPage /> },
  { path: '/me/posts', element: <MyPostsPage /> },
  { path: '/me/favorites', element: <FavoritesPage /> },
  { path: '/notifications', element: <NotificationsPage /> },
  { path: '/users/:username', element: <ProfilePage /> },
  { path: '/settings/profile', element: <ProfileSettingsPage /> },
]

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route
          path="/*"
          element={
            <Shell>
              <Routes>
                {shellPages.map(({ path, element }) => (
                  <Route key={path} path={path} element={element} />
                ))}
                <Route path="*" element={<div className="container empty-state post-not-found"><h1>页面不存在</h1></div>} />
              </Routes>
            </Shell>
          }
        />
        <Route path="/login" element={<AuthPage />} />
        <Route path="/register" element={<AuthPage />} />
        <Route path="/auth/magic-link" element={<MagicLinkPage />} />
        <Route path="/admin" element={<AdminPage />} />
      </Routes>
    </BrowserRouter>
  )
}
