"use client";

import { use, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Loader2, AlertCircle, RefreshCw } from "lucide-react";
import { normalizeVariant } from "@/lib/variants";

interface CreateGameProps {
    params: Promise<{
        variant: string;
    }>;
}

export default function CreateGame({ params }: CreateGameProps) {
    const router = useRouter();
    const [error, setError] = useState<string | null>(null);
    const { variant: rawVariant } = use(params);
    const variant = normalizeVariant(rawVariant);

    useEffect(() => {
        async function generateUniqueCode() {
            try {
                const response = await fetch("/api/generate-code");
                if (!response.ok) {
                    throw new Error("Failed to generate code");
                }

                const data = await response.json();
                router.push(`/${variant}/game/${data.code}?action=create`);
            } catch (err) {
                console.error("Error generating code:", err);
                setError("Failed to create game. Please try again.");
            }
        }

        generateUniqueCode();
    }, [router, variant]);

    if (error) {
        return (
            <div className="flex min-h-screen items-center justify-center p-4 bg-slate-950 text-slate-50">
                <div className="text-center space-y-6 max-w-md w-full p-8 bg-slate-900/50 border border-red-900/50 rounded-2xl backdrop-blur-sm">
                    <div className="flex justify-center">
                        <div className="p-4 bg-red-900/20 rounded-full">
                            <AlertCircle className="h-12 w-12 text-red-500" />
                        </div>
                    </div>
                    <h1 className="text-2xl font-bold text-red-400">Connection Error</h1>
                    <p className="text-slate-400">{error}</p>
                    <button
                        onClick={() => window.location.reload()}
                        className="flex items-center justify-center w-full px-4 py-3 bg-slate-800 hover:bg-slate-700 text-white rounded-lg transition-colors font-medium"
                    >
                        <RefreshCw className="mr-2 h-4 w-4" />
                        Try Again
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className="flex min-h-screen items-center justify-center p-4 bg-slate-950 text-slate-50">
            <div className="text-center space-y-8">
                <div className="relative">
                    <div className="absolute inset-0 bg-amber-500/20 blur-xl rounded-full animate-pulse"></div>
                    <Loader2 className="h-16 w-16 animate-spin text-amber-500 relative z-10 mx-auto" />
                </div>
                <div className="space-y-2">
                    <h1 className="text-2xl font-bold text-amber-100">Creating Room...</h1>
                    <p className="text-slate-500 animate-pulse">Setting up the court</p>
                </div>
            </div>
        </div>
    );
}
