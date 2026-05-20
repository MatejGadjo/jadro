import { cn } from '@/lib/utils'

interface AdSlotProps {
  position: 'top-leaderboard' | 'feed-inline' | 'sidebar' | 'cluster-detail'
  index?: number
  className?: string
}

export default function AdSlot({ position, index, className }: AdSlotProps) {
  return (
    <div
      className={cn(
        "bg-muted border-2 border-dashed border-border rounded-lg p-6 text-center",
        className
      )}
    >
      <div className="text-sm font-medium text-muted-foreground">
        Ad Position: {position}
      </div>
      {index !== undefined && (
        <div className="text-xs text-muted-foreground mt-1">Index: {index}</div>
      )}
    </div>
  )
}
