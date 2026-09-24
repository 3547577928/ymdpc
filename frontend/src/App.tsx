import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { lazy, Suspense } from 'react'
import { Shell } from './components/Shell'
import { AuthProvider } from './services/auth'
import { ErrorBoundary } from './components/ErrorBoundary'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from './services/queries'

// 路由级代码分割：首屏只下载当前页 chunk，管理后台、编辑器与 Markdown 渲染链
// （react-markdown/micromark）不再对普通访客全量下发
const AboutPage = lazy(() => import('./pages/AboutPage').then((m) => ({ default: m.AboutPage })))
const AdminPage = lazy(() => import('./pages/AdminPage').then((m) => ({ default: m.AdminPage })))
const AuthPage = lazy(() => import('./pages/AuthPage').then((m) => ({ default: m.AuthPage })))
const FavoritesPage = lazy(() => import('./pages/FavoritesPage').then((m) => ({ default: m.FavoritesPage })))
const ForumPage = lazy(() => import('./pages/ForumPage').then((m) => ({ default: m.ForumPage })))
const ForumTopicPage = lazy(() => import('./pages/ForumTopicPage').then((m) => ({ default: m.ForumTopicPage })))
const HomePage = lazy(() => import('./pages/HomePage').then((m) => ({ default: m.HomePage })))
const MyPostsPage = lazy(() => import('./pages/MyPostsPage').then((m) => ({ default: m.MyPostsPage })))
const NotificationsPage = lazy(() => import('./pages/NotificationsPage').then((m) => ({ default: m.NotificationsPage })))
const PostPage = lazy(() => import('./pages/PostPage').then((m) => ({ default: m.PostPage })))
const PostsPage = lazy(() => import('./pages/PostsPage').then((m) => ({ default: m.PostsPage })))
const ProfilePage = lazy(() => import('./pages/ProfilePage').then((m) => ({ default: m.ProfilePage })))
const ProfileSettingsPage = lazy(() => import('./pages/ProfileSettingsPage').then((m) => ({ default: m.ProfileSettingsPage })))
const SearchPage = lazy(() => import('./pages/SearchPage').then((m) => ({ default: m.SearchPage })))
const TagPage = lazy(() => import('./pages/TagPage').then((m) => ({ default: m.TagPage })))
const MyStatsPage = lazy(() => import('./pages/MyStatsPage').then((m) => ({ default: m.MyStatsPage })))
const SeriesPage = lazy(() => import('./pages/SeriesPage').then((m) => ({ default: m.SeriesPage })))
const WritePage = lazy(() => import('./pages/WritePage').then((m) => ({ default: m.WritePage })))

// 社区内容页统一挂在 Shell 布局（页头导航 + 页脚）下；
// 登录、注册、后台没有导航需求，使用各自的独立布局
const shellPages = [
  { path: '/', element: <HomePage /> },
  { path: '/posts', element: <PostsPage /> },
  { path: '/posts/:slug', element: <PostPage /> },
  { path: '/posts/:id/edit', element: <WritePage /> },
  { path: '/forum', element: <ForumPage /> },
  { path: '/forum/:id', element: <ForumTopicPage /> },
  { path: '/write', element: <WritePage /> },
  { path: '/about', element: <AboutPage /> },
  { path: '/me/posts', element: <MyPostsPage /> },
  { path: '/me/favorites', element: <FavoritesPage /> },
  { path: '/me/stats', element: <MyStatsPage /> },
  { path: '/notifications', element: <NotificationsPage /> },
  { path: '/users/:username', element: <ProfilePage /> },
  { path: '/settings/profile', element: <ProfileSettingsPage /> },
  { path: '/search', element: <SearchPage /> },
  { path: '/tags/:slug', element: <TagPage /> },
  { path: '/series/:slug', element: <SeriesPage /> },
]

const pageFallback = <div className="container page-state"><span className="eyebrow">Loading</span><h1>正在加载。</h1></div>

export default function App() {
  return (
    <BrowserRouter>
      <QueryClientProvider client={queryClient}>
      <AuthProvider>
      <Routes>
        <Route
          path="/*"
          element={
            <Shell>
              <ErrorBoundary>
              <Suspense fallback={pageFallback}>
                <Routes>
                  {shellPages.map(({ path, element }) => (
                    <Route key={path} path={path} element={element} />
                  ))}
                  <Route path="*" element={<div className="container empty-state post-not-found"><h1>页面不存在</h1></div>} />
                </Routes>
              </Suspense>
              </ErrorBoundary>
            </Shell>
          }
        />
        <Route path="/login" element={<ErrorBoundary><Suspense fallback={pageFallback}><AuthPage /></Suspense></ErrorBoundary>} />
        <Route path="/register" element={<ErrorBoundary><Suspense fallback={pageFallback}><AuthPage /></Suspense></ErrorBoundary>} />
        <Route path="/admin" element={<ErrorBoundary><Suspense fallback={pageFallback}><AdminPage /></Suspense></ErrorBoundary>} />
      </Routes>
      </AuthProvider>
      </QueryClientProvider>
    </BrowserRouter>
  )
}
