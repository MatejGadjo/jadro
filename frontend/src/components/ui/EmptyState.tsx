import type { LucideIcon } from 'lucide-react'

interface EmptyStateProps {
  icon: LucideIcon
  heading: string
  subtext: string
}

export default function EmptyState({ icon: Icon, heading, subtext }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-4">
      <div className="flex items-center gap-3 mb-10 w-full max-w-[200px]">
        <div className="flex-1 h-px bg-border" />
        <div className="w-1.5 h-1.5 bg-editorial rounded-full shrink-0" />
        <div className="flex-1 h-px bg-border" />
      </div>

      <div className="mb-5 text-muted-foreground/30">
        <Icon size={44} strokeWidth={1} />
      </div>

      <h3
        className="text-xl font-bold text-foreground mb-2 text-center leading-tight"
        style={{ fontFamily: 'var(--font-serif)', fontStyle: 'italic' }}
      >
        {heading}
      </h3>

      <p
        className="text-center text-muted-foreground max-w-[240px]"
        style={{ fontFamily: 'var(--font-mono)', fontSize: '11px', letterSpacing: '0.04em', lineHeight: '1.6' }}
      >
        {subtext}
      </p>

      <div className="flex items-center gap-3 mt-10 w-full max-w-[200px]">
        <div className="flex-1 h-px bg-border" />
        <div className="w-1.5 h-1.5 bg-editorial rounded-full shrink-0" />
        <div className="flex-1 h-px bg-border" />
      </div>
    </div>
  )
}
