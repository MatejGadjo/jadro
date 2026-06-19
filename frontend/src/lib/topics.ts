export const TOPIC_ITEMS = [
  { id: 'makedonija', label: 'Македонија', value: 'Македонија' },
  { id: 'hronika', label: 'Хроника', value: 'Хроника' },
  { id: 'svet', label: 'Свет', value: 'Свет' },
  { id: 'ekonomija', label: 'Економија', value: 'Економија' },
  { id: 'sport', label: 'Спорт', value: 'Спорт' },
  { id: 'skopje', label: 'Скопје', value: 'Скопје' },
  { id: 'region', label: 'Регион', value: 'Регион' },
  { id: 'tehnologija', label: 'Технологија', value: 'Технологија' },
  { id: 'kultura', label: 'Култура', value: 'Култура' },
  { id: 'scena', label: 'Сцена', value: 'Сцена' },
] as const

export const HOMEPAGE_TOPIC_IDS = [
  'makedonija',
  'hronika',
  'svet',
  'ekonomija',
  'sport',
  'region',
] as const

const HOMEPAGE_TOPIC_ID_SET = new Set<string>(HOMEPAGE_TOPIC_IDS)

export const HOMEPAGE_TOPICS = TOPIC_ITEMS.filter((topic) => (
  HOMEPAGE_TOPIC_ID_SET.has(topic.id)
))

export type TopicSlug = (typeof TOPIC_ITEMS)[number]['id']

export function topicValueFromSlug(slug?: string): string | undefined {
  if (!slug) {
    return undefined
  }
  return TOPIC_ITEMS.find((topic) => topic.id === slug)?.value
}

export function topicLabelFromSlug(slug?: string): string {
  if (!slug) {
    return 'Тема'
  }
  return TOPIC_ITEMS.find((topic) => topic.id === slug)?.label ?? slug
}

// Maps a backend topic name (Cyrillic `value`/`label`) back to its latin slug
// so analytics events stay consistent with `topic_select` (which sends the slug).
// Falls back to the raw input when unknown.
export function topicSlugFromValue(value?: string): string | undefined {
  if (!value) {
    return undefined
  }
  return TOPIC_ITEMS.find(
    (topic) => topic.value === value || topic.label === value,
  )?.id ?? value
}
