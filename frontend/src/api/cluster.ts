import client from './client';
import { ClusterResponseSchema, LatestArticlesResponseSchema } from './schemas';
import type { ClusterResponse, LatestArticlesResponse } from './types';

export async function fetchCluster(clusterId: number): Promise<ClusterResponse> {
  const response = await client.get(`/api/clusters/${clusterId}`);
  return ClusterResponseSchema.parse(response.data);
}

export async function fetchLatestArticles(limit = 8): Promise<LatestArticlesResponse> {
  const response = await client.get('/api/articles/latest', { params: { limit } });
  return LatestArticlesResponseSchema.parse(response.data);
}

export function clearArticleImage(articleId: number): void {
  client.patch(`/api/articles/${articleId}/clear-image`).catch(() => undefined);
}
