import { Bell, BellOff } from 'lucide-react';
import { usePush } from '@/hooks/usePush';

interface PushBellProps {
  /** Render the text label next to the bell. Icon-only when false. */
  showLabel?: boolean;
  /** Render a "·" separator before the bell (for inline placement next to other text). */
  leadingSeparator?: boolean;
  /** Render a "·" separator after the bell (e.g. before the theme toggle on mobile). */
  trailingSeparator?: boolean;
  /** Classes for the button (font-size + color). Defaults to the footer mono link style. */
  className?: string;
  /** Classes for the icon (size). */
  iconClassName?: string;
}

function Separator({ className = 'px-2' }: { className?: string }) {
  return (
    <span className={`${className} text-muted-foreground/40`} style={{ fontFamily: 'var(--font-mono)' }} aria-hidden>
      ·
    </span>
  );
}

// PushBell is the persistent push opt-in/out control, reused in the header and
// footer. It renders only when push is usable: either the browser supports it
// and the backend has it enabled, or it's an iOS browser that needs the
// home-screen install first (in which case the click opens the iOS help popup).
// Hard-denied notifications hide it (no in-page way back).
export default function PushBell({
  showLabel = false,
  leadingSeparator = false,
  trailingSeparator = false,
  className,
  iconClassName,
}: PushBellProps) {
  const {
    supported,
    available,
    permission,
    subscribed,
    busy,
    subscribe,
    unsubscribe,
    iosNeedsInstall,
    showIosHelp,
  } = usePush();

  const iosMode = iosNeedsInstall && available;
  const canPush = supported && available && permission !== 'denied' && !iosNeedsInstall;

  if (!canPush && !iosMode) return null;

  // In iOS mode push can't be toggled in-browser — the click explains how to
  // install instead. The icon stays a plain bell (never "off").
  const label = iosMode ? 'Извести ме' : subscribed ? 'Исклучи известувања' : 'Извести ме';
  const Icon = !iosMode && subscribed ? BellOff : Bell;
  const onClick = iosMode ? showIosHelp : () => (subscribed ? void unsubscribe() : void subscribe());

  return (
    <span className="inline-flex items-center">
      {leadingSeparator && <Separator />}
      <button
        type="button"
        onClick={onClick}
        disabled={!iosMode && busy}
        aria-label={label}
        title={label}
        className={`flex w-fit cursor-pointer items-center gap-1.5 transition-colors duration-150 hover:text-editorial disabled:opacity-50 ${className ?? 'text-xs text-muted-foreground'}`}
        style={{ fontFamily: 'var(--font-mono)' }}
      >
        <Icon className={iconClassName ?? 'size-3.5'} aria-hidden />
        {showLabel && label}
      </button>
      {/* Trailing dot uses left-only padding; the parent's gap supplies the right
          side, so the dot sits centered between the bell and the next control. */}
      {trailingSeparator && <Separator className="pl-2" />}
    </span>
  );
}
