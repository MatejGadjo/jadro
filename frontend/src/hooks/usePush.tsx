import { createContext, useCallback, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { fetchPushConfig, subscribeToPush, unsubscribeFromPush } from '../api/push';
import { trackEvent } from '../analytics';

export type PushPermission = 'default' | 'granted' | 'denied' | 'unsupported';

// Module-scoped cache so the VAPID key is fetched once per page load. Resolves to
// null when push is disabled on the backend (404) or the request fails.
let vapidKeyPromise: Promise<string | null> | null = null;

function loadVapidKey(): Promise<string | null> {
  if (!vapidKeyPromise) {
    vapidKeyPromise = fetchPushConfig()
      .then((c) => c.vapidPublicKey)
      .catch(() => null);
  }
  return vapidKeyPromise;
}

function isPushSupported(): boolean {
  return (
    typeof window !== 'undefined' &&
    'serviceWorker' in navigator &&
    'PushManager' in window &&
    'Notification' in window
  );
}

function isIOS(): boolean {
  if (typeof navigator === 'undefined') return false;
  const ua = navigator.userAgent;
  // iPadOS 13+ reports a Mac UA, so also treat a touch-capable "Mac" as iOS.
  return /iPhone|iPad|iPod/.test(ua) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);
}

function isStandalone(): boolean {
  if (typeof window === 'undefined') return false;
  return (
    window.matchMedia?.('(display-mode: standalone)').matches ||
    (navigator as { standalone?: boolean }).standalone === true
  );
}

// urlBase64ToUint8Array converts a base64url VAPID key to the byte array the
// PushManager.subscribe applicationServerKey expects.
function urlBase64ToUint8Array(base64: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64.length % 4)) % 4);
  const normalized = (base64 + padding).replace(/-/g, '+').replace(/_/g, '/');
  const raw = atob(normalized);
  const output = new Uint8Array(new ArrayBuffer(raw.length));
  for (let i = 0; i < raw.length; i += 1) {
    output[i] = raw.charCodeAt(i);
  }
  return output;
}

interface PushState {
  supported: boolean; // browser can do Web Push
  available: boolean; // backend push is enabled (VAPID key served)
  permission: PushPermission;
  subscribed: boolean;
  busy: boolean;
  subscribe: () => Promise<boolean>;
  unsubscribe: () => Promise<void>;
  // iOS: Safari can only do push after the site is installed to the home screen.
  // When true, show install instructions instead of the normal subscribe flow.
  iosNeedsInstall: boolean;
  iosHelpOpen: boolean; // bell-triggered open state for the iOS help popup
  showIosHelp: () => void;
  hideIosHelp: () => void;
}

const PushContext = createContext<PushState | null>(null);

// PushProvider owns the single source of truth for push state so every consumer
// (header bell, footer bell, soft prompt) stays in sync — subscribing in one
// place immediately updates the others, and hides the soft prompt.
export function PushProvider({ children }: { children: ReactNode }) {
  const supported = isPushSupported();
  const [permission, setPermission] = useState<PushPermission>(
    supported ? (Notification.permission as PushPermission) : 'unsupported',
  );
  const [subscribed, setSubscribed] = useState(false);
  const [available, setAvailable] = useState(false);
  const [busy, setBusy] = useState(false);
  const [iosHelpOpen, setIosHelpOpen] = useState(false);

  const iosNeedsInstall = isIOS() && !isStandalone();

  // One-time discovery: is the backend serving push (fetched even on iOS, so the
  // install prompt knows push is configured), and is this browser already
  // subscribed (only meaningful where push is supported)?
  useEffect(() => {
    let cancelled = false;
    void (async () => {
      const key = await loadVapidKey();
      if (cancelled) return;
      setAvailable(Boolean(key));
      if (!supported || Notification.permission === 'denied') return;
      try {
        const reg = await navigator.serviceWorker.ready;
        const sub = await reg.pushManager.getSubscription();
        if (!cancelled) setSubscribed(Boolean(sub));
      } catch {
        /* registration not ready — leave subscribed=false */
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [supported]);

  const showIosHelp = useCallback(() => setIosHelpOpen(true), []);
  const hideIosHelp = useCallback(() => setIosHelpOpen(false), []);

  const subscribe = useCallback(async (): Promise<boolean> => {
    if (!supported) return false;
    setBusy(true);
    try {
      const key = await loadVapidKey();
      if (!key) return false;

      const result = (await Notification.requestPermission()) as PushPermission;
      setPermission(result);
      if (result !== 'granted') {
        trackEvent('push_opt_out', { reason: result });
        return false;
      }

      const reg = await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(key),
      });
      await subscribeToPush(sub.toJSON() as PushSubscriptionJSON);
      setSubscribed(true);
      trackEvent('push_opt_in');
      return true;
    } catch {
      return false;
    } finally {
      setBusy(false);
    }
  }, [supported]);

  const unsubscribe = useCallback(async (): Promise<void> => {
    if (!supported) return;
    setBusy(true);
    try {
      const reg = await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.getSubscription();
      if (sub) {
        const { endpoint } = sub;
        await sub.unsubscribe();
        await unsubscribeFromPush(endpoint).catch(() => undefined);
      }
      setSubscribed(false);
      trackEvent('push_opt_out', { reason: 'manual' });
    } finally {
      setBusy(false);
    }
  }, [supported]);

  const value: PushState = {
    supported,
    available,
    permission,
    subscribed,
    busy,
    subscribe,
    unsubscribe,
    iosNeedsInstall,
    iosHelpOpen,
    showIosHelp,
    hideIosHelp,
  };
  return <PushContext.Provider value={value}>{children}</PushContext.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components -- hook co-located with its provider
export function usePush(): PushState {
  const ctx = useContext(PushContext);
  if (!ctx) throw new Error('usePush must be used within a PushProvider');
  return ctx;
}
