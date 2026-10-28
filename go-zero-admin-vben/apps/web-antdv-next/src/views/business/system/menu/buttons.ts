import type { MenuButton } from '#/api/business/system/types';

export function prepareMenuButtons(buttons: MenuButton[]): MenuButton[] {
  const names = new Set<string>();
  return buttons.map((button) => {
    const name = button.name.trim();
    const desc = button.desc.trim();
    if (!/^[A-Za-z][\w-]{0,63}$/.test(name))
      throw new Error(
        '按钮标识需以英文字母开头，只能包含字母、数字、下划线或短横线，最多 64 个字符',
      );
    if (!desc || desc.length > 100)
      throw new Error('请输入按钮名称，最多 100 个字符');
    if (names.has(name)) throw new Error(`按钮标识重复：${name}`);
    names.add(name);
    return { ...button, name, desc };
  });
}

export function changedMenuButtons(before: MenuButton[], after: MenuButton[]) {
  const current = new Map(
    after.filter((button) => button.ID).map((button) => [button.ID, button]),
  );
  return before.filter(
    (button) =>
      button.ID &&
      (!current.has(button.ID) || current.get(button.ID)!.name !== button.name),
  );
}
