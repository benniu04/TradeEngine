import { useState, useEffect, useCallback } from 'react';
import { getQuote } from '../api/client';
import type { Quote } from '../api/types';

export function MarketQuote({ symbol }: { symbol: string }) {
  const [quote, setQuote] = useState<Quote | null>(null);
  const [error, setError] = useState(false);

  const fetchQuote = useCallback(() => {
    getQuote(symbol)
      .then((q) => {
        setQuote(q);
        setError(false);
      })
      .catch(() => setError(true));
  }, [symbol]);

  useEffect(() => {
    fetchQuote();
    const id = setInterval(fetchQuote, 15000);
    return () => clearInterval(id);
  }, [fetchQuote]);

  if (error) {
    return (
      <div className="bg-gray-900/40 rounded-xl p-5">
        <p className="text-gray-600 text-xs">
          Market data unavailable — set FINNHUB_API_KEY to enable
        </p>
      </div>
    );
  }

  if (!quote || quote.c === 0) {
    return (
      <div className="bg-gray-900/40 rounded-xl p-5">
        <p className="text-gray-600 text-xs">Loading {symbol}...</p>
      </div>
    );
  }

  const isUp = quote.d >= 0;

  return (
    <div className="bg-gray-900/40 rounded-xl p-5">
      <p className="text-xs text-gray-500 mb-2">{symbol}</p>

      <div className="flex items-baseline gap-2 mb-4">
        <span className="text-2xl font-bold text-white font-mono tabular-nums">
          ${quote.c.toFixed(2)}
        </span>
        <span className={`text-xs font-medium ${isUp ? 'text-[#00C805]' : 'text-red-400'}`}>
          {isUp ? '▲' : '▼'} {Math.abs(quote.d).toFixed(2)} ({Math.abs(quote.dp).toFixed(2)}%)
        </span>
      </div>

      <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 text-xs">
        <div className="flex justify-between">
          <span className="text-gray-600">Open</span>
          <span className="text-gray-400 font-mono tabular-nums">${quote.o.toFixed(2)}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-gray-600">Prev</span>
          <span className="text-gray-400 font-mono tabular-nums">${quote.pc.toFixed(2)}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-gray-600">High</span>
          <span className="text-gray-400 font-mono tabular-nums">${quote.h.toFixed(2)}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-gray-600">Low</span>
          <span className="text-gray-400 font-mono tabular-nums">${quote.l.toFixed(2)}</span>
        </div>
      </div>
    </div>
  );
}
