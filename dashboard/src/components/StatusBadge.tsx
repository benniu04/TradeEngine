import type { Order } from '../api/types';

const statusStyles: Record<Order['status'], string> = {
  pending: 'bg-yellow-500/10 text-yellow-500',
  validated: 'bg-yellow-500/10 text-yellow-500',
  open: 'bg-blue-500/10 text-blue-400',
  partial: 'bg-blue-500/10 text-blue-400',
  executed: 'bg-green-500/10 text-green-400',
  settled: 'bg-green-500/10 text-green-400',
  rejected: 'bg-red-500/10 text-red-400',
};

export function StatusBadge({ status }: { status: Order['status'] }) {
  return (
    <span className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-medium uppercase tracking-wide ${statusStyles[status]}`}>
      {status}
    </span>
  );
}
