import { useQuery } from '@tanstack/react-query'
import { fetchLatestArticles } from '../api/cluster'
import { queryKeys } from '../api/queryKeys'
import type { LatestArticlesResponse } from '../api/types'

export function useLatestArticles(limit = 8) {
  return useQuery<LatestArticlesResponse>({
    queryKey: queryKeys.latestArticles(limit),
    queryFn: () => fetchLatestArticles(limit),
    staleTime: 2 * 60 * 1000,
    retry: 2,
  })
}
