import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App.tsx'

// Nothing in this app relies on react-query re-fetching to pick up
// server-side changes made elsewhere — every mutation updates the editor's
// Zustand store directly (setTrack/patchLine) and no query is ever
// invalidated. Refetch-on-focus was actively harmful here: switching tabs
// and back re-fetched the track, handed EditorPage a new object reference,
// and its effect called setTrack() again — which resets `showRomanized`
// (Asli/Romaji toggle) back to false every time. See editorStore.ts.
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      refetchOnReconnect: false,
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
)
