"use client";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { GameState, CharacterType } from "@/lib/game-logic";
import { AlertTriangle, Shield, Swords, Target } from "lucide-react";
import Image from "next/image";
import { cn } from "@/lib/utils";
import { CHARACTER_IMAGES, getVariantConfig } from "@/lib/variants";

interface BlockChallengePanelProps {
    gameState: GameState;
    myPlayerId: string;
    onBlock: (character: CharacterType) => void;
    onChallenge: (targetPlayerId: string, character: CharacterType) => void;
    onPass: () => void;
}

export function BlockChallengePanel({
    gameState,
    myPlayerId,
    onBlock,
    onChallenge,
    onPass,
}: BlockChallengePanelProps) {
    const myPlayer = gameState.players.find(p => p.id === myPlayerId);
    const variantConfig = getVariantConfig(gameState.variant);

    if (!myPlayer || !myPlayer.isAlive) return null;

    // Check if I have already passed
    const hasPassed = gameState.passedPlayers?.includes(myPlayerId);

    // Block Window
    if (gameState.phase === 'block_window' && gameState.pendingAction) {
        const actor = gameState.players.find(p => p.id === gameState.pendingAction?.actorId);
        const target = gameState.pendingAction.targetId
            ? gameState.players.find(p => p.id === gameState.pendingAction?.targetId)
            : null;

        const canBlock = (gameState.pendingAction.type === 'foreign_aid' && gameState.pendingAction.actorId !== myPlayerId) ||
            (target && target.id === myPlayerId);

        const isTargeted = target?.id === myPlayerId;
        const blockCharacters = variantConfig.actionRequirements[gameState.pendingAction.type]?.blockingCharacters || [];

        if (!canBlock || hasPassed) {
            return (
                <Card className="bg-slate-800/50 backdrop-blur-sm border-slate-700">
                    <CardContent className="p-6 text-center">
                        <p className="text-slate-400">
                            Waiting for others to respond to <span className="font-bold text-white">{actor?.name}</span>&apos;s action...
                        </p>
                    </CardContent>
                </Card>
            );
        }

        return (
            <Card className={cn(
                "backdrop-blur-sm border-2 animate-pulse transition-colors",
                isTargeted
                    ? "bg-slate-950 bg-linear-to-r from-red-900/50 to-orange-900/50 border-red-500 shadow-[0_0_20px_rgba(239,68,68,0.3)]"
                    : "bg-slate-950 bg-linear-to-r from-yellow-900/30 to-orange-900/30 border-yellow-500/50"
            )}>
                <CardHeader>
                    <CardTitle className={cn(
                        "flex items-center gap-2",
                        isTargeted ? "text-red-400 text-2xl uppercase tracking-wider" : "text-yellow-300"
                    )}>
                        {isTargeted ? <Target className="size-6 animate-bounce" /> : <Shield className="size-5" />}
                        {isTargeted ? "You are Targeted!" : "Block Opportunity!"}
                    </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                    <p className="text-white text-lg">
                        {isTargeted ? (
                            <>
                                <span className="font-bold text-red-300">{actor?.name}</span> is trying to <span className="font-bold text-red-400 underline decoration-red-500/50 underline-offset-4">{gameState.pendingAction.type.replace('_', ' ')}</span> YOU!
                            </>
                        ) : (
                            <>
                                <span className="font-bold">{actor?.name}</span> is attempting{" "}
                                <span className="font-bold text-yellow-400">{gameState.pendingAction.type.replace('_', ' ')}</span>
                                {target && (
                                    <>
                                        {" "}targeting <span className="font-bold">{target.name}</span>
                                    </>
                                )}
                            </>
                        )}
                    </p>

                    <div className="space-y-2">
                        {blockCharacters.length > 0 && (gameState.pendingAction.type === 'foreign_aid' || target?.id === myPlayerId) && (
                            <>
                                {blockCharacters.map((character) => (
                                    <Button
                                        key={character}
                                        onClick={() => onBlock(character)}
                                        className={`w-full h-14 flex items-center justify-start gap-3 px-4 ${character === 'Duke'
                                            ? 'bg-purple-600 hover:bg-purple-700'
                                            : character === 'Contessa'
                                                ? 'bg-red-600 hover:bg-red-700'
                                                : character === 'Captain'
                                                    ? 'bg-cyan-600 hover:bg-cyan-700'
                                                    : 'bg-indigo-600 hover:bg-indigo-700'
                                            }`}
                                    >
                                        <div className="relative w-8 h-8 rounded-full overflow-hidden border border-white/50">
                                            <Image src={CHARACTER_IMAGES[character]} alt={character} fill className="object-cover" />
                                        </div>
                                        <span className="font-bold">Block with {character}</span>
                                    </Button>
                                ))}
                            </>
                        )}

                        <Button
                            onClick={onPass}
                            variant="outline"
                            className="w-full border-slate-500 text-slate-400 hover:bg-slate-800"
                        >
                            Allow Action
                        </Button>
                    </div>
                </CardContent>
            </Card>
        );
    }

    // Challenge Window
    if (gameState.phase === 'challenge_window') {
        let targetPlayerId: string | undefined;
        let claimedCharacter: CharacterType | undefined;
        let actionDescription = "";

        if (gameState.pendingBlock) {
            // Someone blocked an action - they can be challenged
            targetPlayerId = gameState.pendingBlock.blockerId;
            claimedCharacter = gameState.pendingBlock.claimedCharacter;
            const blocker = gameState.players.find(p => p.id === targetPlayerId);
            actionDescription = `${blocker?.name} claims to have ${claimedCharacter} to block`;
        } else if (gameState.pendingAction?.claimedCharacter) {
            // Someone claimed a character for their action - they can be challenged
            targetPlayerId = gameState.pendingAction.actorId;
            claimedCharacter = gameState.pendingAction.claimedCharacter;
            const actor = gameState.players.find(p => p.id === targetPlayerId);
            actionDescription = `${actor?.name} claims to have ${claimedCharacter}`;
        }

        // Don't show panel if:
        // - No one to challenge (no character was claimed)
        // - I am the one being challenged (can't challenge myself)
        if (!targetPlayerId || !claimedCharacter || targetPlayerId === myPlayerId || hasPassed) {
            // Only show waiting message if there IS someone to challenge (just not me)
            if ((targetPlayerId && claimedCharacter && targetPlayerId === myPlayerId) || hasPassed) {
                return (
                    <Card className="bg-slate-800/50 backdrop-blur-sm border-slate-700">
                        <CardContent className="p-6 text-center">
                            <p className="text-slate-400">Waiting for others to challenge...</p>
                        </CardContent>
                    </Card>
                );
            }
            return null;
        }

        // Check if I am the target of the action being challenged
        const isTargetedByAction = !gameState.pendingBlock && gameState.pendingAction?.targetId === myPlayerId;
        const canBlockLater = isTargetedByAction && gameState.pendingAction?.type && ['assassinate', 'steal'].includes(gameState.pendingAction.type);

        return (
            <Card className={cn(
                "backdrop-blur-sm border-2 animate-pulse transition-colors",
                isTargetedByAction
                    ? "bg-slate-950 bg-linear-to-r from-red-900/50 to-orange-900/50 border-red-500 shadow-[0_0_20px_rgba(239,68,68,0.3)]"
                    : "bg-slate-950 bg-linear-to-r from-red-900/30 to-pink-900/30 border-red-500/50"
            )}>
                <CardHeader>
                    <CardTitle className={cn(
                        "flex items-center gap-2",
                        isTargetedByAction ? "text-red-400 text-2xl uppercase tracking-wider" : "text-red-300"
                    )}>
                        {isTargetedByAction ? <Target className="size-6 animate-bounce" /> : <Swords className="size-5" />}
                        {isTargetedByAction ? "You are Targeted!" : "Challenge Opportunity!"}
                    </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="flex items-start gap-3">
                        <div className={cn(
                            "relative w-16 h-16 rounded-lg overflow-hidden border-2 shrink-0 shadow-lg",
                            isTargetedByAction ? "border-red-400 shadow-red-500/30" : "border-red-500 shadow-red-500/20"
                        )}>
                            <Image src={CHARACTER_IMAGES[claimedCharacter]} alt={claimedCharacter} fill className="object-cover" />
                        </div>
                        <div>
                            <p className="text-white mb-2 font-medium text-lg">
                                {isTargetedByAction ? (
                                    <>
                                        <span className="font-bold text-red-300">{gameState.players.find(p => p.id === targetPlayerId)?.name}</span> claims <span className="font-bold text-yellow-400">{claimedCharacter}</span> to <span className="underline decoration-red-500/50 underline-offset-4">attack YOU!</span>
                                    </>
                                ) : (
                                    actionDescription
                                )}
                            </p>
                            <p className="text-sm text-slate-300 font-medium">
                                {isTargetedByAction
                                    ? "You can challenge their character claim now. If you pass, you will have a chance to BLOCK."
                                    : `If you challenge and they don't have ${claimedCharacter}, they lose influence.`
                                }
                            </p>
                        </div>
                    </div>                    <div className="space-y-2">
                        <Button
                            onClick={() => onChallenge(targetPlayerId!, claimedCharacter!)}
                            className={cn(
                                "w-full transition-all hover:scale-[1.02]",
                                isTargetedByAction ? "bg-red-600 hover:bg-red-700 h-12 text-lg font-bold shadow-lg shadow-red-900/20" : "bg-red-600 hover:bg-red-700"
                            )}
                        >
                            Challenge {claimedCharacter}
                        </Button>
                        <Button
                            onClick={onPass}
                            variant="outline"
                            className="w-full border-slate-500 text-slate-400 hover:bg-slate-800"
                        >
                            {canBlockLater ? "Pass (Proceed to Block)" : "Allow"}
                        </Button>
                    </div>
                </CardContent>
            </Card>
        );
    }

    return null;
}
