import { createRootRoute, createRoute, Outlet, useParams } from '@tanstack/react-router'
import HomePage from './pages/HomePage'
import FeedPage from './pages/FeedPage'
import ClusterPage from './pages/ClusterPage'
import AboutPage from './pages/AboutPage'
import PrivacyPage from './pages/PrivacyPage'
import TermsPage from './pages/TermsPage'
import MainLayout from './components/layout/MainLayout'
import { NotFound } from './components/NotFound'
import { isValidTopicSlug } from './lib/topics'

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
  component: TopicRoute,
})

// TopicRoute renders the branded 404 for unknown slugs (e.g. /sss, /hronikasss)
// instead of querying the feed for an invalid topic and rendering an empty page
// (soft-404). The check is done in the component — rendering NotFound directly
// is reliable, unlike a notFound() thrown from beforeLoad which TanStack renders
// against the matched route and here left the page blank. Validates against the
// local topic list only — no remote fetch.
function TopicRoute() {
  const { topic } = useParams({ strict: false })
  if (!isValidTopicSlug(topic)) {
    return <NotFound />
  }
  return <FeedPage />
}

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
