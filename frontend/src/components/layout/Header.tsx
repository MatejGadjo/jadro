import { Link } from '@tanstack/react-router'
import { Sun, Moon } from 'lucide-react'
import { Switch } from '@/components/ui/switch'
import CategoryNav from './CategoryNav'
import PushBell from '@/components/push/PushBell'
import { useState, useEffect } from 'react'

const MK_DAYS = ['Недела', 'Понеделник', 'Вторник', 'Среда', 'Четврток', 'Петок', 'Сабота']
const MK_MONTHS = [
  'Јануари', 'Февруари', 'Март', 'Април', 'Мај', 'Јуни',
  'Јули', 'Август', 'Септември', 'Октомври', 'Ноември', 'Декември',
]

function getMkDate() {
  const now = new Date()
  return `${MK_DAYS[now.getDay()]}, ${now.getDate()} ${MK_MONTHS[now.getMonth()]} ${now.getFullYear()}`
}

function HeaderMeta() {
  const mono: React.CSSProperties = {
    fontFamily:    'var(--font-mono)',
    fontSize:      '11px',
    letterSpacing: '0.06em',
  }

  return (
    <span
      className="hidden lg:flex items-center gap-0 text-foreground absolute left-1/2 -translate-x-1/2 pointer-events-none whitespace-nowrap"
      style={mono}
    >
      <span className="uppercase">{getMkDate()}</span>
      <PushBell
        showLabel
        leadingSeparator
        className="text-[11px] uppercase tracking-[0.06em] text-foreground pointer-events-auto"
      />
    </span>
  )
}

export default function Header() {
  const [isDarkMode, setIsDarkMode] = useState(false)

  useEffect(() => {
    const darkModePreference = localStorage.getItem('darkMode') === 'true'
    setIsDarkMode(darkModePreference)
    if (darkModePreference) {
      document.documentElement.classList.add('dark')
    }
  }, [])

  const toggleDarkMode = (checked: boolean) => {
    document.documentElement.classList.add('theme-transition')
    setIsDarkMode(checked)
    if (checked) {
      document.documentElement.classList.add('dark')
      localStorage.setItem('darkMode', 'true')
    } else {
      document.documentElement.classList.remove('dark')
      localStorage.setItem('darkMode', 'false')
    }
    setTimeout(() => document.documentElement.classList.remove('theme-transition'), 200)
  }

  return (
    <div className="fixed top-0 left-0 right-0 z-50 bg-card border-b-2 border-foreground">
      <div className="border-b border-border">
        <div className="container mx-auto px-4">
          <div className="flex items-center justify-between h-11 gap-4">

            {/* Wordmark */}
            <Link
              to="/"
              className="group flex items-baseline gap-2 shrink-0 select-none"
            >
              <span
                className="text-2xl font-black text-foreground leading-none group-hover:text-editorial transition-colors duration-150"
                style={{ fontFamily: 'var(--font-serif)', fontStyle: 'italic', letterSpacing: '-0.02em' }}
              >
                jadro
              </span>
              <span
                className="text-2xl font-black text-editorial leading-none"
                style={{ fontFamily: 'var(--font-serif)', fontStyle: 'italic', letterSpacing: '-0.02em' }}
              >
                .mk
              </span>
            </Link>

            {/* Center meta: date */}
            <HeaderMeta />

            {/* Controls */}
            <div className="flex items-center gap-2 shrink-0">
              {/* Mobile/tablet: icon-only bell + dot separator, left of the theme
                  toggle. Desktop shows the labelled bell in the centered meta. */}
              <span className="lg:hidden flex items-center">
                <PushBell trailingSeparator iconClassName="w-4 h-4" className="text-foreground" />
              </span>
              <Sun className="w-4 h-4 text-foreground" />
              <Switch
                checked={isDarkMode}
                onCheckedChange={toggleDarkMode}
                aria-label="Toggle dark mode"
                className="cursor-pointer data-[state=checked]:bg-foreground data-[state=unchecked]:bg-muted-foreground"
              />
              <Moon className="w-4 h-4 text-foreground" />
            </div>

          </div>
        </div>
      </div>

      <CategoryNav />
    </div>
  )
}
