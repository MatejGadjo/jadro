import { Fragment, useMemo } from 'react'
import { Newspaper } from 'lucide-react'
import StoryCluster from '@/components/feed/StoryCluster'
import HomePageSkeleton from '@/components/feed/HomePageSkeleton'
import MobileTopStoriesFeed from '@/components/feed/MobileTopStoriesFeed'
import YourTopics from '@/components/topics/YourTopics'
import HomeSidebar from '@/components/layout/HomeSidebar'
import { Card } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import EmptyState from '@/components/ui/EmptyState'
import { useFeed } from '@/hooks/useFeed'
import { useIsMobile } from '@/hooks/useIsMobile'
import { mapFeaturedSectionsToStoryClusters, mapCategorySectionsToTopicSections } from '@/lib/homepageData'

export default function HomePage() {
  const { data, isLoading, error } = useFeed('', 0)
  const isMobile = useIsMobile()
  const featuredClusters = mapFeaturedSectionsToStoryClusters(data?.featured_clusters ?? [])
  const topicSections = mapCategorySectionsToTopicSections(data?.top_clusters_by_category ?? [])
  const featuredClusterIds = useMemo(
    () => new Set(featuredClusters.map((c) => c.id)),
    [featuredClusters]
  )

  return (
    <div className="space-y-6">
      {/* Top Stories Section */}
      <div className="mt-2">
        <div className="flex items-end gap-3 border-b-2 border-foreground pb-3 mb-4">
          <div className="w-1 self-stretch bg-editorial shrink-0" style={{ minHeight: '2.5rem' }} />
          <div>
            <h2
              className="text-3xl md:text-4xl font-black text-foreground leading-none"
              style={{ fontFamily: 'var(--font-serif)', fontStyle: 'italic' }}
            >
              Топ вести
            </h2>
            <p
              className="text-xs text-muted-foreground mt-1 uppercase tracking-widest"
              style={{ fontFamily: 'var(--font-mono)', fontSize: '10px' }}
            >
              Најважни вести денес
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 items-start">
          <div className="lg:col-span-2">
            <Card className="bg-card border-border gap-0">
              {isLoading && <HomePageSkeleton />}

              {error && (
                <div className="p-4 text-sm text-destructive">
                  Неуспешно вчитување на водечки вести.
                </div>
              )}

              {!isLoading && !error && featuredClusters.length === 0 && (
                <EmptyState
                  icon={Newspaper}
                  heading="Нема водечки вести"
                  subtext="Моментално нема достапни водечки вести."
                />
              )}

              {!isLoading && !error && featuredClusters.map((cluster, index) => (
                <Fragment key={cluster.id}>
                  {index > 0 && <Separator />}
                  <div className="p-4">
                    <StoryCluster cluster={cluster} featured={true} embedded={true} />
                  </div>
                </Fragment>
              ))}
            </Card>
          </div>

          <HomeSidebar />
        </div>
      </div>

      {/* Mobile: extra top stories beyond featured */}
      {isMobile && !isLoading && !error && (
        <MobileTopStoriesFeed
          featuredCount={featuredClusters.length}
          featuredClusterIds={featuredClusterIds}
        />
      )}

      {/* Your Topics Section */}
      <div className="hidden md:block">
        {error ? (
          <div className="rounded-lg border border-border bg-card p-6 text-sm text-destructive">
            Неуспешно вчитување на написи за темата.
          </div>
        ) : (
          <YourTopics sections={topicSections} />
        )}
      </div>

    </div>
  )
}
