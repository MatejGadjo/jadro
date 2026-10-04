import { useParams } from '@tanstack/react-router'
import FeedPage from './FeedPage'
import { NotFound } from '@/components/NotFound'
import { isValidTopicSlug } from '@/lib/topics'

// TopicRoute renders the branded 404 for unknown slugs (e.g. /sss, /hronikasss)
// instead of querying the feed for an invalid topic and rendering an empty page
// (soft-404). The check is done in the component — rendering NotFound directly
// is reliable, unlike a notFound() thrown from beforeLoad which TanStack renders
// against the matched route and here left the page blank. Validates against the
// local topic list only — no remote fetch.
export default function TopicPage() {
  const { topic } = useParams({ strict: false })
  if (!isValidTopicSlug(topic)) {
    return <NotFound />
  }
  return <FeedPage />
}
