import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from "framer-motion"
import { FolderOpen, FileVideo, Clock, RefreshCcw, FileDigit, ArrowLeft, MoreHorizontal, Film, Loader2, FolderArchive } from 'lucide-react'
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
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

export function ClipsDrawer() {
    const [clips, setClips] = useState<Clip[]>([])
    const [loading, setLoading] = useState(false)
    const [open, setOpen] = useState(false)
    const [converting, setConverting] = useState<Record<string, boolean>>({})

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

    const handleOpenClip = async (path: string) => {
        try {
            await api.openClip(path)
        } catch (err) {
            toast.error("Failed to open clip")
        }
    }

    const getIcon = (clip: Clip) => {
        if (clip.isRawFolder) return <FolderArchive className="h-5 w-5 text-amber-500/70" />
        if (clip.name.endsWith('.ts')) return <FileDigit className="h-5 w-5 text-amber-500/70" />
        return <FileVideo className="h-5 w-5 text-primary" />
    }

    const [newClipName, setNewClipName] = useState<string | null>(null)

    const handleConvert = async (path: string) => {
        setConverting(prev => ({ ...prev, [path]: true }))
        try {
            await api.convertToMP4(path)

            // Extract expected new filename for highlight animation
            const originalFileName = path.split(/[/\\]/).pop() || ''
            // For raw folders: folder name + .mp4, for .ts files: replace .ts with .mp4
            const newFileName = originalFileName.endsWith('.ts')
                ? originalFileName.replace(/\.ts$/, '.mp4')
                : originalFileName + '.mp4'
            setNewClipName(newFileName)

            // Scroll to top first so animation is visible
            const scrollArea = document.querySelector('[data-radix-scroll-area-viewport]')
            if (scrollArea) {
                scrollArea.scrollTo({ top: 0, behavior: 'smooth' })
            }

            await fetchClips()

            // Clear highlight after animation
            setTimeout(() => setNewClipName(null), 1000)
        } catch (err) {
            toast.error("Conversion failed")
            console.error(err)
        } finally {
            setConverting(prev => ({ ...prev, [path]: false }))
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
                                    <motion.div
                                        key={clip.path}
                                        layout
                                        initial={{ opacity: 0, scale: 0.8, y: 20 }}
                                        animate={{
                                            opacity: 1,
                                            scale: 1,
                                            y: 0,
                                            boxShadow: newClipName === clip.name
                                                ? "0 0 24px 6px rgba(245, 158, 11, 0.35)"
                                                : "none"
                                        }}
                                        exit={{ opacity: 0, scale: 0.8, y: -10 }}
                                        transition={{
                                            opacity: { duration: 0.25 },
                                            scale: { duration: 0.4, type: "spring", stiffness: 200, damping: 20 },
                                            y: { duration: 0.4, type: "spring", stiffness: 200, damping: 20 },
                                            layout: { duration: 0.5, type: "spring", stiffness: 150, damping: 25 },
                                            boxShadow: { duration: 0.8, ease: "easeOut" }
                                        }}
                                        className={cn(
                                            "group relative flex items-center gap-3 p-3 rounded-lg border border-border/40 bg-card/50 hover:bg-accent/50 transition-colors duration-300",
                                            newClipName === clip.name && "border-amber-500/50 bg-amber-500/10"
                                        )}
                                    >
                                        <button
                                            onClick={() => handleOpenClip(clip.path)}
                                            className="flex-1 flex items-center gap-3 min-w-0 text-left"
                                        >
                                            <div className="h-10 w-10 rounded-md bg-secondary/50 flex items-center justify-center shrink-0">
                                                {getIcon(clip)}
                                            </div>
                                            <div className="flex-1 min-w-0">
                                                <p className="font-medium text-sm truncate">{clip.name}</p>
                                                <div className="flex items-center gap-2 text-[10px] text-muted-foreground/80 mt-0.5">
                                                    <span className="flex items-center gap-1 bg-background/50 px-1.5 py-0.5 rounded">
                                                        <Clock className="w-2.5 h-2.5" />
                                                        {new Date(clip.modTime).toLocaleString()}
                                                    </span>
                                                    <span className="font-mono">{formatBytes(clip.size)}</span>
                                                    {clip.isRawFolder && clip.durationSec && (
                                                        <span className="text-amber-500/80">RAW • {clip.durationSec}s</span>
                                                    )}
                                                </div>
                                            </div>
                                        </button>

                                        {/* Actions for raw folders and TS files */}
                                        {(clip.isRawFolder || clip.name.endsWith('.ts')) && (
                                            <DropdownMenu>
                                                <DropdownMenuTrigger asChild>
                                                    <Button
                                                        variant="ghost"
                                                        size="icon"
                                                        className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity focus:opacity-100 data-[state=open]:opacity-100"
                                                        disabled={converting[clip.path]}
                                                    >
                                                        {converting[clip.path] ? (
                                                            <Loader2 className="h-4 w-4 animate-spin" />
                                                        ) : (
                                                            <MoreHorizontal className="h-4 w-4" />
                                                        )}
                                                    </Button>
                                                </DropdownMenuTrigger>
                                                <DropdownMenuContent align="end">
                                                    <DropdownMenuItem onClick={() => handleConvert(clip.path)}>
                                                        <Film className="w-4 h-4 mr-2" />
                                                        Convert to MP4
                                                    </DropdownMenuItem>
                                                </DropdownMenuContent>
                                            </DropdownMenu>
                                        )}
                                    </motion.div>
                                ))}
                            </AnimatePresence>
                        </div>
                    )}
                </ScrollArea>
            </SheetContent >
        </Sheet >
    )
}
