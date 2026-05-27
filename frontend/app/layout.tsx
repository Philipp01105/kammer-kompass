import type { Metadata } from "next";
import "./globals.css";
import { QueryProvider } from "@/lib/queryClient";

export const metadata: Metadata = {
  title: "KammerKompass",
  description: "Inoffizielle Community-Datenbank fuer IHK-spezifische Hinweise.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="de">
      <body>
        <QueryProvider>{children}</QueryProvider>
      </body>
    </html>
  );
}
