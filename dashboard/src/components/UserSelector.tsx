const USERS = [
  { id: '11111111-1111-1111-1111-111111111111', name: 'alice', color: '#00C805' },
  { id: '22222222-2222-2222-2222-222222222222', name: 'bob', color: '#3B82F6' },
];

interface Props {
  selectedId: string;
  onChange: (id: string) => void;
}

export function UserSelector({ selectedId, onChange }: Props) {
  const selected = USERS.find((u) => u.id === selectedId) ?? USERS[0];

  return (
    <div className="flex items-center gap-2">
      <div
        className="w-6 h-6 rounded-full flex items-center justify-center text-[10px] font-bold text-black shrink-0"
        style={{ backgroundColor: selected.color }}
      >
        {selected.name[0].toUpperCase()}
      </div>
      <div className="relative">
        <select
          value={selectedId}
          onChange={(e) => onChange(e.target.value)}
          className="appearance-none bg-transparent border border-gray-800 text-gray-300 rounded-md pl-2 pr-7 py-1 text-xs focus:border-gray-600 focus:outline-none cursor-pointer"
        >
          {USERS.map((u) => (
            <option key={u.id} value={u.id} className="bg-gray-900">
              {u.name}
            </option>
          ))}
        </select>
        <svg className="absolute right-2 top-1/2 -translate-y-1/2 w-3 h-3 text-gray-500 pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </div>
    </div>
  );
}
