# 登录猫咪素材与生成器

登录页使用本地托管的 Rive 动画、WASM 和静态备用图。动作素材由 ImageGen 为本项目生成，造型参考[登录猫咪视频](https://www.douyin.com/video/7698291534474562856)。原图仅机械裁切并对齐，各帧脚掌基线为 460px、足部水平中心为 256px。Rive 运行库使用 MIT 许可证。

## 从图集生成

唯一维护的源图为 [login-cat-atlas.png](../src/assets/login-cat-atlas.png)，宽高均为 1536px，3×3 等分。使用透明 RGBA8、非交错静态 PNG，宽高须能被 3 整除。各帧保持同一身体基线和缩放，按从左到右、从上到下排列：

| 第 1 列 | 第 2 列 | 第 3 列 |
|---|---|---|
| 正面闲置 | 向画面左看 | 向画面右看 |
| 半抬前爪 | 完全捂眼 | 放爪至胸前 |
| 成功挥爪 | 失败低头 | 正面闭眼 |

从 `go-zero-admin-react/` 执行：

```powershell
node scripts/build-login-cat.mjs public/mascots/login-cat.riv src/assets/login-cat-atlas.png
node --test scripts/pngSpriteAtlas.test.mjs
```

[生成器](build-login-cat.mjs)将九帧嵌入 `public/mascots/login-cat.riv`，以第一帧输出同目录的 `login-cat.png` 作为备用图。修改源图后重新生成两份产物。也可在输出路径后按表格顺序传入九张同尺寸 PNG；模块入口为 `buildLoginCat(imagePaths, outputPath)`。输出使用独立路径，保留源图。

## 接入接口

| 输入 | 类型与行为 |
|---|---|
| `is_hands_up` | Boolean，密码框聚焦后抬爪捂眼，失焦后放下。动画只接收聚焦状态和登录结果，不接收密码内容。 |
| `look_direction` | Number，`-1/0/1` 分别为左、正、右；`[-0.25, 0.25]` 为正面死区。捂眼与结果状态期间保持头向。 |
| `success`、`fail` | Trigger，从任意状态反馈结果，约一秒后回到闲置；捂眼输入仍为真时恢复捂眼。 |

Artboard 为 `cat_artboard`（尺寸与单帧一致），状态机为 `Login Machine`。生成器按 Rive 公开格式写入[文件头](https://github.com/rive-app/rive-runtime/blob/main/include/rive/runtime_header.hpp)、[对象定义](https://github.com/rive-app/rive-runtime/tree/main/include/rive/generated)和[过渡标志](https://github.com/rive-app/rive-runtime/blob/main/include/rive/animation/state_transition_flags.hpp)。

改动后用浏览器在登录页检查左右转头、密码聚焦与快速切换、成功和失败反馈、窄屏隐藏，以及资源无法加载时的备用图。桌面动画就绪时，登录成功反馈约 1.1 秒后继续跳转；窄屏或资源未就绪时直接完成。集成入口见 [useLoginFeedback.ts](../src/pages/login/useLoginFeedback.ts) 和 [RiveLoginCat.tsx](../src/pages/login/RiveLoginCat.tsx)。
