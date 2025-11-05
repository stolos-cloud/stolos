import { StorageService } from '@/services/storage.service';

const DEFAULT_RECONNECT_DELAY = 2000;
const WILDCARD_EVENT = '*';

class WsEventService {
    constructor() {
        this.ws = null;
        this.shouldReconnect = false;
        this.reconnectDelay = DEFAULT_RECONNECT_DELAY;
        this.reconnectTimeout = null;
        this.subscribers = new Map();
    }

    connect() {
        if (this.ws && [WebSocket.OPEN, WebSocket.CONNECTING].includes(this.ws.readyState)) {
            return this.ws;
        }

        const endpoint = this.buildEndpoint();
        if (!endpoint) {
            console.warn('wsEventService: Missing auth token, cannot connect to event stream.');
            return null;
        }

        this.shouldReconnect = true;
        this.ws = new WebSocket(endpoint);
        this.ws.onopen = () => this.notifySubscribers('ws:open', { type: 'ws:open' });
        this.ws.onerror = error => this.notifySubscribers('ws:error', { type: 'ws:error', error });
        this.ws.onmessage = event => this.handleIncomingMessage(event);
        this.ws.onclose = () => this.handleClose();

        return this.ws;
    }

    disconnect() {
        this.shouldReconnect = false;
        if (this.reconnectTimeout) {
            clearTimeout(this.reconnectTimeout);
            this.reconnectTimeout = null;
        }
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }

    isConnected() {
        return this.ws?.readyState === WebSocket.OPEN;
    }

    subscribe(eventType, handler) {
        if (!eventType || typeof handler !== 'function') {
            throw new Error('wsEventService.subscribe requires an eventType and handler');
        }

        if (!this.subscribers.has(eventType)) {
            this.subscribers.set(eventType, new Set());
        }

        const handlers = this.subscribers.get(eventType);
        handlers.add(handler);

        return () => this.unsubscribe(eventType, handler);
    }

    subscribeToAll(handler) {
        return this.subscribe(WILDCARD_EVENT, handler);
    }

    unsubscribe(eventType, handler) {
        const handlers = this.subscribers.get(eventType);
        if (!handlers) return;
        handlers.delete(handler);
        if (handlers.size === 0) {
            this.subscribers.delete(eventType);
        }
    }

    handleIncomingMessage(event) {
        try {
            const payload = JSON.parse(event.data);
            const normalized = {
                type: payload?.type || 'unknown',
                data: payload?.payload,
            };
            this.notifySubscribers(normalized.type, normalized);
        } catch (err) {
            console.error('wsEventService: Failed to parse event message', err);
        }
    }

    handleClose() {
        this.notifySubscribers('ws:close', { type: 'ws:close' });
        this.ws = null;
        if (!this.shouldReconnect) return;

        if (this.reconnectTimeout) {
            clearTimeout(this.reconnectTimeout);
        }

        this.reconnectTimeout = setTimeout(() => {
            this.reconnectTimeout = null;
            this.connect();
        }, this.reconnectDelay);
    }

    notifySubscribers(eventType, payload) {
        const runHandlers = handlers => {
            if (!handlers) return;
            handlers.forEach(handler => {
                try {
                    handler(payload);
                } catch (err) {
                    console.error('wsEventService: subscriber threw an error', err);
                }
            });
        };

        runHandlers(this.subscribers.get(eventType));
        if (eventType !== WILDCARD_EVENT) {
            runHandlers(this.subscribers.get(WILDCARD_EVENT));
        }
    }

    buildEndpoint() {
        const token = StorageService.get('token');
        if (!token) return null;
        const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        return `${wsProtocol}//${window.location.host}/api/events/stream?token=${token}`;
    }
}

const wsEventService = new WsEventService();
export default wsEventService;
