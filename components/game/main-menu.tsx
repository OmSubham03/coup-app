"use client"

import { Button } from "@/components/ui/button"
import { useRouter } from "next/navigation"
import { Crown, Users, Eye, BookOpen, ArrowLeft } from "lucide-react"
import { useState } from "react"
import { GameplayPreviewModal } from "./gameplay-preview-modal"
import { RulesModal } from "./rules-modal"
import { normalizeVariant, VariantKey } from "@/lib/variants"

export function MainMenu({ variant }: { variant: VariantKey | string }) {
    const router = useRouter()
    const [showPreview, setShowPreview] = useState(false)
    const [showRules, setShowRules] = useState(false)
    const normalizedVariant = normalizeVariant(variant)
    const basePath = `/${normalizedVariant}`

    return (
        <div className="flex min-h-screen items-center justify-center bg-slate-950 text-slate-50">
            <GameplayPreviewModal isOpen={showPreview} onClose={() => setShowPreview(false)} />
            <RulesModal isOpen={showRules} onClose={() => setShowRules(false)} variant={normalizedVariant} />
            <div className="flex flex-col gap-6 w-full max-w-2xl px-12 py-16 bg-slate-900/50 border border-slate-800 rounded-2xl shadow-2xl backdrop-blur-sm">
                <div className="-mt-4">
                    <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => router.push("/")}
                        className="text-slate-400 hover:text-slate-100 hover:bg-slate-800/60"
                    >
                        <ArrowLeft className="mr-2 h-4 w-4" />
                        Choose another variant
                    </Button>
                </div>
                <div className="text-center space-y-2 mb-8">
                    <h1 className="text-7xl font-black tracking-tighter text-transparent bg-clip-text bg-linear-to-br from-amber-300 to-amber-600 drop-shadow-sm">
                        COUP
                    </h1>
                    <p className="text-slate-400 font-medium tracking-wide uppercase text-sm">
                        Bluff • Deduce • Dominate
                    </p>
                </div>

                <Button
                    size="lg"
                    onClick={() => router.push(`${basePath}/create`)}
                    className="w-full text-lg py-8 bg-linear-to-r from-amber-600 to-amber-700 hover:from-amber-500 hover:to-amber-600 text-white border-0 shadow-lg transition-all hover:scale-[1.02]"
                >
                    <Crown className="mr-2 h-6 w-6" />
                    Create New Game
                </Button>

                <Button
                    size="lg"
                    variant="outline"
                    onClick={() => router.push(`${basePath}/join`)}
                    className="w-full text-lg py-8 border-2 border-slate-700 bg-transparent hover:bg-slate-800 text-slate-200 hover:text-white transition-all hover:scale-[1.02]"
                >
                    <Users className="mr-2 h-6 w-6" />
                    Join Game
                </Button>

                <Button
                    size="lg"
                    variant="ghost"
                    onClick={() => setShowPreview(true)}
                    className="w-full text-lg py-6 text-slate-400 hover:text-amber-400 hover:bg-slate-800/50 transition-all"
                >
                    <Eye className="mr-2 h-5 w-5" />
                    See Gameplay
                </Button>

                <Button
                    size="lg"
                    variant="ghost"
                    onClick={() => setShowRules(true)}
                    className="w-full text-lg py-6 text-slate-400 hover:text-purple-400 hover:bg-slate-800/50 transition-all"
                >
                    <BookOpen className="mr-2 h-5 w-5" />
                    Game Rules
                </Button>
            </div>
        </div>
    )
}
