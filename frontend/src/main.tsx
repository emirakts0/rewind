import ReactDOM from 'react-dom/client'
import App from './App'
import { NotificationApp } from './NotificationApp'
import { ThemeProvider } from '@/components/theme-provider'
import './index.css'

const isNotificationWindow = window.location.hash.includes('notification')

ReactDOM.createRoot(document.getElementById('root')!).render(
    <ThemeProvider defaultTheme="dark" storageKey="rewind-ui-theme">
        {isNotificationWindow ? <NotificationApp /> : <App />}
    </ThemeProvider>,
)
