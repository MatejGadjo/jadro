import { useQuery } from '@tanstack/react-query';
import axios from 'axios';
import { fetchCluster } from '../api/cluster';
import { queryKeys } from '../api/queryKeys';
import type { ClusterResponse } from '../api/types';

export function useCluster(clusterId: number | string) {
  const numericId = typeof clusterId === 'string' ? Number(clusterId) : clusterId;

  return useQuery<ClusterResponse>({
    queryKey: queryKeys.cluster(numericId),
    queryFn: () => fetchCluster(numericId),
    staleTime: 5 * 60 * 1000, // 5 minutes
    // Don't retry client errors (e.g. a 404 for a missing cluster) — they won't
    // succeed on retry and would only delay showing the 404 page.
    retry: (failureCount, error) => {
      if (axios.isAxiosError(error) && error.response && error.response.status < 500) {
        return false;
      }
      return failureCount < 2;
    },
  });
}
