import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from "framer-motion"
import { FolderOpen, FileVideo, Clock, RefreshCcw, ArrowLeft } from 'lucide-react'
import { api, type Clip } from '@/lib/wails'
import { cn, formatBytes } from '@/lib/utils'

import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetHeader,
    SheetTitle,
    SheetTrigger,
} from "@/components/ui/sheet"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"

export function ClipsDrawer() {
    const [clips, setClips] = useState<Clip[]>([])
    const [loading, setLoading] = useState(false)
    const [open, setOpen] = useState(false)

    const fetchClips = async () => {
        setLoading(true)
        try {
            const data = await api.getClips()
            // Sort by modTime desc (newest first)
            const sorted = (data || []).sort((a, b) =>
                new Date(b.modTime).getTime() - new Date(a.modTime).getTime()
            )
            setClips(sorted)
        } catch (err) {
            console.error(err)
            toast.error("Failed to load clips")
        } finally {
            setLoading(false)
        }
    }

    useEffect(() => {
        if (open) {
            toast.dismiss()
            fetchClips()
        }
    }, [open])

    // Listen for clips-updated event
    useEffect(() => {
        const unsub = api.Events.On('clips-updated', () => {
            if (open) {
                fetchClips()
            }
        })
        return () => unsub()
    }, [open])

    const handleOpenClip = async (path: string) => {
        try {
            await api.openClip(path)
        } catch (err) {
            toast.error("Failed to open clip")
        }
    }

    return (
        <Sheet open={open} onOpenChange={setOpen}>
            <SheetTrigger asChild>
                <Button
                    variant="ghost"
                    size="icon"
                    className="h-9 w-9 text-muted-foreground hover:text-foreground rounded-full hover:bg-white/10"
                >
                    <FolderOpen className="h-5 w-5" />
                </Button>
            </SheetTrigger>
            <SheetContent className="w-screen max-w-none h-full border-l-0 [&>button]:hidden p-0 flex flex-col gap-0">
                <SheetHeader className="px-4 py-2 border-b border-border/50 flex-shrink-0" style={{ background: 'linear-gradient(to bottom, hsl(249 10% 18%), hsl(249 10% 14%))', '--wails-draggable': 'drag' } as React.CSSProperties}>
                    <SheetTitle className="flex items-center gap-3 text-sm font-bold tracking-tight h-6">
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7 text-muted-foreground hover:text-foreground"
                            style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
                            onClick={() => setOpen(false)}
                        >
                            <ArrowLeft className="h-4 w-4" />
                        </Button>
                        <span>Library</span>
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6 text-muted-foreground hover:text-foreground ml-auto"
                            style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
                            onClick={fetchClips}
                            disabled={loading}
                        >
                            <RefreshCcw className={cn("h-3.5 w-3.5", loading && "animate-spin")} />
                        </Button>
                    </SheetTitle>
                    <SheetDescription className="hidden">
                        All your captured moments.
                    </SheetDescription>
                </SheetHeader>

                <ScrollArea className="flex-1">
                    {clips.length === 0 ? (
                        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground">
                            <FileVideo className="h-12 w-12 opacity-10 mb-4" />
                            <p className="text-sm font-medium">No clips found</p>
                        </div>
                    ) : (
                        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2 px-4 pt-3 pb-4">
                            <AnimatePresence mode="popLayout" initial={false}>
                                {clips.map((clip) => (
                                    <motion.button
                                        key={clip.path}
                                        layout
                                        initial={{ opacity: 0, scale: 0.8, y: 20 }}
                                        animate={{ opacity: 1, scale: 1, y: 0 }}
                                        exit={{ opacity: 0, scale: 0.8, y: -10 }}
                                        transition={{
                                            opacity: { duration: 0.25 },
                                            scale: { duration: 0.4, type: "spring", stiffness: 200, damping: 20 },
                                            y: { duration: 0.4, type: "spring", stiffness: 200, damping: 20 },
                                            layout: { duration: 0.5, type: "spring", stiffness: 150, damping: 25 }
                                        }}
                                        onClick={() => handleOpenClip(clip.path)}
                                        className="group relative flex items-center gap-3 p-3 rounded-lg border border-border/40 bg-card/50 hover:bg-accent/50 transition-colors duration-300 text-left"
                                    >
                                        <div className="h-10 w-10 rounded-md bg-secondary/50 flex items-center justify-center shrink-0">
                                            <FileVideo className="h-5 w-5 text-primary" />
                                        </div>
                                        <div className="flex-1 min-w-0">
                                            <p className="font-medium text-sm truncate">{clip.name}</p>
                                            <div className="flex items-center gap-2 text-[10px] text-muted-foreground/80 mt-0.5">
                                                <span className="flex items-center gap-1 bg-background/50 px-1.5 py-0.5 rounded">
                                                    <Clock className="w-2.5 h-2.5" />
                                                    {new Date(clip.modTime).toLocaleString()}
                                                </span>
                                                <span className="font-mono">{formatBytes(clip.size)}</span>
                                            </div>
                                        </div>
                                    </motion.button>
                                ))}
                            </AnimatePresence>
                        </div>
                    )}
                </ScrollArea>
            </SheetContent >
        </Sheet >
    )
}
