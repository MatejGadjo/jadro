import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { getSourceColor, getSourceInitials } from '@/lib/source'

interface SourceAvatarProps {
  source: { name: string; logo_url?: string }
  /** Sizing/ring classes for the avatar wrapper. */
  className?: string
  /** Extra classes for the initials fallback (e.g. text size). */
  fallbackClassName?: string
}

/**
 * Square source logo with a colored-initials fallback. Centralizes the
 * avatar + fallback markup that was duplicated across feed/cluster views.
 */
export function SourceAvatar({ source, className, fallbackClassName }: SourceAvatarProps) {
  return (
    <Avatar className={className}>
      <AvatarImage
        src={source.logo_url}
        alt={source.name}
        className="rounded-sm bg-card object-contain p-0.5"
      />
      <AvatarFallback
        className={`rounded-sm font-bold text-white ${getSourceColor(source.name)} ${fallbackClassName ?? ''}`}
      >
        {getSourceInitials(source.name)}
      </AvatarFallback>
    </Avatar>
  )
}

export default SourceAvatar
