import { useEffect, useRef, useState } from 'react';
import { getToken } from './auth';

import { API_BASE } from './config';

export interface SSEEvent {
    type: string;
    data: any;
}

export function useSSE(onEvent: (event: SSEEvent) => void) {
    const [connected, setConnected] = useState(false);
    const eventSourceRef = useRef<EventSource | null>(null);
    const reconnectTimeoutRef = useRef<NodeJS.Timeout>();

    useEffect(() => {
        const connect = () => {
            try {
                const token = getToken();
                if (!token) return;
                // EventSource cannot set headers, so the token rides as a query param.
                const eventSource = new EventSource(`${API_BASE}/api/events?access_token=${encodeURIComponent(token)}`);
                eventSourceRef.current = eventSource;

                eventSource.onopen = () => {
                    console.log('SSE connected');
                    setConnected(true);
                };

                eventSource.onmessage = (event) => {
                    try {
                        const data = JSON.parse(event.data);
                        onEvent(data);
                    } catch (error) {
                        console.error('Failed to parse SSE event:', error);
                    }
                };

                eventSource.onerror = () => {
                    console.log('SSE connection error, reconnecting...');
                    setConnected(false);
                    eventSource.close();

                    // Reconnect after 3 seconds
                    reconnectTimeoutRef.current = setTimeout(() => {
                        connect();
                    }, 3000);
                };
            } catch (error) {
                console.error('Failed to create EventSource:', error);
                setConnected(false);
            }
        };

        connect();

        return () => {
            if (reconnectTimeoutRef.current) {
                clearTimeout(reconnectTimeoutRef.current);
            }
            if (eventSourceRef.current) {
                eventSourceRef.current.close();
            }
        };
    }, [onEvent]);

    return { connected };
}
