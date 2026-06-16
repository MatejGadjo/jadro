import { Link } from '@tanstack/react-router'
import PageMasthead from '@/components/content/PageMasthead'

const PIPELINE = [
  {
    step: '01',
    title: 'Собираме',
    body: 'Континуирано ги следиме објавите на македонските медиуми во реално време.',
  },
  {
    step: '02',
    title: 'Групираме',
    body: 'Алгоритам препознава кога повеќе извори известуваат за истиот настан и ги спојува во еден кластер.',
  },
  {
    step: '03',
    title: 'Прикажуваме',
    body: 'Го гледате настанот од повеќе агли, со линк до секој оригинален напис.',
  },
] as const

const PRINCIPLES = [
  {
    label: 'Доверба пред кликови',
    body: 'Не гониме сензации. Целта е јасна слика на денот, не повеќе кликови.',
  },
  {
    label: 'Повеќе агли',
    body: 'Еден настан, повеќе извори. Вие одлучувате кому ќе верувате.',
  },
  {
    label: 'Содржината им припаѓа на медиумите',
    body: 'Секој напис води до оригиналниот извор. Авторството останува нивно.',
  },
] as const

export default function AboutPage() {
  return (
    <div className="max-w-3xl mx-auto">
      <PageMasthead
        eyebrow="За платформата"
        title="Повеќе извори, една слика"
        lede="jadro.mk ги собира вестите од македонските медиуми и ги групира сличните написи заедно — за да ја видите целата слика на еден настан, а не само едно заглавие."
      />

      <div className="space-y-12">
        {/* Што е jadro.mk */}
        <section>
          <p
            className="text-[10px] uppercase tracking-[0.28em] text-muted-foreground mb-3"
            style={{ fontFamily: 'var(--font-mono)' }}
          >
            Што е jadro.mk
          </p>
          <p className="text-[15px] leading-relaxed text-foreground/85">
            jadro.mk е агрегатор на вести. Не пишуваме сопствени написи и немаме редакција.
            Наместо тоа, ги читаме објавите од повеќе извори, препознаваме кога различни
            медиуми известуваат за истиот настан и ги поврзуваме во еден кластер.
          </p>
        </section>

        {/* Како функционира — genuine pipeline, so the numbering is real */}
        <section>
          <p
            className="text-[10px] uppercase tracking-[0.28em] text-muted-foreground mb-5"
            style={{ fontFamily: 'var(--font-mono)' }}
          >
            Како функционира
          </p>
          <div className="grid gap-px bg-border border border-border sm:grid-cols-3">
            {PIPELINE.map(({ step, title, body }) => (
              <div key={step} className="bg-card p-5">
                <span
                  className="text-2xl font-black text-editorial tabular-nums"
                  style={{ fontFamily: 'var(--font-mono)' }}
                >
                  {step}
                </span>
                <h3
                  className="mt-3 text-lg font-bold text-foreground"
                  style={{ fontFamily: 'var(--font-serif)' }}
                >
                  {title}
                </h3>
                <p className="mt-1 text-sm leading-relaxed text-foreground/80">{body}</p>
              </div>
            ))}
          </div>
        </section>

        {/* Што веруваме */}
        <section>
          <p
            className="text-[10px] uppercase tracking-[0.28em] text-muted-foreground mb-5"
            style={{ fontFamily: 'var(--font-mono)' }}
          >
            Што веруваме
          </p>
          <div className="space-y-5">
            {PRINCIPLES.map(({ label, body }) => (
              <div key={label} className="border-l-2 border-editorial pl-4">
                <h3
                  className="text-lg font-bold text-foreground"
                  style={{ fontFamily: 'var(--font-serif)' }}
                >
                  {label}
                </h3>
                <p className="mt-1 text-[15px] leading-relaxed text-foreground/85">{body}</p>
              </div>
            ))}
          </div>
        </section>

        {/* Контакт */}
        <section className="border-t-2 border-foreground pt-6">
          <p
            className="text-[10px] uppercase tracking-[0.28em] text-muted-foreground mb-3"
            style={{ fontFamily: 'var(--font-mono)' }}
          >
            Контакт
          </p>
          <p className="text-[15px] leading-relaxed text-foreground/85">
            За соработка, исправки или прашања, пишете ни на{' '}
            <a
              href="mailto:kontakt@jadro.mk"
              className="text-editorial hover:underline underline-offset-4"
              style={{ fontFamily: 'var(--font-mono)' }}
            >
              kontakt@jadro.mk
            </a>
            . Прочитајте ги и нашите{' '}
            <Link to="/terms" className="text-editorial hover:underline underline-offset-4">
              услови
            </Link>{' '}
            и{' '}
            <Link to="/privacy" className="text-editorial hover:underline underline-offset-4">
              политика на приватност
            </Link>
            .
          </p>
        </section>
      </div>
    </div>
  )
}
