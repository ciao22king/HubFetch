import type { Metadata } from "next"
import "./globals.css"

export const metadata: Metadata = {
  title: "GoFetch — From repo URL to ready-to-code",
  description: "A calm, keyboard-first way to clone, build and revisit Git repositories.",
}

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en"><body>{children}</body></html>
}
