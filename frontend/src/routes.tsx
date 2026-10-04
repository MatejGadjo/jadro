import { createRootRoute, createRoute, Outlet } from '@tanstack/react-router'
import HomePage from './pages/HomePage'
import TopicPage from './pages/TopicPage'
import ClusterPage from './pages/ClusterPage'
import AboutPage from './pages/AboutPage'
import PrivacyPage from './pages/PrivacyPage'
import TermsPage from './pages/TermsPage'
import MainLayout from './components/layout/MainLayout'
import { NotFound } from './components/NotFound'

// Root route
const rootRoute = createRootRoute()

// Layout route - wraps all main pages with MainLayout
const layoutRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'layout',
  component: () => (
    <MainLayout>
      <Outlet />
    </MainLayout>
  ),
  // Renders the branded 404 (inside MainLayout) for unmatched paths and any
  // `notFound()` thrown by a child route.
  notFoundComponent: () => <NotFound />,
})

// Index route - shows HomePage
const indexRoute = createRoute({
  getParentRoute: () => layoutRoute,
  path: '/',
  component: HomePage,
})

// Topic route - /$topic
const topicRoute = createRoute({
  getParentRoute: () => layoutRoute,
  path: '/$topic',
  component: TopicPage,
})

// Cluster route - /cluster/$clusterId
const clusterRoute = createRoute({
  getParentRoute: () => layoutRoute,
  path: '/cluster/$clusterId',
  component: ClusterPage,
})

// About route - /about
const aboutRoute = createRoute({
  getParentRoute: () => layoutRoute,
  path: '/about',
  component: AboutPage,
})

// Privacy route - /privacy
const privacyRoute = createRoute({
  getParentRoute: () => layoutRoute,
  path: '/privacy',
  component: PrivacyPage,
})

// Terms route - /terms
const termsRoute = createRoute({
  getParentRoute: () => layoutRoute,
  path: '/terms',
  component: TermsPage,
})

// Route tree
export const routeTree = rootRoute.addChildren([
  layoutRoute.addChildren([indexRoute, topicRoute, clusterRoute, aboutRoute, privacyRoute, termsRoute]),
])
