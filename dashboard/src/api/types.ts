export interface User {
  id: string;
  username: string;
  balance: string;
  created_at: string;
}

export interface Position {
  id: string;
  user_id: string;
  symbol: string;
  quantity: number;
  avg_cost: string;
  updated_at: string;
}

export interface Order {
  id: string;
  user_id: string;
  symbol: string;
  side: 'buy' | 'sell';
  quantity: number;
  price: string;
  filled_quantity?: number;
  order_type?: string;
  status: 'pending' | 'validated' | 'open' | 'partial' | 'executed' | 'settled' | 'rejected';
  idempotency_key?: string;
  created_at: string;
  executed_at?: string;
  settled_at?: string;
}

export interface WSMessage {
  type: 'order_update' | 'trade' | 'balance_update';
  data: unknown;
}

export interface CreateOrderRequest {
  user_id: string;
  symbol: string;
  side: 'buy' | 'sell';
  quantity: number;
  price: string;
  idempotency_key?: string;
}
