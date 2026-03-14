import type { User, Position } from '../api/types';

function formatUSD(value: number): string {
  return value.toLocaleString('en-US', { style: 'currency', currency: 'USD' });
}

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
    <div className="bg-gray-900 rounded-xl p-6">
      <p className="text-gray-400 text-sm mb-1">Portfolio Value</p>
      <p className="text-4xl font-bold text-[#00C805] mb-4">{formatUSD(totalValue)}</p>

      <div className="flex gap-6 mb-6">
        <div>
          <p className="text-gray-400 text-xs">Cash</p>
          <p className="text-white text-lg font-semibold">{formatUSD(cash)}</p>
        </div>
        <div>
          <p className="text-gray-400 text-xs">Holdings</p>
          <p className="text-white text-lg font-semibold">{formatUSD(holdingsValue)}</p>
        </div>
      </div>

      <div>
        <p className="text-gray-400 text-sm mb-3">Positions</p>
        {positions.length === 0 ? (
          <p className="text-gray-500 text-sm">No holdings yet</p>
        ) : (
          <div className="space-y-3">
            {positions.map((p) => {
              const value = p.quantity * parseFloat(p.avg_cost);
              return (
                <div
                  key={p.id}
                  className="flex items-center justify-between border-b border-gray-800 pb-3"
                >
                  <div>
                    <p className="text-white text-lg font-semibold">{p.symbol}</p>
                    <p className="text-gray-400 text-xs">
                      {p.quantity} shares @ {formatUSD(parseFloat(p.avg_cost))}
                    </p>
                  </div>
                  <p className="text-white font-medium">{formatUSD(value)}</p>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
