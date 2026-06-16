// Shared helpers for rendering source badges/avatars consistently across the
// feed, topic cards, and cluster detail views.

export const SOURCE_COLORS = [
  'bg-red-700',
  'bg-blue-700',
  'bg-emerald-700',
  'bg-violet-700',
  'bg-amber-700',
  'bg-cyan-700',
  'bg-rose-700',
]

export function getSourceInitials(name: string): string {
  return name
    .split(' ')
    .map((word) => word[0])
    .join('')
    .substring(0, 2)
    .toUpperCase()
}

export function getSourceColor(name: string): string {
  return SOURCE_COLORS[name.length % SOURCE_COLORS.length]
}
