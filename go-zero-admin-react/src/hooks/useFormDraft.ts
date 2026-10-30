import { useCallback, useEffect, useRef } from 'react';
import type { FormInstance } from 'antd';
import { planFormDraft } from '../utils/formDraft';
import type { FormDraftSession } from '../utils/formDraft';

interface FormDraftOptions<Values extends object> {
  entityKey: string | number;
  open?: boolean;
  values?: Partial<Values>;
}

export function useFormDraft<Values extends object>(form: FormInstance<Values>, {
  entityKey, open = true, values,
}: FormDraftOptions<Values>) {
  const session = useRef<FormDraftSession | undefined>(undefined);
  const apply = useCallback((nextValues: Partial<Values> | undefined, forceReset = false) => {
    const update = planFormDraft(session.current, {
      open, entityKey, values: nextValues, forceReset,
      isTouched: (name) => form.isFieldTouched(name as Parameters<typeof form.isFieldTouched>[0]),
    });
    if (update.reset) form.resetFields();
    // setFieldsValue 会把服务端回填也标成 touched，必须显式保持未编辑状态。
    if (update.fields.length) form.setFields(update.fields as Parameters<typeof form.setFields>[0]);
    session.current = update.session;
  }, [form, open, entityKey]);
  useEffect(() => { apply(values); }, [apply, values]);
  return useCallback((savedValues: Partial<Values>) => { apply(savedValues, true); }, [apply]);
}
