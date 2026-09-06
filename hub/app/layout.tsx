import type { Metadata, Viewport } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import "@/styles/globals.css";
import { Providers } from "@/components/layout/providers";
import { ServiceWorkerRegistration } from "@/components/layout/service-worker-registration";
import { ErrorBoundary } from "@/components/ui/error-boundary";
import { LocaleProvider } from "@/lib/i18n/use-locale";
import { cn } from "@/lib/utils";

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
  display: "swap",
});

const jetbrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-jetbrains-mono",
  display: "swap",
});

export const metadata: Metadata = {
  title: "RenOS",
  description: "Your devices. One intelligent system.",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#FFFFFF" },
    { media: "(prefers-color-scheme: dark)", color: "#1C1C1E" },
  ],
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <link rel="manifest" href="/manifest.json" />
        <meta name="apple-mobile-web-app-capable" content="yes" />
        <meta name="mobile-web-app-capable" content="yes" />
      </head>
      <body
        className={cn("min-h-screen", inter.variable, jetbrainsMono.variable)}
      >
        <ErrorBoundary>
          <Providers>
            <ServiceWorkerRegistration />
            <LocaleProvider>{children}</LocaleProvider>
          </Providers>
        </ErrorBoundary>
      </body>
    </html>
  );
}
