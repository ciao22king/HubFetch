import type { Metadata } from "next"
import "./globals.css"

export const metadata: Metadata = {
  title: "HubFetch — From repo URL to ready-to-code",
  description: "HubFetch is the focused open-source workflow for cloning, organizing, and revisiting Git repositories with less setup and more momentum.",
}

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en"><body>{children}</body></html>
}
