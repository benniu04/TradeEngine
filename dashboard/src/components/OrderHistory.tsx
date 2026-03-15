import type { Order } from '../api/types';
import { StatusBadge } from './StatusBadge';

function formatUSD(value: number): string {
  return value.toLocaleString('en-US', { style: 'currency', currency: 'USD' });
}

function timeAgo(dateStr: string): string {
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
  if (seconds < 60) return 'just now';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

interface Props {
  orders: Order[];
}

export function OrderHistory({ orders }: Props) {
  const sorted = [...orders].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  );

  return (
    <div>
      <p className="text-sm text-gray-500 mb-3">Order History</p>

      {sorted.length === 0 ? (
        <p className="text-gray-600 text-sm">No orders yet</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-[11px] text-gray-600 uppercase tracking-wider">
                <th className="text-left py-2 font-medium">Time</th>
                <th className="text-left py-2 font-medium">Symbol</th>
                <th className="text-left py-2 font-medium">Side</th>
                <th className="text-right py-2 font-medium">Filled</th>
                <th className="text-right py-2 font-medium">Price</th>
                <th className="text-right py-2 font-medium">Total</th>
                <th className="text-right py-2 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((o, i) => {
                const total = o.quantity * parseFloat(o.price);
                const isBuy = o.side === 'buy';
                return (
                  <tr
                    key={o.id}
                    className={i % 2 === 0 ? 'bg-white/2' : ''}
                  >
                    <td
                      className="py-2.5 text-gray-500 text-xs"
                      title={new Date(o.created_at).toLocaleString()}
                    >
                      {timeAgo(o.created_at)}
                    </td>
                    <td className="py-2.5 text-white font-medium">{o.symbol}</td>
                    <td className="py-2.5">
                      <span className="flex items-center gap-1.5">
                        <span className={`w-1.5 h-1.5 rounded-full ${isBuy ? 'bg-[#00C805]' : 'bg-red-500'}`} />
                        <span className="text-gray-400 text-xs">{o.side}</span>
                      </span>
                    </td>
                    <td className="py-2.5 text-right font-mono tabular-nums text-gray-300 text-xs">
                      {o.filled_quantity}/{o.quantity}
                    </td>
                    <td className="py-2.5 text-right font-mono tabular-nums text-gray-300">
                      {formatUSD(parseFloat(o.price))}
                    </td>
                    <td className="py-2.5 text-right font-mono tabular-nums text-white">
                      {formatUSD(total)}
                    </td>
                    <td className="py-2.5 text-right">
                      <StatusBadge status={o.status} />
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
