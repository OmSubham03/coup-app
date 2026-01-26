"use client";

import { Button } from "@/components/ui/button";
import { X, BookOpen } from "lucide-react";
import Image from "next/image";
import { VariantKey, getVariantConfig, CHARACTER_IMAGES } from "@/lib/variants";

interface RulesModalProps {
    isOpen: boolean;
    onClose: () => void;
    variant: VariantKey | string;
}

export function RulesModal({ isOpen, onClose, variant }: RulesModalProps) {
    if (!isOpen) return null;

    const variantConfig = getVariantConfig(variant);

    return (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm flex items-center justify-center z-50 p-4 overflow-y-auto">
            <div className="bg-slate-900 border border-slate-700 rounded-xl w-full max-w-4xl max-h-[90vh] flex flex-col shadow-2xl animate-in zoom-in duration-300">
                {/* Header */}
                <div className="flex items-center justify-between p-6 border-b border-slate-700 bg-slate-800/50 rounded-t-xl">
                    <div className="flex items-center gap-3">
                        <BookOpen className="size-6 text-purple-400" />
                        <h2 className="text-2xl font-bold text-white">Game Rules</h2>
                    </div>
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={onClose}
                        className="text-slate-400 hover:text-white hover:bg-slate-700"
                    >
                        <X className="size-6" />
                    </Button>
                </div>

                {/* Content */}
                <div className="flex-1 overflow-y-auto p-6 space-y-8 text-slate-300">

                    {/* Overview */}
                    <section>
                        <h3 className="text-xl font-bold text-purple-400 mb-2">Objective</h3>
                        <p>
                            Eliminate the influence of all other players. The last player with influence (cards) remaining wins.
                        </p>
                    </section>

                    {/* Characters */}
                    <section>
                        <h3 className="text-xl font-bold text-purple-400 mb-4">Characters & Actions</h3>
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                            {variantConfig.characterRules.map((rule) => (
                                <div key={rule.character} className="bg-slate-800/50 p-4 rounded-lg border border-slate-700 flex gap-4">
                                    <div className="relative w-28 h-40 shrink-0 rounded overflow-hidden border border-slate-600">
                                        <Image src={CHARACTER_IMAGES[rule.character]} alt={rule.character} fill className="object-cover" />
                                    </div>
                                    <div>
                                        <h4 className="font-bold text-white text-lg">{rule.character}</h4>
                                        <ul className="text-sm space-y-1 mt-1">
                                            {rule.actions.map((action) => (
                                                <li key={action.label}>
                                                    <span className="text-purple-300 font-semibold">{action.label}:</span> {action.description}
                                                </li>
                                            ))}
                                            {rule.blocks.map((block) => (
                                                <li key={block.label}>
                                                    <span className="text-blue-400 font-semibold">{block.label}:</span> {block.description}
                                                </li>
                                            ))}
                                        </ul>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </section>

                    {/* General Actions */}
                    <section>
                        <h3 className="text-xl font-bold text-purple-400 mb-2">General Actions</h3>
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                            {variantConfig.generalActions.map((action) => (
                                <div key={action.label} className="bg-slate-800/30 p-3 rounded border border-slate-700">
                                    <h4 className="font-bold text-white">{action.label}</h4>
                                    <p className="text-sm">{action.description}</p>
                                </div>
                            ))}
                        </div>
                    </section>

                    {/* Challenges & Blocking */}
                    <section className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div>
                            <h3 className="text-xl font-bold text-purple-400 mb-2">Challenges</h3>
                            <p className="text-sm leading-relaxed">
                                Any action that uses a character (Tax, Assassinate, Steal, {variantConfig.availableActions.includes('interrogate') ? 'Interrogate, Inquire' : 'Exchange'}) or any Block can be challenged.
                                <br /><br />
                                If challenged, you must prove you have the character.
                                <br />
                                <span className="text-green-400">If you show the card:</span> You shuffle it back, draw a new one, and the challenger loses an influence.
                                <br />
                                <span className="text-red-400">If you can&apos;t/won&apos;t:</span> You lose an influence.
                            </p>
                        </div>
                        <div>
                            <h3 className="text-xl font-bold text-purple-400 mb-2">Blocking</h3>
                            <p className="text-sm leading-relaxed">
                                You can claim a character to block an action against you (or Foreign Aid).
                                <br /><br />
                                Blocks can be challenged just like actions.
                            </p>
                        </div>
                    </section>
                </div>

                {/* Footer */}
                <div className="p-6 border-t border-slate-700 bg-slate-800/50 rounded-b-xl flex justify-end">
                    <Button onClick={onClose} className="bg-purple-600 hover:bg-purple-700">
                        Close Rules
                    </Button>
                </div>
            </div>
        </div>
    );
}
