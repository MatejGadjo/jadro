import type { Cluster } from '../../api/types'
import FeedCard from './FeedCard'

interface FeedListProps {
  clusters: Cluster[]
}

export default function FeedList({ clusters }: FeedListProps) {
  return (
    <div className="space-y-4">
      {clusters.map((cluster) => (
        <FeedCard key={cluster.id} cluster={cluster} />
      ))}
    </div>
  )
}
