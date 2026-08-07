# 渲染、匹配验证与输出（12）

## 1. 渲染

```bash
skills/draw-mermaid/scripts/render-mermaid.sh figure-01-order-flow.mmd figures/ figure-01-order-flow
```

- 同时产出 `figure-01-order-flow.svg` 与 `.png`（源 `.mmd` 一并保存）；
- 后端自动选择：mermaid.ink 服务器 → 本地 mermaid-cli（mmdc）；
- 强制后端：`MERMAID_BACKEND=server|local`；服务器地址 `MERMAID_SERVER=...`；
- 本地后端需 Chrome（puppeteer）：`PUPPETEER_EXECUTABLE_PATH=...` 指定；
- PNG 失败时用 `svg-to-png-cjk.cjs` 从 SVG 转：`node svg-to-png-cjk.cjs x.svg x.png 2`。

## 2. 匹配与微调

1. 渲染后 **Read 图片**（PNG/SVG）与用户要求比对；
2. 检查：元素齐全？方向/层级对？标签是否截断/溢出？配色符合样式规范？
3. 差异 → 微调 `.mmd` 重渲（文件已保存，直接改）；
4. 图集逐图检查：自足性、交叉引用、跨图一致（配色/字号/编号）。

## 3. 质量检查清单

- [ ] 每图：标题 + 渲染图 + 简要说明；
- [ ] 引用与源文件同目录、相对路径；
- [ ] PNG 与 SVG 同时产出；
- [ ] 图集交叉引用一致（`▶ 见图N` 指向存在）；
- [ ] WBS/甘特过了量测三判据（measure-svg-layout.py）；
- [ ] 大图只用 SVG 交付（PNG 超限时）且预览可用。

## 4. HTML 输出

### 4.1 单个 HTML 文档

```html
<h3>图 1：订单流程</h3>
<a href="figure-01-order-flow.svg" target="_blank" rel="noopener">
  <img src="figure-01-order-flow.png" alt="订单流程">
</a>
<p>说明：……</p>
```

### 4.2 PNG 与 SVG 双引用（同一机制）

⚠️ **PNG 与 SVG 引用必须用同一机制**：Markdown 图片 `![]()` 与内联 HTML `<a>` 走不同路径解析管线（有的渲染器代理 Markdown 图片 URL 却透传 HTML href），混用会导致两条路径不一致。

- **首选：全内联 HTML**：`<a href=x.svg target=_blank rel=noopener><img src=x.png></a>`（点图开 SVG 新标签，可无损放大）；
- **回退：全纯 Markdown**：`[![alt](x.png)](x.svg)`（渲染器剥 HTML 时同标签打开）；
- 不要混用两种机制。

### 4.3 嵌入 Markdown 文档（最佳实践）

- 默认引用 PNG（预览友好），SVG 同时产出供放大；
- 同一文档内所有图统一用同一种引用机制。

## 5. 交付物

- 渲染图（PNG+SVG）+ `.mmd` 源文件 + HTML 文档，同一目录；
- HTML 通过相对路径引用图片（可整体移动）。
