import { useEffect, useState } from 'react';
import { useRouterState } from '@tanstack/react-router';
import { Bell, Share, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { usePush } from '@/hooks/usePush';
import { trackEvent } from '@/analytics';
import { PUSH_ENGAGE_DELAY_MS, PUSH_IOS_DISMISSED_KEY } from '@/lib/constants';

function isDismissed(): boolean {
  try {
    return localStorage.getItem(PUSH_IOS_DISMISSED_KEY) === '1';
  } catch {
    return false;
  }
}

function markDismissed(): void {
  try {
    localStorage.setItem(PUSH_IOS_DISMISSED_KEY, '1');
  } catch {
    /* ignore storage errors (private mode) */
  }
}

// IosInstallBanner is the iOS counterpart to PushPromptBanner: same look, but
// since Safari can't enable web push in a tab, it shows how to add the site to
// the home screen instead of a subscribe button. It auto-appears after the
// reader engages, and also opens on demand when the iOS bell is tapped
// (usePush().iosHelpOpen).
export default function IosInstallBanner() {
  const { iosNeedsInstall, available, iosHelpOpen, hideIosHelp } = usePush();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const [engaged, setEngaged] = useState(false);
  const [dismissed, setDismissed] = useState(isDismissed);
  const [closedNow, setClosedNow] = useState(false);
  const [show, setShow] = useState(false);

  useEffect(() => {
    if (engaged) return;
    const delay = pathname.startsWith('/cluster/') ? 0 : PUSH_ENGAGE_DELAY_MS;
    const timer = window.setTimeout(() => setEngaged(true), delay);
    return () => window.clearTimeout(timer);
  }, [pathname, engaged]);

  // Opens automatically after engagement, or any time the bell requests it.
  const visible =
    iosNeedsInstall && available && (iosHelpOpen || (engaged && !dismissed && !closedNow));

  useEffect(() => {
    if (!visible) return;
    trackEvent('push_ios_help_shown');
    const id = requestAnimationFrame(() => setShow(true));
    return () => cancelAnimationFrame(id);
  }, [visible]);

  if (!visible) return null;

  // X = not now (returns next visit). Re-arms the slide for the next open.
  const handleClose = () => {
    setShow(false);
    setClosedNow(true);
    hideIosHelp();
  };

  // "Сфатив" = don't auto-show again (the bell can still reopen it).
  const handleGotIt = () => {
    setShow(false);
    markDismissed();
    setDismissed(true);
    hideIosHelp();
  };

  return (
    <div
      className={`fixed inset-x-0 bottom-0 z-50 px-3 pb-3 sm:px-4 sm:pb-5 transition-transform duration-300 ease-out ${
        show ? 'translate-y-0' : 'translate-y-full'
      }`}
    >
      <div className="relative mx-auto flex min-h-[25vh] w-full max-w-2xl flex-col justify-center gap-4 rounded-xl border-2 border-foreground bg-card p-5 shadow-2xl sm:min-h-0 sm:p-6">
        <button
          type="button"
          onClick={handleClose}
          aria-label="Затвори"
          className="absolute right-2.5 top-2.5 cursor-pointer text-muted-foreground transition-colors duration-150 hover:text-foreground"
        >
          <X className="size-5" aria-hidden />
        </button>

        <div className="flex items-start gap-3 pr-8">
          <Bell className="size-7 shrink-0 text-editorial sm:size-9" aria-hidden />
          <div className="min-w-0">
            <p
              className="text-xl font-bold text-foreground sm:text-2xl"
              style={{ fontFamily: 'var(--font-serif)' }}
            >
              Извести ме на iPhone
            </p>
            <p className="mt-1 text-sm leading-relaxed text-muted-foreground sm:text-base">
              За известувања на iPhone, прво додај го Jadro на почетниот екран:
            </p>
            <ol className="mt-3 space-y-1.5 text-sm text-foreground sm:text-base">
              <li className="flex items-center gap-2">
                <span className="font-semibold text-editorial">1.</span>
                <span className="inline-flex items-center gap-1">
                  Притисни <Share className="size-4" aria-hidden /> „Share“ во Safari
                </span>
              </li>
              <li className="flex items-center gap-2">
                <span className="font-semibold text-editorial">2.</span>
                <span>Избери „Add to Home Screen“</span>
              </li>
              <li className="flex items-center gap-2">
                <span className="font-semibold text-editorial">3.</span>
                <span>Отвори го Jadro од почетниот екран и дозволи известувања</span>
              </li>
            </ol>
          </div>
        </div>

        <div className="flex shrink-0 items-center justify-end">
          <Button size="lg" className="w-full cursor-pointer sm:w-auto" onClick={handleGotIt}>
            Сфатив
          </Button>
        </div>
      </div>
    </div>
  );
}
