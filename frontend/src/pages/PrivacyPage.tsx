import { Link } from '@tanstack/react-router'
import PageMasthead from '@/components/content/PageMasthead'
import LegalSection from '@/components/content/LegalSection'

export default function PrivacyPage() {
  return (
    <div className="max-w-3xl mx-auto">
      <PageMasthead
        eyebrow="Правно"
        title="Политика на приватност"
        lede="jadro.mk е дизајниран да работи без да знае кој сте. Не бараме регистрација и не градиме профили на корисници."
        dateline="Последно ажурирано · јуни 2026"
      />

      <div className="space-y-6">
        <LegalSection index={1} title="Кои податоци ги собираме">
          <p>
            jadro.mk не бара регистрација и не собира лични податоци. Не складираме имиња,
            адреси на е-пошта ниту кориснички профили.
          </p>
        </LegalSection>

        <LegalSection index={2} title="Колачиња и статистика">
          <p>
            Не користиме рекламни или следачки колачиња. Може да користиме основна,
            анонимна статистика за да разбереме колку луѓе ја посетуваат платформата —
            овие податоци не ве идентификуваат лично.
          </p>
        </LegalSection>

        <LegalSection index={3} title="Линкови до трети страни">
          <p>
            Платформата содржи линкови до написи на надворешни медиуми. Кога ќе кликнете на
            некој напис, важат правилата за приватност на тој сајт, не нашите. Не одговараме
            за начинот на кој трети страни ги обработуваат вашите податоци.
          </p>
        </LegalSection>

        <LegalSection index={4} title="Промени на политиката">
          <p>
            Оваа политика може да се ажурира. Датумот на последна измена е наведен на врвот
            на страницата.
          </p>
        </LegalSection>

        <LegalSection index={5} title="Контакт">
          <p>
            За прашања во врска со приватноста, контактирајте не преку страницата{' '}
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
