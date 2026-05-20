import client from './client';
import { FeedResponseSchema } from './schemas';
import type { FeedResponse } from './types';

interface FetchFeedParams {
  topic?: string;
  page?: number;
}

export async function fetchFeed(params?: FetchFeedParams): Promise<FeedResponse> {
  const response = await client.get('/api/feed', {
    params: {
      topic: params?.topic,
      page: params?.page,
    },
  });

  const validatedData = FeedResponseSchema.parse(response.data);
  return validatedData;
}
