# 文档站点构建（Hugo）

> `docs/` 既是 Markdown 内容源，也是 Hugo 项目根。内容一律**挂载而非复制** —— Markdown 树保持纯净
> （无 frontmatter、保留仓库原生相对链接），构建产物落在 `docs/public/`（已 gitignore）。
> 机制细节见 `create-docs` 技能的 `references/hugo-site.md`。

---

## 1. 构建与预览

```bash
# 构建到 docs/public（Hugo 配置在 docs/ 下，必须从 docs/ 运行）
cd docs && hugo

# 本地预览（live reload）
cd docs && hugo server

# 或经技能脚本（等效，附带产物报告）
python3 .qoder/skills/create-docs/scripts/scaffold-hugo.py --action build --root .
```

当前基线：`hugo v0.141.0+extended`，29 页 / 74 静态文件 / 零 warning。

## 2. 站点层文件归属

| 路径 | 归属 | 说明 |
|---|---|---|
| `docs/<type>/**.md` | 文档 | 内容本体，站点层从不改写 |
| `docs/hugo.toml` | 共用 | `# >>> speckit:mounts` 块由脚本再生成；**块外内容归人工** |
| `docs/layouts/**` | 脚手架（改后归你） | 极简自包含模板，无外部主题、无网络依赖 |
| `docs/static/css/site.css` | 脚手架（改后归你） | 单一样式表 |
| `docs/public/` | Hugo | 构建产物，永不提交 |

`layouts/`、`static/`、`public/`、`resources/` 等目录**不是文档**，reconcile 不得当作内容归类。

## 3. 本项目的四处人工挂载（勿删）

`scaffold-hugo.py` 只按 `docs/` 的**子目录**计算挂载，且假定链接均为 Markdown 语法。本项目两点不符，
故在 `hugo.toml` 的托管块**之上**手工加了 4 条挂载 —— 重新生成挂载块不会影响它们，但**手工删除会导致回归**：

| 挂载 | 解决的问题 |
|---|---|
| `.` → `content`（`includeFiles = ["/*.md"]`） | `docs/` 根部 10 篇 upstream 文档（architecture/overview/glossary/api-guide/roadmap/threat-model/observability/request-parking/api-style-guide/code-style-guide）本不在发布范围，且指向它们的相对链接全为死链 |
| `figures` → `static/concepts/figures` | `concepts/core-concepts.md` 用 raw HTML `<img src="../figures/…">` 实现点图放大 |
| `figures` → `static/reference/figures` | `reference/actor-lifecycle-flows.md` 同上 |
| `figures` → `static/overview/figures` | 根级 `overview.md` 用裸相对路径 `src="figures/…"` |

**为什么需要**：render hooks 只改写 Markdown 语法的链接与图片；raw HTML 的相对路径被原样输出，
而页面发布后比源文件深一层（`/concepts/core-concepts/`），于是解析到不存在的 `/concepts/figures/`。

## 4. 无 frontmatter 与页面标题

本项目 Markdown 刻意不带 frontmatter，因此 Hugo 的 `.Title`/`.LinkTitle` 为空，导航、首页卡片与页面
列表的链接文字会全部空白。`docs/layouts/partials/page-title.html` 以「首个 H1 → 文件名 → section 名」
兜底，`baseof.html`/`list.html`/`index.html` 均经它取标题。新增模板若要显示标题，请一并调用该 partial，
不要改回 `.Title`，也不要为迁就 Hugo 给 Markdown 补 frontmatter。

## 5. 发布范围与注意事项

- 发布范围是 `docs/` 全部（六型 + `notes/` + 媒体）。**任何放进 `docs/` 的内容都会公开发布** ——
  本仓有公开 remote，敏感/内网素材不得置于 `docs/` 下。
- `notes/` 的 `status: draft` 不是 Hugo 的 `draft`，草稿同样会发布；要排除某区就删掉它的挂载并在
  `hugo.toml` 中记录原因。
- 指向仓库源码（`manifests/`、`hack/`、`tools/`）与仓库根文件的链接在站点上无法解析 —— 在仓库内浏览
  正确，属预期取舍；不要为此把源码挂进站点。
- CI：安装 Hugo extended → 在 `docs/` 下 `hugo --minify --baseURL <url>` → 发布 `docs/public`。
