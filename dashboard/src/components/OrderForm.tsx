import { useState } from 'react';
import { createOrder } from '../api/client';

interface Props {
  userId: string;
  symbol: string;
  onSymbolChange: (symbol: string) => void;
  onOrderPlaced: () => void;
}

export function OrderForm({ userId, symbol, onSymbolChange, onOrderPlaced }: Props) {
  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [quantity, setQuantity] = useState('');
  const [price, setPrice] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const qty = parseInt(quantity, 10) || 0;
  const prc = parseFloat(price) || 0;
  const estTotal = qty * prc;

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
      setQuantity('');
      setPrice('');
      onOrderPlaced();
    } catch (err) {
      setMessage({ type: 'error', text: err instanceof Error ? err.message : 'Failed to place order' });
    } finally {
      setSubmitting(false);
    }
  };

  const isBuy = side === 'buy';

  return (
    <div className="bg-gray-900/60 rounded-xl p-5">
      <p className="text-white text-sm font-semibold mb-3">Place Order</p>

      <div className="flex mb-4 rounded-md overflow-hidden text-xs font-medium">
        <button
          type="button"
          onClick={() => setSide('buy')}
          className={`flex-1 py-2 transition-colors ${
            isBuy
              ? 'bg-[#00C805] text-black'
              : 'bg-gray-800 text-gray-500 hover:text-gray-300'
          }`}
        >
          Buy
        </button>
        <button
          type="button"
          onClick={() => setSide('sell')}
          className={`flex-1 py-2 transition-colors ${
            !isBuy
              ? 'bg-red-500 text-white'
              : 'bg-gray-800 text-gray-500 hover:text-gray-300'
          }`}
        >
          Sell
        </button>
      </div>

      <form onSubmit={handleSubmit} className="space-y-3">
        <div>
          <label className="block text-[11px] text-gray-500 mb-1 uppercase tracking-wider">Symbol</label>
          <input
            type="text"
            placeholder="AAPL"
            value={symbol}
            onChange={(e) => onSymbolChange(e.target.value)}
            required
            className="w-full bg-gray-800/80 border border-gray-700/50 text-white rounded-md px-3 py-2 text-sm uppercase placeholder:text-gray-600 focus:border-gray-500 focus:outline-none transition-colors"
          />
        </div>

        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className="block text-[11px] text-gray-500 mb-1 uppercase tracking-wider">Quantity</label>
            <input
              type="number"
              placeholder="0"
              value={quantity}
              onChange={(e) => setQuantity(e.target.value)}
              min="1"
              required
              className="w-full bg-gray-800/80 border border-gray-700/50 text-white rounded-md px-3 py-2 text-sm font-mono placeholder:text-gray-600 focus:border-gray-500 focus:outline-none transition-colors"
            />
          </div>
          <div>
            <label className="block text-[11px] text-gray-500 mb-1 uppercase tracking-wider">Price</label>
            <input
              type="number"
              placeholder="0.00"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              min="0.01"
              step="0.01"
              required
              className="w-full bg-gray-800/80 border border-gray-700/50 text-white rounded-md px-3 py-2 text-sm font-mono placeholder:text-gray-600 focus:border-gray-500 focus:outline-none transition-colors"
            />
          </div>
        </div>

        {estTotal > 0 && (
          <div className="flex justify-between text-xs px-1">
            <span className="text-gray-500">Est. Total</span>
            <span className="text-gray-300 font-mono tabular-nums">
              ${estTotal.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
            </span>
          </div>
        )}

        <button
          type="submit"
          disabled={submitting}
          className={`w-full py-2.5 rounded-md text-sm font-semibold transition-colors ${
            isBuy
              ? 'bg-[#00C805] text-black hover:bg-[#00b004] disabled:opacity-40'
              : 'bg-red-500 text-white hover:bg-red-600 disabled:opacity-40'
          }`}
        >
          {submitting ? 'Placing...' : `${isBuy ? 'Buy' : 'Sell'} ${symbol.toUpperCase() || 'Stock'}`}
        </button>
      </form>

      {message && (
        <p className={`mt-3 text-xs ${message.type === 'success' ? 'text-green-400' : 'text-red-400'}`}>
          {message.text}
        </p>
      )}
    </div>
  );
}
