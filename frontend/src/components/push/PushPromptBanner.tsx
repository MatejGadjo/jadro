import { useEffect, useState } from 'react';
import { useRouterState } from '@tanstack/react-router';
import { Bell, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { usePush } from '@/hooks/usePush';
import { trackEvent } from '@/analytics';
import { PUSH_ENGAGE_DELAY_MS, PUSH_PROMPT_DISMISSED_KEY } from '@/lib/constants';

function isDismissed(): boolean {
  try {
    return localStorage.getItem(PUSH_PROMPT_DISMISSED_KEY) === '1';
  } catch {
    return false;
  }
}

function markDismissed(): void {
  try {
    localStorage.setItem(PUSH_PROMPT_DISMISSED_KEY, '1');
  } catch {
    /* ignore storage errors (private mode) */
  }
}

// PushPromptBanner is the soft pre-prompt: a one-time, dismissible invitation to
// enable the daily notification. It only appears after the reader has engaged
// (opened a story, or dwelled past a delay) and never on first paint — clicking
// "Да" is what triggers the native permission prompt. Once decided (subscribed
// or declined) it never returns, gated by localStorage.
export default function PushPromptBanner() {
  const { supported, available, permission, subscribed, busy, subscribe, iosNeedsInstall } = usePush();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const [engaged, setEngaged] = useState(false);
  const [dismissed, setDismissed] = useState(isDismissed);
  const [closedNow, setClosedNow] = useState(false); // X: hide for this visit only
  const [show, setShow] = useState(false); // drives the slide-up transition

  // Engagement: opening any story is strong intent (fire next tick); otherwise a
  // dwell timer. setState happens only inside the timer callback so it never
  // runs synchronously during the effect.
  useEffect(() => {
    if (engaged) return;
    const delay = pathname.startsWith('/cluster/') ? 0 : PUSH_ENGAGE_DELAY_MS;
    const timer = window.setTimeout(() => setEngaged(true), delay);
    return () => window.clearTimeout(timer);
  }, [pathname, engaged]);

  const visible =
    supported &&
    available &&
    !iosNeedsInstall &&
    permission === 'default' &&
    !subscribed &&
    engaged &&
    !dismissed &&
    !closedNow;

  // Once visible, slide in on the next frame. The rAF keeps setState out of the
  // effect body (it starts off-screen at translate-y-full, then transitions to 0).
  useEffect(() => {
    if (!visible) return;
    trackEvent('push_prompt_shown');
    const id = requestAnimationFrame(() => setShow(true));
    return () => cancelAnimationFrame(id);
  }, [visible]);

  if (!visible) return null;

  const handleAccept = async () => {
    const ok = await subscribe();
    // Whether granted or denied, the soft prompt has done its job — don't nag.
    markDismissed();
    setDismissed(true);
    if (!ok) trackEvent('push_prompt_declined', { via: 'native' });
  };

  const handleDecline = () => {
    markDismissed();
    setDismissed(true);
    trackEvent('push_prompt_declined', { via: 'soft' });
  };

  // X = "not now": hide for this visit only (no persistence), so it returns the
  // next time the site is opened. Distinct from "Не, благодарам" (permanent).
  const handleClose = () => {
    setClosedNow(true);
    trackEvent('push_prompt_dismissed', { via: 'x' });
  };

  return (
    <div
      className={`fixed inset-x-0 bottom-0 z-50 px-3 pb-3 sm:px-4 sm:pb-5 transition-transform duration-300 ease-out ${
        show ? 'translate-y-0' : 'translate-y-full'
      }`}
    >
      <div className="relative mx-auto flex min-h-[25vh] w-full max-w-2xl flex-col justify-center gap-4 rounded-xl border-2 border-foreground bg-card p-5 shadow-2xl sm:min-h-0 sm:flex-row sm:items-center sm:gap-5 sm:p-6">
        <button
          type="button"
          onClick={handleClose}
          aria-label="Затвори"
          className="absolute right-2.5 top-2.5 cursor-pointer text-muted-foreground transition-colors duration-150 hover:text-foreground"
        >
          <X className="size-5" aria-hidden />
        </button>

        <div className="flex items-start gap-3 pr-8 sm:flex-1 sm:pr-0">
          <Bell className="size-7 shrink-0 text-editorial sm:size-9" aria-hidden />
          <div className="min-w-0">
            <p
              className="whitespace-nowrap text-xl font-bold text-foreground sm:text-2xl"
              style={{ fontFamily: 'var(--font-serif)' }}
            >
              Сакаш да си во тек?
            </p>
            <p className="mt-1 text-sm leading-relaxed text-muted-foreground sm:text-base">
              Една вест дневно — најважното од Македонија.
            </p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-3">
          <Button size="lg" className="flex-1 cursor-pointer sm:flex-none" onClick={handleAccept} disabled={busy}>
            Да, извести ме
          </Button>
          <Button
            size="lg"
            variant="outline"
            className="flex-1 cursor-pointer sm:flex-none"
            onClick={handleDecline}
          >
            Не, благодарам
          </Button>
        </div>
      </div>
    </div>
  );
}
