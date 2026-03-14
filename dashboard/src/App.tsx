import { useState, useCallback } from 'react';
import { getUser, getPositions, getOrders } from './api/client';
import type { User, Position, Order } from './api/types';
import { usePolling } from './hooks/usePolling';
import { UserSelector } from './components/UserSelector';
import { PortfolioOverview } from './components/PortfolioOverview';
import { OrderForm } from './components/OrderForm';
import { OrderHistory } from './components/OrderHistory';

const DEFAULT_USER = '11111111-1111-1111-1111-111111111111';

function App() {
  const [userId, setUserId] = useState(DEFAULT_USER);
  const [user, setUser] = useState<User | null>(null);
  const [positions, setPositions] = useState<Position[]>([]);
  const [orders, setOrders] = useState<Order[]>([]);
  const [error, setError] = useState<string | null>(null);

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

  usePolling(fetchAll, 5000);

  return (
    <div className="min-h-screen bg-black text-white">
      <header className="border-b border-gray-800 px-6 py-4 flex items-center justify-between">
        <h1 className="text-xl font-bold text-[#00C805]">TradeEngine</h1>
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
        <main className="max-w-6xl mx-auto grid grid-cols-1 md:grid-cols-3 gap-6 p-6">
          <div className="md:col-span-2">
            <PortfolioOverview user={user} positions={positions} />
          </div>
          <div>
            <OrderForm userId={userId} onOrderPlaced={fetchAll} />
          </div>
          <div className="md:col-span-3">
            <OrderHistory orders={orders} />
          </div>
        </main>
      ) : (
        !error && (
          <div className="flex items-center justify-center h-64">
            <p className="text-gray-400">Loading...</p>
          </div>
        )
      )}
    </div>
  );
}

export default App;
