import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import * as TooltipPrimitive from '@radix-ui/react-tooltip'
import './index.css'
import App from './App.tsx'
import { ToastProvider } from './components/ui'
import { KeyboardNavProvider } from './keyboard'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <TooltipPrimitive.Provider delayDuration={0}>
      <ToastProvider>
        <KeyboardNavProvider>
          <App />
        </KeyboardNavProvider>
      </ToastProvider>
    </TooltipPrimitive.Provider>
  </StrictMode>,
)
