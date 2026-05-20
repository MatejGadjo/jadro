export interface TopicArticle {
  id: number
  clusterId?: number
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

export interface TopicSection {
  id: string
  name: string
  articles: TopicArticle[]
}

export const topicSections: TopicSection[] = [
  {
    id: 'us',
    name: 'U.S.',
    articles: [
      {
        id: 1,
        source: { id: 1, name: 'The Conversation' },
        title: "Shooting of Alex Pretti in Minneapolis has put America's gun lobby at odds with the White House",
        url: '#',
        publishedAt: '19 hours ago',
        author: 'Andrew Gawthorpe',
        imageUrl: 'https://images.unsplash.com/photo-1585829365295-ab7cd400c167?w=200&h=120&fit=crop'
      },
      {
        id: 2,
        source: { id: 2, name: 'Council on Foreign Relations' },
        title: "The Trump Administration's Cyber Strategy Fundamentally Misunderstands China's Threat",
        url: '#',
        publishedAt: '14 hours ago',
        imageUrl: 'https://images.unsplash.com/photo-1523961131990-5ea7c61b2107?w=200&h=120&fit=crop'
      },
      {
        id: 3,
        source: { id: 3, name: 'The New York Times' },
        title: 'Winter Storm Dumps a Foot of Snow in at Least 19 U.S. States: See the Snow Totals',
        url: '#',
        publishedAt: '11 hours ago',
        author: 'Andy Newman',
        imageUrl: 'https://images.unsplash.com/photo-1547754980-3df97fed72a8?w=200&h=120&fit=crop'
      }
    ]
  },
  {
    id: 'technology',
    name: 'Technology',
    articles: [
      {
        id: 4,
        source: { id: 4, name: 'MacRumors' },
        title: 'iPhone 5S Gets New Software Update 13 Years After Launch',
        url: '#',
        publishedAt: '11 hours ago',
        author: 'Juli Clover',
        imageUrl: 'https://images.unsplash.com/photo-1510557880182-3d4d3cba35a5?w=200&h=120&fit=crop'
      },
      {
        id: 5,
        source: { id: 5, name: 'SamMobile' },
        title: "The Galaxy S24 Ultra is so sharp it's tearing through people's pants",
        url: '#',
        publishedAt: '23 hours ago',
        author: 'Mihai Matei',
        imageUrl: 'https://images.unsplash.com/photo-1610945415295-d9bbf067e59c?w=200&h=120&fit=crop'
      },
      {
        id: 6,
        source: { id: 6, name: 'Sammy Fans' },
        title: 'Samsung ends up leaking the Galaxy Buds 4 and Buds 4 Pro',
        url: '#',
        publishedAt: 'Yesterday',
        imageUrl: 'https://images.unsplash.com/photo-1590658268037-6bf12165a8df?w=200&h=120&fit=crop'
      }
    ]
  },
  {
    id: 'business',
    name: 'Business',
    articles: [
      {
        id: 7,
        source: { id: 7, name: 'CNBC' },
        title: 'UnitedHealth posts modest earnings beat, soft revenue guidance as insurer plots turnaround',
        url: '#',
        publishedAt: '36 minutes ago',
        author: 'Annika Kim Constantino',
        imageUrl: 'https://images.unsplash.com/photo-1576091160399-112ba8d25d1d?w=200&h=120&fit=crop'
      },
      {
        id: 8,
        source: { id: 8, name: 'Bloomberg' },
        title: 'Insurers Sink on US Plan to Keep Medicare Payment Rates Flat',
        url: '#',
        publishedAt: '50 minutes ago',
        author: 'John Tozzi',
        imageUrl: 'https://images.unsplash.com/photo-1454165804606-c3d57bc86b40?w=200&h=120&fit=crop'
      }
    ]
  },
  {
    id: 'finance',
    name: 'Finance',
    articles: [
      {
        id: 9,
        source: { id: 9, name: 'CBS News' },
        title: 'Some states expand ACA subsidies as federal tax credits lapse',
        url: '#',
        publishedAt: '12 hours ago',
        author: 'Mary Cunningham',
        imageUrl: 'https://images.unsplash.com/photo-1579621970563-ebec7560ff3e?w=200&h=120&fit=crop'
      }
    ]
  },
  {
    id: 'tv',
    name: 'TV',
    articles: [
      {
        id: 10,
        source: { id: 10, name: 'HuffPost' },
        title: "Stephen Colbert Brings Down The House With 2 Scathing Words About Trump Admin",
        url: '#',
        publishedAt: '1 hour ago',
        author: 'Edward Mazza',
        imageUrl: 'https://images.unsplash.com/photo-1522869635100-9f4c5e86aa37?w=200&h=120&fit=crop'
      }
    ]
  },
  {
    id: 'internet-security',
    name: 'Internet security',
    articles: [
      {
        id: 11,
        source: { id: 11, name: 'Fox News' },
        title: "Chinese hackers reportedly breached phones at 'heart of Downing Street' in global spy operation",
        url: '#',
        publishedAt: '11 hours ago',
        author: 'Emma Bussey',
        imageUrl: 'https://images.unsplash.com/photo-1550751827-4bd374c3f58b?w=200&h=120&fit=crop'
      }
    ]
  }
]

export interface StockData {
  name: string
  value: string
  change: string
  changePercent: string
  isPositive: boolean
}

export const stockMarketData: StockData[] = [
  {
    name: 'Dow Jones',
    value: '49,412.40',
    change: '+0.64%',
    changePercent: '+0.64%',
    isPositive: true
  },
  {
    name: 'S&P 500',
    value: '6,950.23',
    change: '+0.50%',
    changePercent: '+0.50%',
    isPositive: true
  },
  {
    name: 'Nasdaq',
    value: '23,601.36',
    change: '+0.43%',
    changePercent: '+0.43%',
    isPositive: true
  }
]
