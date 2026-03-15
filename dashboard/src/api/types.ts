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

export interface Quote {
  c: number;   // current price
  d: number;   // change
  dp: number;  // percent change
  h: number;   // high
  l: number;   // low
  o: number;   // open
  pc: number;  // previous close
  t: number;   // timestamp
}

export interface BookLevel {
  price: string;
  total_qty: number;
  order_count: number;
  side: 'buy' | 'sell';
}

export interface OrderBookDepth {
  symbol: string;
  bids: BookLevel[];
  asks: BookLevel[];
}

export interface CreateOrderRequest {
  user_id: string;
  symbol: string;
  side: 'buy' | 'sell';
  quantity: number;
  price: string;
  idempotency_key?: string;
}
