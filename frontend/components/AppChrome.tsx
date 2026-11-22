'use client';

import { usePathname } from 'next/navigation';
import Sidebar from '@/components/Sidebar';

// The login page renders standalone; everything else gets the sidebar shell.
export default function AppChrome({ children }: { children: React.ReactNode }) {
    const pathname = usePathname();

    if (pathname === '/login') return <>{children}</>;

    return (
        <div className="flex h-screen bg-background">
            <Sidebar />
            <main className="flex-1 overflow-auto">
                <div className="p-8">{children}</div>
            </main>
        </div>
    );
}
