import { SearchX } from 'lucide-react'
import EmptyState from '@/components/ui/EmptyState'

export default function FeedEmpty() {
  return (
    <EmptyState
      icon={SearchX}
      heading="Нема резултати"
      subtext="Нема пронајдени групи за оваа тема."
    />
  )
}
