import type { User, Position } from '../api/types';

function formatUSD(value: number): string {
  return value.toLocaleString('en-US', { style: 'currency', currency: 'USD' });
}

const COLORS = ['#00C805', '#3B82F6', '#F59E0B', '#EF4444', '#8B5CF6', '#EC4899', '#06B6D4'];

interface Props {
  user: User;
  positions: Position[];
}

export function PortfolioOverview({ user, positions }: Props) {
  const cash = parseFloat(user.balance);
  const holdingsValue = positions.reduce(
    (sum, p) => sum + p.quantity * parseFloat(p.avg_cost),
    0
  );
  const totalValue = cash + holdingsValue;

  return (
    <div>
      <p className="text-sm text-gray-500 mb-1">Portfolio Value</p>
      <p className="text-5xl font-bold tracking-tight text-white mb-4">{formatUSD(totalValue)}</p>

      <div className="flex items-center gap-6 text-sm mb-6">
        <div className="flex items-center gap-2">
          <span className="text-gray-500">Cash</span>
          <span className="text-white font-medium">{formatUSD(cash)}</span>
        </div>
        <span className="text-gray-800">|</span>
        <div className="flex items-center gap-2">
          <span className="text-gray-500">Holdings</span>
          <span className="text-white font-medium">{formatUSD(holdingsValue)}</span>
        </div>
      </div>

      {positions.length > 0 && (
        <div className="flex flex-wrap gap-3">
          {positions.map((p, i) => {
            const value = p.quantity * parseFloat(p.avg_cost);
            const color = COLORS[i % COLORS.length];
            return (
              <div
                key={p.id}
                className="flex items-center gap-3 bg-white/3 rounded-lg px-4 py-3"
              >
                <div
                  className="w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold text-black shrink-0"
                  style={{ backgroundColor: color }}
                >
                  {p.symbol[0]}
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-white font-medium text-sm">{p.symbol}</span>
                    <span className="text-gray-500 text-xs">{p.quantity} shares</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-white text-sm font-mono tabular-nums">{formatUSD(value)}</span>
                    <span className="text-gray-600 text-xs">@ {formatUSD(parseFloat(p.avg_cost))}</span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
