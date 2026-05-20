export interface MockArticle {
  id: number
  source: {
    id: number
    name: string
    logo_url?: string
  }
  title: string
  url: string
  publishedAt: string
  author?: string
  imageUrl?: string
}

export interface MockCluster {
  id: number
  headline: string
  displayTitle?: string
  articles: MockArticle[]
  topic: string
}

export const mockClusters: MockCluster[] = [
  {
    id: 1,
    headline: "Minnesota killing produces backlash against Trump administration from Second Amendment advocates",
    topic: "US News",
    articles: [
      {
        id: 1,
        source: { id: 1, name: "AP News" },
        title: "Minnesota killing produces backlash against Trump administration from Second Amendment advocates",
        url: "#",
        publishedAt: "3 hours ago",
        author: "Bill Barrow & Nicholas Riccardi",
        imageUrl: "https://images.unsplash.com/photo-1585829365295-ab7cd400c167?w=400&h=250&fit=crop"
      },
      {
        id: 2,
        source: { id: 2, name: "The New York Times" },
        title: "How We Determined That Minneapolis Videos Contradicted Federal Officials",
        url: "#",
        publishedAt: "12 hours ago",
        author: "Mark Scheffler"
      },
      {
        id: 3,
        source: { id: 3, name: "NPR" },
        title: "Minneapolis killings put a focus on use of body cameras",
        url: "#",
        publishedAt: "1 hour ago",
        author: "Ximena Bustillo"
      },
      {
        id: 4,
        source: { id: 4, name: "BBC" },
        title: "'We're all terrified', Minnesotan tells the BBC in tears",
        url: "#",
        publishedAt: "13 hours ago"
      }
    ]
  },
  {
    id: 2,
    headline: "Tom Homan dispatched to Minnesota amid immigration operation fallout",
    topic: "Politics",
    articles: [
      {
        id: 5,
        source: { id: 5, name: "ABC News" },
        title: "Border Patrol Commander Greg Bovino returning to Minnesota weeks after deadly shooting",
        url: "#",
        publishedAt: "2 hours ago",
        imageUrl: "https://images.unsplash.com/photo-1529107386315-e1a2ed48a620?w=400&h=250&fit=crop"
      },
      {
        id: 6,
        source: { id: 4, name: "BBC" },
        title: "US immigration chief Gregory Bovino set to leave Minneapolis after deadly shooting",
        url: "#",
        publishedAt: "2 hours ago",
        author: "Ana Faguy"
      },
      {
        id: 7,
        source: { id: 6, name: "The Guardian" },
        title: "Bovino's future in doubt as White House walks back initial claims about Alex Pretti – US politics live",
        url: "#",
        publishedAt: "40 minutes ago",
        author: "Vivian Ho"
      },
      {
        id: 8,
        source: { id: 7, name: "CNN" },
        title: "Immigration enforcement operation in Minneapolis draws scrutiny",
        url: "#",
        publishedAt: "5 hours ago"
      }
    ]
  },
  {
    id: 3,
    headline: "Supreme Court to hear arguments on social media content moderation",
    topic: "Technology",
    articles: [
      {
        id: 9,
        source: { id: 8, name: "Reuters" },
        title: "Supreme Court weighs state laws on social media content",
        url: "#",
        publishedAt: "6 hours ago",
        author: "Lawrence Hurley",
        imageUrl: "https://images.unsplash.com/photo-1551836022-d5d88e9218df?w=400&h=250&fit=crop"
      },
      {
        id: 10,
        source: { id: 2, name: "The New York Times" },
        title: "Tech Giants Face Crucial Test in Content Moderation Case",
        url: "#",
        publishedAt: "8 hours ago",
        author: "Adam Liptak"
      },
      {
        id: 11,
        source: { id: 9, name: "The Washington Post" },
        title: "First Amendment battles over online speech reach high court",
        url: "#",
        publishedAt: "7 hours ago",
        author: "Ann E. Marimow"
      }
    ]
  },
  {
    id: 4,
    headline: "Global markets rally as inflation concerns ease",
    topic: "Business",
    articles: [
      {
        id: 12,
        source: { id: 10, name: "Bloomberg" },
        title: "Stocks surge on softer-than-expected inflation data",
        url: "#",
        publishedAt: "4 hours ago",
        author: "Sarah Ponczek",
        imageUrl: "https://images.unsplash.com/photo-1611974789855-9c2a0a7236a3?w=400&h=250&fit=crop"
      },
      {
        id: 13,
        source: { id: 11, name: "CNBC" },
        title: "Dow jumps 400 points as investors cheer cooling inflation",
        url: "#",
        publishedAt: "5 hours ago",
        author: "Jesse Pound"
      },
      {
        id: 14,
        source: { id: 12, name: "Financial Times" },
        title: "Central banks hint at potential rate cuts amid easing price pressures",
        url: "#",
        publishedAt: "6 hours ago",
        author: "Kate Duguid"
      }
    ]
  }
]
