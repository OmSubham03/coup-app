"use client";

import { useEffect, useState, useRef } from "react";
import { Player } from "@/lib/game-logic";
import { cn } from "@/lib/utils";
import { Crown, Users } from "lucide-react";

interface StartingPlayerRouletteProps {
    players: Player[];
    startingPlayerId: string;
    onComplete: () => void;
}

export function StartingPlayerRoulette({ players, startingPlayerId, onComplete }: StartingPlayerRouletteProps) {
    const [highlightedIndex, setHighlightedIndex] = useState(0);
    const [isFinished, setIsFinished] = useState(false);

    // Use refs for mutable state in the timeout loop
    const speedRef = useRef(50);
    const counterRef = useRef(0);
    const timeoutRef = useRef<NodeJS.Timeout | null>(null);

    useEffect(() => {
        const targetIndex = players.findIndex(p => p.id === startingPlayerId);
        const minSpins = 4; // Spin at least this many times around
        const totalSteps = (players.length * minSpins) + targetIndex; // Total steps to reach target

        // Adjust starting index to be consistent if we want, but 0 is fine.
        // Actually, if we start at 0, we need to make sure the math aligns.
        // If current is 0, and we want to reach targetIndex.
        // (0 + steps) % length === targetIndex
        // steps % length === targetIndex
        // So (length * minSpins) + targetIndex is correct.

        const spin = () => {
            setHighlightedIndex(prev => (prev + 1) % players.length);
            counterRef.current += 1;

            const stepsRemaining = totalSteps - counterRef.current;

            if (stepsRemaining <= 0) {
                // We landed on the winner!
                setIsFinished(true);
                setTimeout(onComplete, 2500); // Show winner for 2.5 seconds
                return;
            }

            // Slow down as we get closer
            if (stepsRemaining < 15) {
                speedRef.current += 15;
            } else if (stepsRemaining < 8) {
                speedRef.current += 30;
            } else if (stepsRemaining < 3) {
                speedRef.current += 50;
            }

            timeoutRef.current = setTimeout(spin, speedRef.current);
        };

        timeoutRef.current = setTimeout(spin, speedRef.current);

        return () => {
            if (timeoutRef.current) clearTimeout(timeoutRef.current);
        };
    }, [players, startingPlayerId, onComplete]);

    return (
        <div className="fixed inset-0 z-100 bg-black/80 backdrop-blur-md flex flex-col items-center justify-center p-4 animate-in fade-in duration-300">
            <div className="max-w-2xl w-full space-y-12 text-center">
                <div className="space-y-2">
                    <h2 className="text-4xl font-bold text-transparent bg-clip-text bg-linear-to-r from-purple-400 to-pink-600 animate-pulse">
                        {isFinished ? "Starting Player Selected!" : "Choosing Starting Player..."}
                    </h2>
                    <p className="text-slate-400 text-lg">The fates are deciding who goes first</p>
                </div>

                <div className="grid grid-cols-2 md:grid-cols-3 gap-6 justify-items-center">
                    {players.map((player, index) => {
                        const isHighlighted = index === highlightedIndex;
                        const isWinner = isFinished && isHighlighted;

                        return (
                            <div
                                key={player.id}
                                className={cn(
                                    "relative w-full max-w-[180px] aspect-square rounded-2xl flex flex-col items-center justify-center gap-3 transition-all duration-150 border-2",
                                    isHighlighted
                                        ? "scale-110 bg-slate-800 border-purple-500 shadow-[0_0_30px_rgba(168,85,247,0.5)] z-10"
                                        : "scale-95 bg-slate-900/50 border-slate-800 opacity-40 grayscale",
                                    isWinner && "bg-linear-to-br from-purple-900 to-pink-900 border-yellow-400 shadow-[0_0_50px_rgba(234,179,8,0.5)] scale-125 z-20"
                                )}
                            >
                                {isWinner && (
                                    <div className="absolute -top-6 animate-bounce">
                                        <Crown className="size-10 text-yellow-400 drop-shadow-lg" />
                                    </div>
                                )}

                                <div className={cn(
                                    "p-4 rounded-full transition-colors",
                                    isHighlighted ? "bg-purple-500/20" : "bg-slate-800"
                                )}>
                                    <Users className={cn(
                                        "size-8",
                                        isHighlighted ? "text-purple-300" : "text-slate-500",
                                        isWinner && "text-yellow-200"
                                    )} />
                                </div>

                                <span className={cn(
                                    "font-bold text-lg truncate max-w-full px-2",
                                    isHighlighted ? "text-white" : "text-slate-500",
                                    isWinner && "text-yellow-200"
                                )}>
                                    {player.name}
                                </span>
                            </div>
                        );
                    })}
                </div>
            </div>
        </div>
    );
}
