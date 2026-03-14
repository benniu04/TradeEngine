const USERS = [
  { id: '11111111-1111-1111-1111-111111111111', name: 'alice' },
  { id: '22222222-2222-2222-2222-222222222222', name: 'bob' },
];

interface Props {
  selectedId: string;
  onChange: (id: string) => void;
}

export function UserSelector({ selectedId, onChange }: Props) {
  return (
    <select
      value={selectedId}
      onChange={(e) => onChange(e.target.value)}
      className="bg-gray-800 border border-gray-700 text-white rounded-lg px-3 py-2 text-sm focus:border-[#00C805] focus:outline-none"
    >
      {USERS.map((u) => (
        <option key={u.id} value={u.id}>
          {u.name}
        </option>
      ))}
    </select>
  );
}
