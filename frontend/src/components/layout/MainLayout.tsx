import { useLayoutEffect } from 'react'
import type { ReactNode } from 'react'
import { useRouterState } from '@tanstack/react-router'
import Header from './Header'
import Footer from './Footer'
import PushPromptBanner from '@/components/push/PushPromptBanner'
import IosInstallBanner from '@/components/push/IosInstallBanner'

interface MainLayoutProps {
  children: ReactNode
}

export default function MainLayout({ children }: MainLayoutProps) {
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  useLayoutEffect(() => {
    window.scrollTo({ top: 0, behavior: 'instant' })
  }, [pathname])

  return (
    <div className="min-h-screen bg-background flex flex-col">
      {/* Header - full width at top */}
      <Header />

      {/* Main content area */}
      <div className="container mx-auto px-3 pb-6 pt-24 flex-1">
        {/* Centered content column. NOTE: do not make <main> a grid/flex item
            with an auto inline-margin — that disables `justify-self: stretch`
            and collapses it to its content's max-content width (skeletons go
            narrow on desktop). If a side rail is added, wrap in a grid and
            keep `w-full` on <main> so stretch is preserved. */}
        <main className="mx-auto w-full max-w-6xl">
          {children}
        </main>
      </div>

      <Footer />

      <PushPromptBanner />
      <IosInstallBanner />
    </div>
  )
}
