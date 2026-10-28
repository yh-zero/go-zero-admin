import type { SelectValue } from 'antdv-next/dist/select/index';

import type { App } from 'vue';

import { createApp, h, nextTick, ref } from 'vue';

import Select from 'antdv-next/dist/select/index';
import { afterEach, describe, expect, it } from 'vitest';

import { membershipPositionOptions } from './helpers';

describe('membership uses real Select tag removal semantics', () => {
  let app: App | undefined;
  let element: HTMLDivElement | undefined;

  afterEach(() => {
    app?.unmount();
    element?.remove();
  });

  it('removes one retained disabled position while preserving the other', async () => {
    const positions = [1, 2, 3].map((id) => ({
      id,
      name: `P${id}`,
      code: `P${id}`,
      sort: 0,
      status: 2,
    }));
    const options = membershipPositionOptions(positions, [1, 2]);
    const value = ref([1, 2]);
    element = document.createElement('div');
    document.body.append(element);
    app = createApp({
      render: () =>
        h(Select, {
          mode: 'multiple',
          value: value.value,
          options,
          'onUpdate:value': (next: SelectValue) => {
            if (
              !Array.isArray(next) ||
              !next.every((item) => typeof item === 'number')
            ) {
              throw new TypeError(
                'Membership selection must contain numeric IDs',
              );
            }
            value.value = next;
          },
        }),
    });
    app.mount(element);
    await nextTick();
    const removeButtons = element.querySelectorAll<HTMLElement>(
      '.ant-select-selection-item-remove',
    );
    expect(removeButtons).toHaveLength(2);
    removeButtons[0]!.click();
    await nextTick();
    expect(value.value).toEqual([2]);
    expect(
      element.querySelectorAll('.ant-select-selection-item-remove'),
    ).toHaveLength(1);
    expect(options.find((item) => item.value === 3)?.disabled).toBe(true);
  });
});
