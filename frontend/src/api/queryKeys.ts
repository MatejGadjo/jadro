export const queryKeys = {
  feed: (topic: string, page: number) => ['feed', topic, page] as const,
  cluster: (clusterId: number) => ['cluster', clusterId] as const,
  latestArticles: (limit: number) => ['articles', 'latest', limit] as const,
} as const;
