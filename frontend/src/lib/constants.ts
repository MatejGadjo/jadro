// Number of clusters requested per feed page (matches backend pagination).
export const FEED_PAGE_SIZE = 10

// Number of articles shown in the homepage "Latest news" sidebar.
export const LATEST_ARTICLES_LIMIT = 5

// localStorage key marking the push soft-prompt as dismissed (so it isn't
// re-shown after the reader declines or subscribes).
export const PUSH_PROMPT_DISMISSED_KEY = 'jadro_push_dismissed'

// localStorage key marking the iOS "add to home screen" help as dismissed.
export const PUSH_IOS_DISMISSED_KEY = 'jadro_push_ios_dismissed'

// How long a reader must dwell before the soft push-prompt may appear, unless
// they engage sooner by opening a story. Avoids prompting on first paint.
export const PUSH_ENGAGE_DELAY_MS = 10_000
