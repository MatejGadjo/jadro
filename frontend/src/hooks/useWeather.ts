import { useQuery } from '@tanstack/react-query'

export interface WeatherData {
  temperature: number
  weatherCode: number
}

async function fetchWeather(): Promise<WeatherData> {
  const res = await fetch(
    'https://api.open-meteo.com/v1/forecast?latitude=41.9981&longitude=21.4254&current=temperature_2m,weather_code&timezone=Europe/Skopje'
  )
  if (!res.ok) throw new Error('Weather fetch failed')
  const json = await res.json()
  return {
    temperature: Math.round(json.current.temperature_2m),
    weatherCode: json.current.weather_code,
  }
}

export function useWeather() {
  return useQuery({
    queryKey: ['weather', 'skopje'],
    queryFn: fetchWeather,
    staleTime: 15 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
    retry: 2,
  })
}
