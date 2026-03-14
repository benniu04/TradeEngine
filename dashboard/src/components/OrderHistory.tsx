import type { Order } from '../api/types';
import { StatusBadge } from './StatusBadge';

function formatUSD(value: number): string {
  return value.toLocaleString('en-US', { style: 'currency', currency: 'USD' });
}

interface Props {
  orders: Order[];
}

export function OrderHistory({ orders }: Props) {
  const sorted = [...orders].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  );

  return (
    <div className="bg-gray-900 rounded-xl p-6">
      <p className="text-white text-lg font-semibold mb-4">Order History</p>

      {sorted.length === 0 ? (
        <p className="text-gray-500 text-sm">No orders yet</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-gray-400 text-xs border-b border-gray-800">
                <th className="text-left py-2 font-medium">Time</th>
                <th className="text-left py-2 font-medium">Symbol</th>
                <th className="text-left py-2 font-medium">Side</th>
                <th className="text-right py-2 font-medium">Qty</th>
                <th className="text-right py-2 font-medium">Price</th>
                <th className="text-right py-2 font-medium">Total</th>
                <th className="text-right py-2 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((o) => {
                const total = o.quantity * parseFloat(o.price);
                return (
                  <tr key={o.id} className="border-b border-gray-800/50 hover:bg-gray-800/30">
                    <td className="py-3 text-gray-400">
                      {new Date(o.created_at).toLocaleString()}
                    </td>
                    <td className="py-3 text-white font-medium">{o.symbol}</td>
                    <td className="py-3">
                      <span className={o.side === 'buy' ? 'text-[#00C805]' : 'text-red-400'}>
                        {o.side.toUpperCase()}
                      </span>
                    </td>
                    <td className="py-3 text-right text-white">{o.quantity}</td>
                    <td className="py-3 text-right text-white">{formatUSD(parseFloat(o.price))}</td>
                    <td className="py-3 text-right text-white">{formatUSD(total)}</td>
                    <td className="py-3 text-right">
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
