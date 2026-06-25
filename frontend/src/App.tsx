import { RouterProvider, createRouter } from '@tanstack/react-router'
import { routeTree } from './routes'
import { PushProvider } from './hooks/usePush'

const router = createRouter({
  routeTree,
})

function App() {
  return (
    <PushProvider>
      <RouterProvider router={router} />
    </PushProvider>
  )
}

export default App
