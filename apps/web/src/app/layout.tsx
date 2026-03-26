import type { Metadata, Viewport } from "next";
import { Syne, JetBrains_Mono, Space_Mono, Instrument_Serif } from "next/font/google";
import "./globals.css";
import { Providers } from "./providers";

const syne = Syne({
  subsets: ["latin"],
  variable: "--font-display",
  display: "swap",
  weight: ["400", "500", "600", "700", "800"],
});

const jetbrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-mono",
  display: "swap",
});

const spaceMono = Space_Mono({
  subsets: ["latin"],
  variable: "--font-terminal",
  display: "swap",
  weight: ["400", "700"],
});

const instrumentSerif = Instrument_Serif({
  subsets: ["latin"],
  variable: "--font-editorial",
  display: "swap",
  weight: "400",
});

export const metadata: Metadata = {
  title: "GAENDE Radio",
  description: "Anti-algorithm 24/7 webradio — 9 stages, radical transparency",
  icons: { icon: "/favicon.ico" },
};

export const viewport: Viewport = {
  themeColor: "#020204",
  width: "device-width",
  initialScale: 1,
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${syne.variable} ${jetbrainsMono.variable} ${spaceMono.variable} ${instrumentSerif.variable}`}
    >
      <body>
        <Providers>
          {children}
        </Providers>
      </body>
    </html>
  );
}
