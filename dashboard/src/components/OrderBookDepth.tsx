import { useState, useEffect, useCallback } from 'react';
import { getOrderBook } from '../api/client';
import type { OrderBookDepth as OrderBookData, BookLevel } from '../api/types';

function LevelRow({ level, maxQty, side }: { level: BookLevel; maxQty: number; side: 'bid' | 'ask' }) {
  const pct = maxQty > 0 ? (level.total_qty / maxQty) * 100 : 0;
  const barColor = side === 'bid' ? 'bg-[#00C805]/15' : 'bg-red-500/15';
  const textColor = side === 'bid' ? 'text-[#00C805]' : 'text-red-400';

  return (
    <div className="relative flex items-center text-xs h-6 px-2 hover:bg-white/2 transition-colors">
      <div
        className={`absolute inset-y-0 ${side === 'bid' ? 'right-0' : 'left-0'} ${barColor}`}
        style={{ width: `${pct}%` }}
      />
      <span className={`relative w-20 ${textColor} font-mono tabular-nums`}>
        ${parseFloat(level.price).toFixed(2)}
      </span>
      <span className="relative flex-1 text-right text-gray-400 font-mono tabular-nums">
        {level.total_qty.toLocaleString()}
      </span>
      <span className="relative w-8 text-right text-gray-600 font-mono tabular-nums">
        {level.order_count}
      </span>
    </div>
  );
}

export function OrderBookDepth({ symbol }: { symbol: string }) {
  const [book, setBook] = useState<OrderBookData | null>(null);

  const fetchBook = useCallback(() => {
    getOrderBook(symbol)
      .then(setBook)
      .catch(() => {});
  }, [symbol]);

  useEffect(() => {
    fetchBook();
    const id = setInterval(fetchBook, 5000);
    return () => clearInterval(id);
  }, [fetchBook]);

  const asks = book?.asks ?? [];
  const bids = book?.bids ?? [];
  const allQty = [...asks, ...bids].map((l) => l.total_qty);
  const maxQty = allQty.length > 0 ? Math.max(...allQty) : 0;

  const spread =
    asks.length > 0 && bids.length > 0
      ? (parseFloat(asks[0].price) - parseFloat(bids[0].price)).toFixed(2)
      : null;

  return (
    <div className="bg-gray-900/40 rounded-xl p-5">
      <p className="text-xs text-gray-500 mb-3">Order Book</p>

      {/* Header */}
      <div className="flex items-center text-[10px] text-gray-600 uppercase tracking-wider px-2 mb-1">
        <span className="w-20">price</span>
        <span className="flex-1 text-right">qty</span>
        <span className="w-8 text-right">orders</span>
      </div>

      {/* Asks */}
      <div className="mb-1">
        {asks.length > 0 ? (
          [...asks].reverse().slice(0, 8).map((level, i) => (
            <LevelRow key={`ask-${i}`} level={level} maxQty={maxQty} side="ask" />
          ))
        ) : (
          <p className="text-gray-700 text-xs px-2 py-1">No asks</p>
        )}
      </div>

      {/* Spread */}
      {spread && (
        <div className="flex items-center justify-center text-[10px] text-gray-600 py-1 border-y border-white/5">
          Spread ${spread}
        </div>
      )}

      {/* Bids */}
      <div className="mt-1">
        {bids.length > 0 ? (
          bids.slice(0, 8).map((level, i) => (
            <LevelRow key={`bid-${i}`} level={level} maxQty={maxQty} side="bid" />
          ))
        ) : (
          <p className="text-gray-700 text-xs px-2 py-1">No bids</p>
        )}
      </div>
    </div>
  );
}
