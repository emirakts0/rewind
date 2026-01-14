import { Minus, X } from 'lucide-react'
import { Window } from '@wailsio/runtime'

interface TitleBarProps {
    title?: string
    children?: React.ReactNode
}

export function TitleBar({ title = "Rewind", children }: TitleBarProps) {
    const handleMinimize = () => {
        Window.Minimise()
    }

    const handleClose = () => {
        Window.Close()
    }

    return (
        <div className="title-bar">
            <div className="title-bar-drag">
                <div className="title-bar-icon">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
                        <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="2" className="text-primary" />
                        <path d="M8 12L11 15L16 9" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-primary" />
                    </svg>
                </div>
                <span className="title-bar-title">{title}</span>
            </div>

            {/* Extra content slot (for status badge, etc.) */}
            {children && <div className="title-bar-content">{children}</div>}

            <div className="title-bar-controls">
                <button
                    onClick={handleMinimize}
                    className="title-bar-btn title-bar-btn-minimize"
                    aria-label="Minimize"
                >
                    <Minus className="w-3.5 h-3.5" />
                </button>
                <button
                    onClick={handleClose}
                    className="title-bar-btn title-bar-btn-close"
                    aria-label="Close"
                >
                    <X className="w-3.5 h-3.5" />
                </button>
            </div>
        </div>
    )
}
