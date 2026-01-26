"use client";

import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { ArrowLeft, LogIn } from "lucide-react";
import { normalizeVariant } from "@/lib/variants";

interface JoinGameProps {
    params: Promise<{
        variant: string;
    }>;
}

export default function JoinGame({ params }: JoinGameProps) {
    const [code, setCode] = useState("");
    const router = useRouter();
    const { variant: rawVariant } = use(params);
    const variant = normalizeVariant(rawVariant);

    const handleJoin = () => {
        const trimmedCode = code.trim().toUpperCase();
        if (trimmedCode.length === 4) {
            router.push(`/${variant}/game/${trimmedCode}`);
        }
    };

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const value = e.target.value.toUpperCase().replace(/[^A-Z]/g, "");
        if (value.length <= 4) {
            setCode(value);
        }
    };

    return (
        <div className="flex min-h-screen items-center justify-center p-4 bg-slate-950 text-slate-50">
            <Card className="w-full max-w-xl bg-slate-900/50 border-slate-800 shadow-2xl backdrop-blur-sm p-6">
                <CardHeader className="text-center space-y-2 pb-6">
                    <CardTitle className="text-4xl font-bold text-amber-500">Join Game</CardTitle>
                    <p className="text-slate-400 text-lg">Enter the 4-letter room code</p>
                </CardHeader>

                <CardContent className="space-y-6">
                    <div className="space-y-2">
                        <Input
                            placeholder="CODE"
                            value={code}
                            onChange={handleInputChange}
                            onKeyDown={(e) => e.key === "Enter" && handleJoin()}
                            className="text-center text-5xl font-black font-mono tracking-[0.5em] h-24 uppercase bg-slate-950/50 border-slate-700 focus-visible:ring-amber-500/50 text-amber-100 placeholder:text-slate-800"
                            maxLength={4}
                            autoFocus
                        />
                    </div>

                    <Button
                        onClick={handleJoin}
                        className="w-full h-14 text-lg font-bold bg-amber-600 hover:bg-amber-500 text-white transition-all"
                        disabled={code.length !== 4}
                    >
                        <LogIn className="mr-2 h-5 w-5" />
                        Enter Room
                    </Button>

                    <div className="pt-6 border-t border-slate-800">
                        <Button
                            onClick={() => router.push(`/${variant}`)}
                            variant="ghost"
                            className="w-full text-slate-400 hover:text-white hover:bg-slate-800"
                        >
                            <ArrowLeft className="mr-2 h-4 w-4" />
                            Back to Menu
                        </Button>
                    </div>
                </CardContent>
            </Card>
        </div>
    );
}
