import { useEffect } from 'react'
import { NotificationContainer } from '@/components/notifications'

export function NotificationApp() {
    useEffect(() => {
        // Tag the body so CSS can target it
        document.body.setAttribute('data-notification-window', 'true')

        // Force transparent backgrounds
        document.body.style.setProperty('background-color', 'transparent', 'important')
        document.body.style.setProperty('background', 'transparent', 'important')
        document.documentElement.style.setProperty('background-color', 'transparent', 'important')
        document.documentElement.style.setProperty('background', 'transparent', 'important')

        const root = document.getElementById('root')
        if (root) {
            root.style.setProperty('background-color', 'transparent', 'important')
            root.style.setProperty('background', 'transparent', 'important')
        }
    }, [])

    return (
        <div className="w-screen h-screen bg-transparent pointer-events-none overflow-hidden text-foreground">
            <NotificationContainer />
        </div>
    )
}
