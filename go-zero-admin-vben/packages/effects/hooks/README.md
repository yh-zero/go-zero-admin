# @vben/hooks

用于多个 `app` 公用的 hook，重新导出 `@vben-core/composables` 的能力，并提供应用配置、分页、页签等组合函数。业务通用 hooks 优先放在应用自己的业务目录。

## 用法

### 添加依赖

```bash
# 进入目标应用目录，例如 apps/xxxx-app
# cd apps/xxxx-app
pnpm add @vben/hooks
```

### 使用

```ts
import { useNamespace } from '@vben/hooks';
```
