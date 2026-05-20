import { useQuery, keepPreviousData } from '@tanstack/react-query';
import { useRef } from 'react';
import { fetchFeed } from '../api/feed';
import { queryKeys } from '../api/queryKeys';
import type { FeedResponse } from '../api/types';

type VersionCache = { key: string; version: number; data: FeedResponse };

export function useFeed(topic: string, page: number, enabled = true) {
  const prevRef = useRef<VersionCache | null>(null);
  const queryKey = `${topic}:${page}`;

  return useQuery<FeedResponse>({
    queryKey: queryKeys.feed(topic, page),
    queryFn: () => fetchFeed({ topic, page }),
    enabled,
    staleTime: 5 * 60 * 1000,
    retry: 2,
    placeholderData: keepPreviousData,
    select: (data) => {
      const version = data.feed_version ?? 0;
      if (
        prevRef.current &&
        prevRef.current.key === queryKey &&
        version !== 0 &&
        version === prevRef.current.version
      ) {
        return prevRef.current.data;
      }
      prevRef.current = { key: queryKey, version, data };
      return data;
    },
  });
}
