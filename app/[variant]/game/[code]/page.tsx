import { redirect } from "next/navigation";
import { CoupGameClient } from "@/components/game/coup-game-client";
import { isVariantKey } from "@/lib/variants";

interface GamePageProps {
    params: Promise<{
        variant: string;
        code: string;
    }>;
}

export default async function GamePage({ params }: GamePageProps) {
    const { variant, code } = await params;

    if (!isVariantKey(variant)) {
        redirect(`/standard/game/${code}`);
    }

    return <CoupGameClient roomCode={code} variant={variant} />;
}
