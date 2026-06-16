import { Link } from '@tanstack/react-router'
import PageMasthead from '@/components/content/PageMasthead'
import LegalSection from '@/components/content/LegalSection'

export default function TermsPage() {
  return (
    <div className="max-w-3xl mx-auto">
      <PageMasthead
        eyebrow="Правно"
        title="Услови за користење"
        lede="jadro.mk е агрегатор на македонски вести. Со користење на платформата се согласувате со следните услови."
        dateline="Последно ажурирано · јуни 2026"
      />

      <div className="space-y-6">
        <LegalSection index={1} title="Прифаќање на условите">
          <p>
            Со пристапување и користење на jadro.mk се согласувате со овие услови. Доколку не
            се согласувате, ве молиме не ја користете платформата.
          </p>
        </LegalSection>

        <LegalSection index={2} title="Природа на услугата">
          <p>
            jadro.mk ги групира и поврзува написите од трети медиуми. Не сме автори на нивната
            содржина и не одговараме за нејзината точност, целосност ниту навременост.
          </p>
        </LegalSection>

        <LegalSection index={3} title="Интелектуална сопственост">
          <p>
            Сите агрегирани написи, слики и заглавија им припаѓаат на изворните медиуми.
            jadro.mk не тврди авторство врз нив. Секој напис води до оригиналниот извор.
          </p>
        </LegalSection>

        <LegalSection index={4} title="Прифатлива употреба">
          <p>
            Забрането е користење на платформата за незаконски цели, автоматско преземање на
            содржина во голем обем или нарушување на правата на трети страни.
          </p>
        </LegalSection>

        <LegalSection index={5} title="Одрекување од одговорност">
          <p>
            Платформата се нуди „каква што е“. Не гарантираме дека ќе биде достапна без прекин,
            ниту дека секоја агрегирана вест е точна или целосна.
          </p>
        </LegalSection>

        <LegalSection index={6} title="Промени на условите">
          <p>
            Овие услови може да се изменат во секое време. Продолженото користење на платформата
            по измените значи прифаќање на новите услови.
          </p>
        </LegalSection>

        <LegalSection index={7} title="Контакт">
          <p>
            За прашања во врска со условите, контактирајте не преку страницата{' '}
            <Link to="/about" className="text-editorial hover:underline underline-offset-4">
              За нас
            </Link>
            .
          </p>
        </LegalSection>
      </div>
    </div>
  )
}
