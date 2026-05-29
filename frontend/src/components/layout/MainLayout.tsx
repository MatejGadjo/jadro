import { useLayoutEffect } from 'react'
import type { ReactNode } from 'react'
import { useRouterState } from '@tanstack/react-router'
import Header from './Header'
import Footer from './Footer'

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
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          {/* Main content - full width on mobile, center column on desktop */}
          <main className="lg:col-span-12 lg:max-w-6xl lg:mx-auto">
            {children}
          </main>

          {/* Ad Sidebar - hidden on mobile, visible on desktop (right) */}
          {/* <aside className="hidden lg:block lg:col-span-2">
            <AdSidebar />
          </aside> */}
        </div>
      </div>

      <Footer />
    </div>
  )
}
