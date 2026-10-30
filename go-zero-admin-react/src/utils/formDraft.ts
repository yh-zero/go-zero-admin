export interface FormDraftSession { entityKey: string | number; initialized: boolean }
interface FormDraftOptions<T extends object> {
  open: boolean;
  entityKey: string | number;
  values?: T;
  isTouched: (name: string) => boolean;
  forceReset?: boolean;
}

export function planFormDraft<T extends object>(session: FormDraftSession | undefined, options: FormDraftOptions<T>) {
  if (!options.open) return { session: undefined, reset: false, fields: [] };
  const sameEntity = session?.entityKey === options.entityKey;
  const hasValues = options.values !== undefined;
  const reset = Boolean(options.forceReset || !sameEntity || (!session?.initialized && hasValues));
  return {
    session: { entityKey: options.entityKey, initialized: hasValues || Boolean(sameEntity && session?.initialized) },
    reset,
    fields: Object.entries(options.values ?? {})
      .filter(([name]) => reset || !options.isTouched(name))
      .map(([name, value]) => ({ name, value, touched: false, errors: [], warnings: [] })),
  };
}
