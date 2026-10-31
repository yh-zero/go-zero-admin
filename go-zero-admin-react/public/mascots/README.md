# 登录页橘猫素材

猫咪动作图由 imagegen 为本项目生成，造型参考用户提供的[登录猫咪视频](https://www.douyin.com/video/7698291534474562856)：橘色虎斑、奶油色胸口与脚掌、粉色肉垫、圆润坐姿和细密毛发。

## 来源与重新生成

- 唯一图片来源：`../../src/assets/login-cat-atlas.png`，3 × 3 动作图集，每格 512 × 512。
- 原始生成图仅做机械裁切与坐姿对齐；各帧脚掌基线为 460px，足部水平中心对齐为 256px，未缩放或重新描绘角色。
- `login-cat.riv` 与 `login-cat.png` 均由脚本生成，不手工编辑。
- `.riv` 使用项目脚本按 Rive 官方公开格式独立编译，包含图片、动画轨道和状态机。

从前端目录执行：

```powershell
node scripts/build-login-cat.mjs public/mascots/login-cat.riv src/assets/login-cat-atlas.png
```

详细格式与维护说明见 `../../scripts/build-login-cat.md`。Rive 运行库按其 MIT 许可证使用。

## 交互接口

- Artboard：`cat_artboard`；State machine：`Login Machine`。
- `is_hands_up`：Boolean，分段抬爪捂眼、放爪。
- `look_direction`：Number，`-1` 左看、`0` 正面、`1` 右看。
- `success` / `fail`：Trigger，开心挥爪或低头轻摇，约一秒后恢复。
- 正面闲置包含轻微呼吸和间歇眨眼。

这是图片姿态与 Rive 动画轨道组合的角色，不是实时三维模型。素材文件和 WASM 都在本地站点提供；密码内容不会传入动画组件。
