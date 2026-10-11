import type { Metadata } from "next"
import "./globals.css"

export const metadata: Metadata = {
  title: "HubFetch — From repo URL to ready-to-code",
  description: "HubFetch rende semplice clonare, preparare e ritrovare repository Git: meno attrito, più tempo per costruire.",
}

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en"><body>{children}</body></html>
}
