import type { User, Position, Order, CreateOrderRequest, Quote, OrderBookDepth } from './types';

const BASE = '/api';

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`);
  if (!res.ok) throw new Error(`GET ${path}: ${res.status}`);
  return res.json();
}

export const getUser = (id: string) => get<User>(`/users/${id}`);
export const getPositions = (userId: string) => get<Position[]>(`/positions?user_id=${userId}`);
export const getOrders = (userId: string) => get<Order[]>(`/orders?user_id=${userId}`);
export const getQuote = (symbol: string) => get<Quote>(`/quote/${symbol}`);
export const getOrderBook = (symbol: string) => get<OrderBookDepth>(`/book/${symbol}`);

export async function createOrder(req: CreateOrderRequest): Promise<Order> {
  const res = await fetch(`${BASE}/orders`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `POST /orders: ${res.status}`);
  }
  return res.json();
}
