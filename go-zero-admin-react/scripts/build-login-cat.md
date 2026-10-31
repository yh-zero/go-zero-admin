# 登录猫咪 Rive 生成器

`build-login-cat.mjs` 使用 Node 内置模块，将对齐的透明 PNG 图集编译为真正的 `.riv` 文件。不需要登录 Rive Editor，不增加项目依赖，也不复制第三方动画模板。

项目以 `src/assets/login-cat-atlas.png` 为唯一图片来源；从前端目录运行以下唯一入口，同时生成 Rive 动画与静态闲置备用图：

```powershell
node scripts/build-login-cat.mjs public/mascots/login-cat.riv src/assets/login-cat-atlas.png
```

输出为 `public/mascots/login-cat.riv` 和 `public/mascots/login-cat.png`，不要手动编辑生成物。图集按从左到右、从上到下严格等分为 3×3；宽高必须能被 3 整除。当前来源为 1536×1536，每格 512×512。原图若排列不规则，应先机械对齐为等分图集，本入口不会猜测边界或缩放、修饰猫咪像素。

`pngSpriteAtlas.mjs` 仅支持 RGBA8 非交错静态 PNG，验证每个 chunk 的 CRC、行滤波、解压长度和尺寸上限，保留色彩元数据与每个 RGBA 通道。裁出的九张 PNG 直接嵌入 `.riv`，不额外保存九份文件。

## 输入素材

九张 PNG 必须尺寸相同，猫咪的身体、坐姿基线和缩放保持对齐，顺序如下：

| 索引 | 文件示例 | 姿态 |
|---|---|---|
| 0 | `idle.png` | 正面闲置 |
| 1 | `look-left.png` | 朝画面左侧看 |
| 2 | `look-right.png` | 朝画面右侧看 |
| 3 | `cover-start.png` | 半抬前爪 |
| 4 | `cover.png` | 完全捂眼 |
| 5 | `cover-end.png` | 放爪至胸前 |
| 6 | `success.png` | 开心挥爪 |
| 7 | `failure.png` | 低头失落 |
| 8 | `blink.png` | 正面闭眼 |

也保留九张独立 PNG 的输入模式，路径替换为实际素材路径：

```powershell
node scripts/build-login-cat.mjs public/mascots/login-cat.riv idle.png look-left.png look-right.png cover-start.png cover.png cover-end.png success.png failure.png blink.png
```

生成器也导出 `buildLoginCat(imagePaths, outputPath)`，`imagePaths` 可为一个图集路径或九个帧路径。输出 Rive 及同目录同名 PNG 备用图；输入文件仅读取，请勿让输出覆盖源图集。

## 接入接口

- Artboard：`cat_artboard`，尺寸与 PNG 一致。
- State machine：`Login Machine`。
- `is_hands_up`：Boolean。`true` 从半抬爪过渡到捂眼并保持；`false` 经过放爪姿态回到闲置。
- `look_direction`：Number。`-1` 朝左、`0` 正面、`1` 朝右；`[-0.25, 0.25]` 为正面死区。捂眼和结果状态期间不响应头向变化。
- `success` / `fail`：Trigger。任何状态都可进入对应结果，约一秒后回到闲置；若捂眼输入仍为真，则恢复捂眼。

闲置正面六秒循环中有两次短眨眼，左看和右看保持对应姿态；闲置时约两像素上下呼吸。成功轻微弹起，失败轻微摇头。姿态切换使用约 40–50ms 的短混合，减少跳变和双影。过渡允许中途退出，避免用户快速提交时结果触发被混合过程吞掉。

这是由多张图片和 Rive 动画轨道组成的交互角色，细节取决于素材对齐质量；不是实时三维模型或骨骼绑定。

## 格式依据与验证

文件格式及字段编号依据 Rive 官方公开运行库：

- [文件头格式](https://github.com/rive-app/rive-runtime/blob/main/include/rive/runtime_header.hpp)
- [对象与属性定义](https://github.com/rive-app/rive-runtime/tree/main/include/rive/generated)
- [状态过渡标志](https://github.com/rive-app/rive-runtime/blob/main/include/rive/animation/state_transition_flags.hpp)

本轮使用项目实际安装的 `@rive-app/canvas@2.48.0` WASM 加载九图原型，核验输入类型、左右头向、捂眼与放爪、放爪途中重新聚焦、成功/失败后返回以及过渡期间触发结果。最终文件已实载并确认九张嵌入 PNG 均为 512×512。Node 不能替代浏览器的实际渲染，最终仍需检查画面、透明边缘和动作对齐。

PNG 辅助模块测试：

```powershell
node --test scripts/pngSpriteAtlas.test.mjs
```

六项测试覆盖全部五种 PNG 滤波、九格逐像素对应关系、色彩元数据保留、CRC 损坏、截断、非法滤波以及不支持的图像格式。最终图集与九格 RGBA 的 SHA-256 还经 Windows `System.Drawing` 独立解码对照一致；生成的静态备用图与原图第一格像素一致。
