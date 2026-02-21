import ReactDOM from 'react-dom/client'
import App from './App'
import { ThemeProvider } from '@/components/theme-provider'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
    <ThemeProvider defaultTheme="dark" storageKey="rewind-ui-theme">
        <App />
    </ThemeProvider>,
)
