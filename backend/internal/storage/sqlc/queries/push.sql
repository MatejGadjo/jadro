-- name: UpsertPushSubscription :exec
INSERT INTO push_subscriptions (endpoint, p256dh, auth)
VALUES ($1, $2, $3)
ON CONFLICT (endpoint) DO UPDATE
SET p256dh = EXCLUDED.p256dh,
    auth   = EXCLUDED.auth;

-- name: ListPushSubscriptions :many
SELECT id, endpoint, p256dh, auth, created_at, last_sent_at
FROM push_subscriptions
ORDER BY id ASC;

-- name: DeletePushSubscriptionByEndpoint :exec
DELETE FROM push_subscriptions
WHERE endpoint = $1;

-- name: TouchPushSubscriptions :exec
UPDATE push_subscriptions
SET last_sent_at = now()
WHERE endpoint = ANY($1::text[]);

-- name: InsertPushSend :exec
INSERT INTO push_sends (cluster_id)
VALUES ($1);

-- name: ClusterSentRecently :one
SELECT EXISTS (
    SELECT 1 FROM push_sends
    WHERE cluster_id = $1
      AND sent_at > now() - make_interval(hours => $2::int4)
);
