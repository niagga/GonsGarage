import type { Metadata } from "next";
import Script from "next/script";
import { Geist, Geist_Mono } from "next/font/google";
import { AuthProvider } from "@/contexts/AuthContext";
import ThemeSwitcher from "@/components/theme/ThemeSwitcher";
import { THEME_INLINE_SCRIPT } from "@/lib/theme-inline-script";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "GonsGarage",
  description: "Gestão de oficina — aplicação web",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="pt" suppressHydrationWarning>
      <body className={`${geistSans.variable} ${geistMono.variable}`}>
        <Script
          id="gg-theme-init"
          strategy="beforeInteractive"
          dangerouslySetInnerHTML={{ __html: THEME_INLINE_SCRIPT }}
        />
        <ThemeSwitcher />
        <AuthProvider>{children}</AuthProvider>
      </body>
    </html>
  );
}