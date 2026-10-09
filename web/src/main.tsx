import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import * as TooltipPrimitive from '@radix-ui/react-tooltip'
import './index.css'
import App from './App.tsx'
import { ToastProvider } from './components/ui'
import { KeyboardNavProvider } from './keyboard'
import { QueryClientProvider } from '@tanstack/react-query'
import { createQueryClient } from './api/queries'

const queryClient = createQueryClient()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <TooltipPrimitive.Provider delayDuration={0}>
      <ToastProvider>
        <KeyboardNavProvider>
          <QueryClientProvider client={queryClient}>
            <App />
          </QueryClientProvider>
        </KeyboardNavProvider>
      </ToastProvider>
    </TooltipPrimitive.Provider>
  </StrictMode>,
)
