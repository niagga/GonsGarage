import type { PartUOM } from '@/types/parts';

export const PART_UOM_OPTIONS = [
  { value: 'unit', label: 'Unidade (unit)' },
  { value: 'liter', label: 'Litro (liter)' },
] as const satisfies ReadonlyArray<{ value: PartUOM; label: string }>;
