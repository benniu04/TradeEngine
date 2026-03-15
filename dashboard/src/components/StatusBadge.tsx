import type { Order } from '../api/types';

const statusStyles: Record<Order['status'], string> = {
  pending: 'bg-yellow-500/20 text-yellow-400',
  validated: 'bg-yellow-500/20 text-yellow-400',
  open: 'bg-blue-500/20 text-blue-400',
  partial: 'bg-blue-500/20 text-blue-400',
  executed: 'bg-green-500/20 text-green-400',
  settled: 'bg-green-500/20 text-green-400',
  rejected: 'bg-red-500/20 text-red-400',
};

export function StatusBadge({ status }: { status: Order['status'] }) {
  return (
    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${statusStyles[status]}`}>
      {status}
    </span>
  );
}
