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
      <div className="bg-gray-900 rounded-xl p-6 border border-gray-800">
        <p className="text-gray-500 text-sm">
          Market data unavailable — set FINNHUB_API_KEY to enable
        </p>
      </div>
    );
  }

  if (!quote || quote.c === 0) {
    return (
      <div className="bg-gray-900 rounded-xl p-6 border border-gray-800">
        <p className="text-gray-500 text-sm">Loading {symbol} quote...</p>
      </div>
    );
  }

  const isUp = quote.d >= 0;

  return (
    <div className="bg-gray-900 rounded-xl p-6 border border-gray-800">
      <div className="flex items-baseline justify-between mb-4">
        <h2 className="text-lg font-semibold text-white">{symbol}</h2>
        <span className="text-xs text-gray-500">Live Market Data</span>
      </div>

      <div className="flex items-baseline gap-3 mb-4">
        <span className="text-3xl font-bold text-white">
          ${quote.c.toFixed(2)}
        </span>
        <span className={`text-sm font-medium ${isUp ? 'text-[#00C805]' : 'text-red-400'}`}>
          {isUp ? '+' : ''}{quote.d.toFixed(2)} ({isUp ? '+' : ''}{quote.dp.toFixed(2)}%)
        </span>
      </div>

      <div className="grid grid-cols-2 gap-3 text-sm">
        <div className="flex justify-between">
          <span className="text-gray-500">Open</span>
          <span className="text-white">${quote.o.toFixed(2)}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-gray-500">Prev Close</span>
          <span className="text-white">${quote.pc.toFixed(2)}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-gray-500">High</span>
          <span className="text-[#00C805]">${quote.h.toFixed(2)}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-gray-500">Low</span>
          <span className="text-red-400">${quote.l.toFixed(2)}</span>
        </div>
      </div>
    </div>
  );
}
