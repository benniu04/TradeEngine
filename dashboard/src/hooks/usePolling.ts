import { useEffect } from 'react';

export function usePolling(callback: () => void, intervalMs: number) {
  useEffect(() => {
    callback();
    const id = setInterval(callback, intervalMs);
    return () => clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [callback, intervalMs]);
}
