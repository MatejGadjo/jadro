import { describe, expect, it } from 'vitest'
import { HOMEPAGE_TOPICS, TOPIC_ITEMS, isValidTopicSlug, topicLabelFromSlug, topicValueFromSlug } from '@/lib/topics'

describe('topics', () => {
  it('validates slugs against the known topic list', () => {
    expect(isValidTopicSlug('sport')).toBe(true)
    expect(isValidTopicSlug('sportsss')).toBe(false)
    expect(isValidTopicSlug(undefined)).toBe(false)
  })

  it('maps slugs to backend values and labels', () => {
    for (const t of TOPIC_ITEMS) {
      expect(topicValueFromSlug(t.id)).toBe(t.value)
      expect(topicLabelFromSlug(t.id)).toBe(t.label)
    }
    expect(topicValueFromSlug('unknown')).toBeUndefined()
    expect(topicLabelFromSlug('unknown')).toBe('unknown')
  })

  it('homepage topics are a subset of all topics', () => {
    const ids = new Set(TOPIC_ITEMS.map((t) => t.id))
    for (const t of HOMEPAGE_TOPICS) expect(ids.has(t.id)).toBe(true)
  })
})
