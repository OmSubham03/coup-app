import { redirect } from "next/navigation";

interface GamePageProps {
    params: Promise<{
        code: string;
    }>;
}

export default async function GamePage({ params }: GamePageProps) {
    const { code } = await params;

    redirect(`/standard/game/${code}`);
}