/* eslint-disable vue/one-component-per-file -- Small controls exercise the permission editor workflow. */
import type { App } from 'vue';

import { createApp, h, nextTick, ref } from 'vue';

import { afterEach, describe, expect, it, vi } from 'vitest';

import PermissionOptions from './permission-options.vue';

vi.mock('antdv-next', async () => {
  const { defineComponent, h } = await import('vue');
  return {
    Space: defineComponent({
      setup:
        (_, { slots }) =>
        () =>
          h('div', slots.default?.()),
    }),
    Button: defineComponent({
      props: { disabled: Boolean },
      setup:
        (props, { attrs, slots }) =>
        () =>
          h(
            'button',
            { ...attrs, disabled: props.disabled },
            slots.default?.(),
          ),
    }),
    Input: defineComponent({
      props: { value: String, disabled: Boolean },
      emits: ['update:value'],
      setup:
        (props, { emit }) =>
        () =>
          h('input', {
            value: props.value,
            disabled: props.disabled,
            onInput: (event: Event) =>
              emit('update:value', (event.target as HTMLInputElement).value),
          }),
    }),
    CheckboxGroup: defineComponent({
      props: {
        options: { type: Array, default: () => [] },
        value: { type: Array, default: () => [] },
        disabled: Boolean,
      },
      emits: ['update:value'],
      setup:
        (props, { emit }) =>
        () =>
          h(
            'div',
            props.options.map((entry) => {
              const option = entry as { label: string; value: number };
              return h('label', [
                h('input', {
                  type: 'checkbox',
                  value: option.value,
                  checked: props.value.includes(option.value),
                  disabled: props.disabled,
                  onChange: (event: Event) =>
                    emit(
                      'update:value',
                      (event.target as HTMLInputElement).checked
                        ? [...props.value, option.value]
                        : props.value.filter((value) => value !== option.value),
                    ),
                }),
                option.label,
              ]);
            }),
          ),
    }),
  };
});

describe('grouped permission editor', () => {
  let app: App | undefined;
  let element: HTMLDivElement;
  afterEach(() => {
    app?.unmount();
    element?.remove();
  });
  it('keeps other groups checked when toggling a checkbox and locks every control while saving', async () => {
    const selected = ref([2, 8]);
    const disabled = ref(false);
    element = document.createElement('div');
    document.body.append(element);
    app = createApp({
      render: () =>
        h(PermissionOptions, {
          modelValue: selected.value,
          original: [2, 8],
          disabled: disabled.value,
          options: [
            { value: 1, group: '用户', label: '新增 create' },
            { value: 2, group: '用户', label: '修改 update' },
            { value: 8, group: '角色', label: '修改 update' },
          ],
          'onUpdate:modelValue': (value: number[]) => {
            selected.value = value;
          },
        }),
    });
    app.mount(element);
    const checkbox = (id: number) =>
      element.querySelector<HTMLInputElement>(
        `input[type="checkbox"][value="${id}"]`,
      )!;
    checkbox(1).click();
    await nextTick();
    expect([...selected.value].sort()).toEqual([1, 2, 8]);
    expect(checkbox(8).checked).toBe(true);
    checkbox(8).click();
    await nextTick();
    expect([...selected.value].sort()).toEqual([1, 2]);
    expect(element.textContent).toContain('已选 2 项 · 新增 1 项 · 撤销 1 项');
    disabled.value = true;
    await nextTick();
    expect(
      [...element.querySelectorAll('button')].every(
        (button) => button.disabled,
      ),
    ).toBe(true);
    expect(
      [...element.querySelectorAll('input')].every((input) => input.disabled),
    ).toBe(true);
    checkbox(1).click();
    expect([...selected.value].sort()).toEqual([1, 2]);
  });
  it('searches group labels and modifies only visible options while keeping other selected permissions', async () => {
    const selected = ref([2, 8]);
    element = document.createElement('div');
    document.body.append(element);
    app = createApp({
      render: () =>
        h(PermissionOptions, {
          modelValue: selected.value,
          original: [2, 8],
          options: [
            { value: 1, group: '用户', label: '新增 create' },
            { value: 2, group: '用户', label: '修改 update' },
            { value: 8, group: '角色', label: '修改 update' },
          ],
          'onUpdate:modelValue': (value: number[]) => {
            selected.value = value;
          },
        }),
    });
    app.mount(element);
    const input = element.querySelector('input')!;
    input.value = 'create';
    input.dispatchEvent(new Event('input'));
    await nextTick();
    expect(element.querySelectorAll('section')).toHaveLength(1);
    const findButton = (text: string) =>
      [...element.querySelectorAll('button')].find(
        (button) => button.textContent === text,
      )!;
    findButton('全选本组').click();
    await nextTick();
    expect(selected.value).toEqual([2, 8, 1]);
    expect(element.textContent).toContain('已选 3 项 · 新增 1 项 · 撤销 0 项');
    expect(element.querySelector('details')?.textContent).toContain(
      '新增：用户 · 新增 create',
    );
    findButton('取消本组').click();
    await nextTick();
    expect(selected.value).toEqual([2, 8]);
    expect(element.querySelector('details')).toBeNull();
    input.value = '角色';
    input.dispatchEvent(new Event('input'));
    await nextTick();
    expect(element.querySelectorAll('section')).toHaveLength(1);
    expect(element.querySelector('section')?.textContent).toContain('角色');
  });
});
