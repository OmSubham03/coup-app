"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Card as GameCard } from "@/lib/game-logic";
import { Skull } from "lucide-react";
import Image from "next/image";
import { CHARACTER_IMAGES } from "@/lib/variants";

interface CardSelectorProps {
    cards: GameCard[];
    title: string;
    description: string;
    selectCount: number;
    onConfirm: (selectedCardIds: string[]) => void;
}

export function CardSelector({
    cards,
    title,
    description,
    selectCount,
    onConfirm,
}: CardSelectorProps) {
    const [selectedCards, setSelectedCards] = useState<string[]>([]);

    const toggleCard = (cardId: string) => {
        setSelectedCards(prev => {
            if (prev.includes(cardId)) {
                return prev.filter(id => id !== cardId);
            } else if (prev.length < selectCount) {
                return [...prev, cardId];
            }
            return prev;
        });
    };

    const handleConfirm = () => {
        if (selectedCards.length === selectCount) {
            onConfirm(selectedCards);
        }
    };

    return (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50 p-4">
            <Card className="bg-linear-to-br from-slate-800 to-slate-900 border-2 border-purple-500 max-w-2xl w-full">
                <CardHeader>
                    <CardTitle className="text-2xl text-purple-300">{title}</CardTitle>
                    <p className="text-slate-400">{description}</p>
                </CardHeader>
                <CardContent className="space-y-6">
                    <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                        {cards.map((card) => (
                            <button
                                key={card.id}
                                onClick={() => toggleCard(card.id)}
                                disabled={card.revealed}
                                className={`w-full h-48 rounded-lg flex flex-col items-center justify-center transition-all relative overflow-hidden group ${card.revealed
                                    ? "opacity-50 grayscale cursor-not-allowed"
                                    : selectedCards.includes(card.id)
                                        ? "ring-4 ring-green-500 scale-105 shadow-lg shadow-green-500/50"
                                        : "ring-2 ring-purple-400 hover:scale-105 hover:shadow-lg hover:shadow-purple-500/50"
                                    }`}
                            >
                                {card.revealed ? (
                                    <div className="w-full h-full relative bg-slate-900 flex flex-col items-center justify-center">
                                        <Image
                                            src={CHARACTER_IMAGES[card.character]}
                                            alt={card.character}
                                            fill
                                            className="object-cover opacity-30"
                                        />
                                        <div className="absolute inset-0 flex flex-col items-center justify-center z-10 bg-black/40">
                                            <Skull className="size-12 text-red-500 mb-2 drop-shadow-lg" />
                                            <span className="text-sm font-bold text-red-500 bg-black/70 px-2 py-1 rounded border border-red-500/30">{card.character}</span>
                                        </div>
                                    </div>
                                ) : (
                                    <div className="w-full h-full relative">
                                        <Image
                                            src={CHARACTER_IMAGES[card.character]}
                                            alt={card.character}
                                            fill
                                            className="object-cover"
                                        />
                                        <div className="absolute bottom-0 inset-x-0 bg-linear-to-t from-black/90 via-black/50 to-transparent p-2 pt-8">
                                            <p className="text-center text-white font-bold text-lg shadow-black drop-shadow-md">{card.character}</p>
                                        </div>
                                        {selectedCards.includes(card.id) && (
                                            <div className="absolute inset-0 bg-green-500/20 flex items-center justify-center">
                                                <div className="bg-green-500 text-white px-3 py-1 rounded-full font-bold shadow-lg">Selected</div>
                                            </div>
                                        )}
                                    </div>
                                )}
                            </button>
                        ))}
                    </div>

                    <div className="space-y-2">
                        <div className="flex justify-between text-sm text-slate-400">
                            <span>Selected: {selectedCards.length} / {selectCount}</span>
                            {selectedCards.length < selectCount && (
                                <span className="text-yellow-400">Select {selectCount - selectedCards.length} more</span>
                            )}
                        </div>
                        <Button
                            onClick={handleConfirm}
                            disabled={selectedCards.length !== selectCount}
                            className="w-full h-12 text-lg bg-linear-to-r from-purple-600 to-pink-600 hover:from-purple-700 hover:to-pink-700 disabled:opacity-50"
                        >
                            Confirm Selection
                        </Button>
                    </div>
                </CardContent>
            </Card>
        </div>
    );
}
