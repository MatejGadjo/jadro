import { z } from 'zod';
import client from './client';

const PushConfigSchema = z.object({
  vapidPublicKey: z.string(),
});

export type PushConfig = z.infer<typeof PushConfigSchema>;

// fetchPushConfig returns the VAPID public key the browser needs to subscribe.
// The backend responds 404 when push is disabled (no keys) — callers treat a
// rejection as "push unavailable".
export async function fetchPushConfig(): Promise<PushConfig> {
  const response = await client.get('/api/push/config');
  return PushConfigSchema.parse(response.data);
}

// subscribeToPush persists a browser subscription. The PushSubscription JSON
// shape ({ endpoint, keys: { p256dh, auth } }) matches the backend contract.
export async function subscribeToPush(sub: PushSubscriptionJSON): Promise<void> {
  await client.post('/api/push/subscribe', sub);
}

export async function unsubscribeFromPush(endpoint: string): Promise<void> {
  await client.post('/api/push/unsubscribe', { endpoint });
}
