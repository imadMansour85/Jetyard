// SPDX-License-Identifier: AGPL-3.0-or-later
import type { Metadata } from "next";
import "./globals.css";
import BackendGuard from "@/components/BackendGuard";
import AuthGuard from "@/components/AuthGuard";
import AppChrome from "@/components/AppChrome";
import { ToastProvider } from "@/components/Toast";
import { ConfirmProvider } from "@/components/ConfirmDialog";

export const metadata: Metadata = {
    title: "Jetyard",
    description: "Manage your NATS JetStream infrastructure",
};

export default function RootLayout({
    children,
}: Readonly<{
    children: React.ReactNode;
}>) {
    return (
        <html lang="en">
            <head>
                <link rel="preconnect" href="https://fonts.googleapis.com" />
                <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="" />
                <link
                    href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap"
                    rel="stylesheet"
                />
            </head>
            <body>
                <ToastProvider>
                    <ConfirmProvider>
                        <BackendGuard>
                            <AuthGuard>
                                <AppChrome>{children}</AppChrome>
                            </AuthGuard>
                        </BackendGuard>
                    </ConfirmProvider>
                </ToastProvider>
            </body>
        </html>
    );
}
