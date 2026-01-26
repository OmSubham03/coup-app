import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

export default function Home() {
  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 flex items-center justify-center p-6">
      <div className="max-w-4xl w-full space-y-8">
        <div className="text-center space-y-3">
          <h1 className="text-6xl font-black tracking-tight text-transparent bg-clip-text bg-linear-to-br from-amber-300 to-amber-600">
            COUP
          </h1>
          <p className="text-slate-400 text-lg">Choose your variant to begin</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <Card className="bg-slate-900/50 border-slate-800 shadow-2xl backdrop-blur-sm">
            <CardHeader>
              <CardTitle className="text-2xl text-amber-400">Standard</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-slate-400">
                Classic Coup rules featuring the Ambassador and Exchange.
              </p>
              <Button asChild className="w-full bg-amber-600 hover:bg-amber-500 text-white">
                <Link href="/standard">Play Standard</Link>
              </Button>
            </CardContent>
          </Card>

          <Card className="bg-slate-900/50 border-slate-800 shadow-2xl backdrop-blur-sm">
            <CardHeader>
              <CardTitle className="text-2xl text-emerald-400">Inquisitor</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-slate-400">
                Swap the Ambassador for the Inquisitor and interrogate opponents.
              </p>
              <Button asChild className="w-full bg-emerald-600 hover:bg-emerald-500 text-white">
                <Link href="/inquisitor">Play Inquisitor</Link>
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
