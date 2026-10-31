import type { ThemeConfig } from 'antd';

export const appTheme: ThemeConfig = {
  token: {
    colorPrimary: '#2563eb', colorInfo: '#2563eb', colorSuccess: '#159570',
    colorWarning: '#d28a16', colorError: '#dc5263', colorText: '#172b4d',
    colorTextSecondary: '#6b7a90', colorBorder: '#dce3ed', colorBorderSecondary: '#e8edf4',
    colorBgLayout: '#f4f6fa', colorBgContainer: '#ffffff',
    borderRadius: 8, borderRadiusLG: 12, controlHeight: 36, fontSize: 14,
    fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", "Microsoft YaHei", sans-serif',
    boxShadow: '0 8px 28px rgba(21, 40, 75, 0.08)',
  },
  components: {
    Layout: { bodyBg: '#f4f6fa', siderBg: '#101c32', headerBg: '#ffffff' },
    Menu: {
      darkItemBg: '#101c32', darkSubMenuItemBg: '#101c32', darkItemColor: '#aebbd0',
      darkItemSelectedBg: '#2563eb', darkItemSelectedColor: '#ffffff',
      darkItemHoverBg: '#1b2b46', darkItemHoverColor: '#ffffff',
      itemHeight: 44, itemMarginInline: 12, itemBorderRadius: 8, iconSize: 17,
    },
    Card: { headerFontSize: 15, headerHeight: 58, paddingLG: 24 },
    Table: {
      headerBg: '#f7f9fc', headerColor: '#5c6d85', headerSplitColor: 'transparent',
      rowHoverBg: '#f5f8ff', borderColor: '#edf0f5', cellPaddingBlock: 15,
      cellPaddingInline: 18, fontSize: 13,
    },
    Button: { fontWeight: 500, primaryShadow: '0 3px 8px rgba(37, 99, 235, 0.15)' },
    Input: { activeShadow: '0 0 0 3px rgba(37, 99, 235, 0.08)' },
    Select: { optionSelectedBg: '#edf3ff' },
    Form: { labelColor: '#465773', itemMarginBottom: 22 },
    Tag: { defaultBg: '#f2f5fa', defaultColor: '#62718a' },
    Breadcrumb: { itemColor: '#8a97ab', lastItemColor: '#324663', separatorColor: '#c3ccda' },
    Modal: { titleFontSize: 17 },
    Descriptions: { labelBg: '#f7f9fc', titleColor: '#172b4d' },
  },
};
