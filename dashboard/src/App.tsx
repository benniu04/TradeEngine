import { useState, useCallback } from 'react';
import { getUser, getPositions, getOrders } from './api/client';
import type { User, Position, Order, WSMessage } from './api/types';
import { usePolling } from './hooks/usePolling';
import { useWebSocket } from './hooks/useWebSocket';
import { UserSelector } from './components/UserSelector';
import { PortfolioOverview } from './components/PortfolioOverview';
import { OrderForm } from './components/OrderForm';
import { OrderHistory } from './components/OrderHistory';
import { MarketQuote } from './components/MarketQuote';
import { OrderBookDepth } from './components/OrderBookDepth';

const DEFAULT_USER = '11111111-1111-1111-1111-111111111111';

function App() {
  const [userId, setUserId] = useState(DEFAULT_USER);
  const [user, setUser] = useState<User | null>(null);
  const [positions, setPositions] = useState<Position[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [selectedSymbol, setSelectedSymbol] = useState('AAPL');

  const fetchAll = useCallback(() => {
    Promise.all([getUser(userId), getPositions(userId), getOrders(userId)])
      .then(([u, p, o]) => {
        setUser(u);
        setPositions(p);
        setOrders(o);
        setError(null);
      })
      .catch((err) => {
        setError(err.message);
      });
  }, [userId]);

  const handleWSMessage = useCallback((_msg: WSMessage) => {
    fetchAll();
  }, [fetchAll]);

  const wsConnected = useWebSocket(userId, handleWSMessage);

  usePolling(fetchAll, wsConnected ? 30000 : 5000);

  return (
    <div className="min-h-screen bg-black text-white">
      <header className="sticky top-0 z-50 backdrop-blur-md bg-black/80 border-b border-white/5 px-6 py-3 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-semibold tracking-tight text-white">TradeEngine</h1>
          <span
            className={`w-1.5 h-1.5 rounded-full ${wsConnected ? 'bg-[#00C805]' : 'bg-red-500'}`}
            title={wsConnected ? 'WebSocket connected' : 'Polling mode'}
          />
        </div>
        <UserSelector selectedId={userId} onChange={setUserId} />
      </header>

      {error && (
        <div className="max-w-6xl mx-auto px-6 pt-4">
          <p className="text-red-400 text-sm bg-red-500/10 rounded-lg px-4 py-2">
            {error} — make sure the backend is running on :8080
          </p>
        </div>
      )}

      {user ? (
        <main className="max-w-6xl mx-auto px-6 py-6 space-y-6">
          {/* Hero: Portfolio */}
          <PortfolioOverview user={user} positions={positions} />

          {/* 3-column: Market + Book + Order Form */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
            <MarketQuote symbol={selectedSymbol} />
            <OrderBookDepth symbol={selectedSymbol} />
            <OrderForm
              userId={userId}
              symbol={selectedSymbol}
              onSymbolChange={setSelectedSymbol}
              onOrderPlaced={fetchAll}
            />
          </div>

          {/* Order history */}
          <OrderHistory orders={orders} />
        </main>
      ) : (
        !error && (
          <div className="flex items-center justify-center h-64">
            <div className="w-4 h-4 border-2 border-gray-700 border-t-gray-400 rounded-full animate-spin" />
          </div>
        )
      )}
    </div>
  );
}

export default App;
