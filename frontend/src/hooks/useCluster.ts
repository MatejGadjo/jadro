import { useQuery } from '@tanstack/react-query';
import { fetchCluster } from '../api/cluster';
import { queryKeys } from '../api/queryKeys';
import type { ClusterResponse } from '../api/types';

export function useCluster(clusterId: number | string) {
  const numericId = typeof clusterId === 'string' ? Number(clusterId) : clusterId;

  return useQuery<ClusterResponse>({
    queryKey: queryKeys.cluster(numericId),
    queryFn: () => fetchCluster(numericId),
    staleTime: 5 * 60 * 1000, // 5 minutes
    retry: 2,
  });
}
