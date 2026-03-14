import { useState } from 'react';
import { createOrder } from '../api/client';

interface Props {
  userId: string;
  onOrderPlaced: () => void;
}

export function OrderForm({ userId, onOrderPlaced }: Props) {
  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [symbol, setSymbol] = useState('');
  const [quantity, setQuantity] = useState('');
  const [price, setPrice] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setMessage(null);

    try {
      await createOrder({
        user_id: userId,
        symbol: symbol.toUpperCase(),
        side,
        quantity: parseInt(quantity, 10),
        price,
        idempotency_key: crypto.randomUUID(),
      });
      setMessage({ type: 'success', text: 'Order placed!' });
      setSymbol('');
      setQuantity('');
      setPrice('');
      onOrderPlaced();
    } catch (err) {
      setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Failed to place order' });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="bg-gray-900 rounded-xl p-6">
      <p className="text-white text-lg font-semibold mb-4">Place Order</p>

      <div className="flex mb-4 rounded-lg overflow-hidden">
        <button
          type="button"
          onClick={() => setSide('buy')}
          className={`flex-1 py-2 text-sm font-medium transition-colors ${
            side === 'buy'
              ? 'bg-[#00C805] text-black'
              : 'bg-gray-800 text-gray-400 hover:text-white'
          }`}
        >
          Buy
        </button>
        <button
          type="button"
          onClick={() => setSide('sell')}
          className={`flex-1 py-2 text-sm font-medium transition-colors ${
            side === 'sell'
              ? 'bg-red-500 text-white'
              : 'bg-gray-800 text-gray-400 hover:text-white'
          }`}
        >
          Sell
        </button>
      </div>

      <form onSubmit={handleSubmit} className="space-y-3">
        <input
          type="text"
          placeholder="Symbol (e.g. AAPL)"
          value={symbol}
          onChange={(e) => setSymbol(e.target.value)}
          required
          className="w-full bg-gray-800 border border-gray-700 text-white rounded-lg px-3 py-2 text-sm uppercase placeholder:normal-case focus:border-[#00C805] focus:outline-none"
        />
        <input
          type="number"
          placeholder="Quantity"
          value={quantity}
          onChange={(e) => setQuantity(e.target.value)}
          min="1"
          required
          className="w-full bg-gray-800 border border-gray-700 text-white rounded-lg px-3 py-2 text-sm focus:border-[#00C805] focus:outline-none"
        />
        <input
          type="number"
          placeholder="Price"
          value={price}
          onChange={(e) => setPrice(e.target.value)}
          min="0.01"
          step="0.01"
          required
          className="w-full bg-gray-800 border border-gray-700 text-white rounded-lg px-3 py-2 text-sm focus:border-[#00C805] focus:outline-none"
        />
        <button
          type="submit"
          disabled={submitting}
          className={`w-full py-2.5 rounded-lg text-sm font-semibold transition-colors ${
            side === 'buy'
              ? 'bg-[#00C805] text-black hover:bg-[#00b004] disabled:opacity-50'
              : 'bg-red-500 text-white hover:bg-red-600 disabled:opacity-50'
          }`}
        >
          {submitting ? 'Placing...' : `${side === 'buy' ? 'Buy' : 'Sell'} ${symbol.toUpperCase() || 'Stock'}`}
        </button>
      </form>

      {message && (
        <p className={`mt-3 text-sm ${message.type === 'success' ? 'text-green-400' : 'text-red-400'}`}>
          {message.text}
        </p>
      )}
    </div>
  );
}
