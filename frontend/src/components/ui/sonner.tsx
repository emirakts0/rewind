"use client"

import { useTheme } from "@/components/theme-provider"
import { Toaster as Sonner } from "sonner"

type ToasterProps = React.ComponentProps<typeof Sonner>

const Toaster = ({ ...props }: ToasterProps) => {
    const { theme = "system" } = useTheme()

    return (
        <Sonner
            theme={theme as ToasterProps["theme"]}
            className="toaster group"
            style={{ top: 48 }}
            toastOptions={{
                classNames: {
                    toast:
                        "group toast !bg-muted !text-foreground !border-border shadow-lg",
                    description: "!text-muted-foreground",
                    actionButton:
                        "!bg-primary !text-primary-foreground",
                    cancelButton:
                        "!bg-muted !text-muted-foreground",
                    error:
                        "!bg-destructive !text-destructive-foreground !border-destructive",
                    success:
                        "!bg-emerald-950 !text-emerald-300 !border-emerald-700",
                    info:
                        "!bg-muted !text-foreground !border-border",
                },
            }}
            {...props}
        />
    )
}

export { Toaster }
