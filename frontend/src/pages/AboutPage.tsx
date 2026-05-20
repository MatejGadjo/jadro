export default function AboutPage() {
  return (
    <div className="space-y-6 max-w-3xl">
      <h1 className="text-3xl font-bold text-foreground">About</h1>
      <div className="prose">
        <p className="text-foreground leading-relaxed">
          NewsAggregator is a modern news aggregation platform that groups similar articles
          into clusters, making it easier to discover and compare news from multiple sources.
        </p>
        <p className="text-foreground leading-relaxed">
          Our platform uses advanced clustering algorithms to identify related articles,
          helping you get a comprehensive view of the news from different perspectives.
        </p>
      </div>
    </div>
  )
}

