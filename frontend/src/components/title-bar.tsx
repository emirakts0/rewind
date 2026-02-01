import { Minus, X } from 'lucide-react'
import { Window } from '@wailsio/runtime'
import packageJson from '../../package.json'

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
                    <img src="/icon.png" alt="Rewind" className="w-6 h-6 object-contain" />
                </div>
                <div className="flex items-baseline">
                    <span className="title-bar-title">{title}</span>
                    <span className="text-[10px] text-muted-foreground/40 font-normal ml-1.5">v{packageJson.version}</span>
                </div>
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
