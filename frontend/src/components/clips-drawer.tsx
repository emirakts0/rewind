import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from "framer-motion"
import { FolderOpen, FileVideo, Clock, RefreshCcw, ArrowLeft, ExternalLink, Trash2, CheckCircle2, Circle, CheckSquare, Square } from 'lucide-react'
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
    const [selectedClips, setSelectedClips] = useState<Set<string>>(new Set())
    const [selectionMode, setSelectionMode] = useState(false)
    const [deleting, setDeleting] = useState(false)
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

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
        } else {
            // Reset selection when drawer closes
            setSelectionMode(false)
            setSelectedClips(new Set())
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
        if (selectionMode) return
        try {
            await api.openClip(path)
        } catch (err) {
            toast.error("Failed to open clip")
        }
    }

    const handleOpenFolder = async () => {
        try {
            await api.openOutputDirectory()
        } catch (err) {
            toast.error("Failed to open folder")
        }
    }

    const toggleSelection = (path: string) => {
        setSelectedClips(prev => {
            const newSet = new Set(prev)
            if (newSet.has(path)) {
                newSet.delete(path)
            } else {
                newSet.add(path)
            }
            return newSet
        })
    }

    const handleDeleteSelected = async () => {
        if (selectedClips.size === 0) return

        setDeleting(true)
        try {
            await api.deleteClips(Array.from(selectedClips))
            toast.success(`Deleted ${selectedClips.size} clip${selectedClips.size > 1 ? 's' : ''}`)
            setSelectedClips(new Set())
            setSelectionMode(false)
            setShowDeleteConfirm(false)
            await fetchClips()
        } catch (err: any) {
            toast.error(err?.message || "Failed to delete clips")
        } finally {
            setDeleting(false)
        }
    }

    const handleDeleteClick = () => {
        if (selectedClips.size === 0) return
        setShowDeleteConfirm(true)
    }

    const handleCancelDelete = () => {
        setShowDeleteConfirm(false)
    }

    const toggleSelectionMode = () => {
        setSelectionMode(!selectionMode)
        setSelectedClips(new Set())
    }

    const selectAll = () => {
        setSelectedClips(new Set(clips.map(clip => clip.path)))
    }

    const deselectAll = () => {
        setSelectedClips(new Set())
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
                        <div className="ml-auto flex items-center gap-1">
                            {selectionMode ? (
                                <>
                                    <Button
                                        variant="ghost"
                                        size="sm"
                                        className="h-7 px-2 text-xs text-muted-foreground hover:text-foreground"
                                        style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
                                        onClick={toggleSelectionMode}
                                    >
                                        Cancel
                                    </Button>
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        className="h-7 w-7 rounded-full text-muted-foreground hover:text-foreground hover:bg-white/10"
                                        style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
                                        onClick={handleDeleteClick}
                                        disabled={selectedClips.size === 0 || deleting}
                                    >
                                        <Trash2 className="h-4 w-4" />
                                    </Button>
                                </>
                            ) : (
                                <>
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        className="h-6 w-6 text-muted-foreground hover:text-foreground"
                                        style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
                                        onClick={toggleSelectionMode}
                                        disabled={clips.length === 0}
                                        title="Select clips to delete"
                                    >
                                        <Trash2 className="h-3.5 w-3.5" />
                                    </Button>
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        className="h-6 w-6 text-muted-foreground hover:text-foreground"
                                        style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
                                        onClick={handleOpenFolder}
                                        title="Open folder in explorer"
                                    >
                                        <ExternalLink className="h-3.5 w-3.5" />
                                    </Button>
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        className="h-6 w-6 text-muted-foreground hover:text-foreground"
                                        style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
                                        onClick={fetchClips}
                                        disabled={loading}
                                    >
                                        <RefreshCcw className={cn("h-3.5 w-3.5", loading && "animate-spin")} />
                                    </Button>
                                </>
                            )}
                        </div>
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
                        <div className="px-4 pt-3 pb-4">
                            {selectionMode && (
                                <div className="flex items-center justify-between px-3 py-3 mb-2">
                                    <span className="text-sm text-muted-foreground">
                                        {selectedClips.size} selected
                                    </span>
                                    <button
                                        onClick={selectedClips.size === clips.length ? deselectAll : selectAll}
                                        className="flex items-center gap-2 transition-colors hover:opacity-80"
                                    >
                                        <span className="text-sm text-muted-foreground">
                                            {selectedClips.size === clips.length ? 'Deselect All' : 'Select All'}
                                        </span>
                                        <div className="flex items-center justify-center">
                                            {selectedClips.size === clips.length ? (
                                                <CheckCircle2 className="h-5 w-5 text-primary" />
                                            ) : (
                                                <Circle className="h-5 w-5 text-muted-foreground/50" />
                                            )}
                                        </div>
                                    </button>
                                </div>
                            )}
                            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
                                <AnimatePresence mode="popLayout" initial={false}>
                                    {clips.map((clip) => {
                                    const isSelected = selectedClips.has(clip.path)
                                    return (
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
                                            onClick={() => selectionMode ? toggleSelection(clip.path) : handleOpenClip(clip.path)}
                                            className={cn(
                                                "group relative flex items-center gap-3 p-3 rounded-lg border transition-colors duration-300 text-left",
                                                selectionMode
                                                    ? isSelected
                                                        ? "border-primary bg-primary/10"
                                                        : "border-border/40 bg-card/50 hover:bg-accent/30"
                                                    : "border-border/40 bg-card/50 hover:bg-accent/50"
                                            )}
                                        >
                                            {selectionMode && (
                                                <div className="absolute top-2 right-2 z-10">
                                                    {isSelected ? (
                                                        <CheckCircle2 className="h-5 w-5 text-primary" />
                                                    ) : (
                                                        <Circle className="h-5 w-5 text-muted-foreground/50" />
                                                    )}
                                                </div>
                                            )}
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
                                    )
                                })}
                            </AnimatePresence>
                            </div>
                        </div>
                    )}
                </ScrollArea>

                {/* Delete Confirmation Overlay */}
                <AnimatePresence>
                    {showDeleteConfirm && (
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            className="absolute inset-0 bg-background/80 backdrop-blur-sm flex items-center justify-center z-50"
                            onClick={handleCancelDelete}
                        >
                            <motion.div
                                initial={{ scale: 0.9, opacity: 0 }}
                                animate={{ scale: 1, opacity: 1 }}
                                exit={{ scale: 0.9, opacity: 0 }}
                                onClick={(e) => e.stopPropagation()}
                                className="bg-card border border-border rounded-lg p-6 max-w-sm mx-4 shadow-xl"
                            >
                                <div className="flex flex-col items-center text-center">
                                    <div className="mb-4">
                                        <Trash2 className="h-12 w-12 text-muted-foreground" />
                                    </div>
                                    <h3 className="text-xl font-semibold mb-2">Delete Clips?</h3>
                                    <p className="text-sm text-muted-foreground mb-6">
                                        Are you sure you want to delete {selectedClips.size} clip{selectedClips.size > 1 ? 's' : ''}? This action cannot be undone.
                                    </p>
                                    <div className="flex gap-3 w-full">
                                        <Button
                                            variant="outline"
                                            onClick={handleCancelDelete}
                                            disabled={deleting}
                                            className="flex-1"
                                        >
                                            Cancel
                                        </Button>
                                        <Button
                                            variant="destructive"
                                            onClick={handleDeleteSelected}
                                            disabled={deleting}
                                            className="flex-1 bg-destructive hover:bg-destructive/90"
                                        >
                                            {deleting ? 'Deleting...' : 'Delete'}
                                        </Button>
                                    </div>
                                </div>
                            </motion.div>
                        </motion.div>
                    )}
                </AnimatePresence>
            </SheetContent >
        </Sheet >
    )
}
