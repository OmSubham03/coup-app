import { redirect } from "next/navigation";
import { MainMenu } from "@/components/game/main-menu";
import { isVariantKey } from "@/lib/variants";

interface VariantPageProps {
    params: Promise<{
        variant: string;
    }>;
}

export default async function VariantPage({ params }: VariantPageProps) {
    const { variant } = await params;

    if (!isVariantKey(variant)) {
        redirect("/standard");
    }

    return <MainMenu variant={variant} />;
}
