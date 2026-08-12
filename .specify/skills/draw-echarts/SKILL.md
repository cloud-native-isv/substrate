---
name: draw-echarts
description: |
  Use Apache ECharts to create data visualizations and output as standalone HTML documents.
  Use when the user mentions "ECharts", "echarts", "Apache ECharts", "数据图表", "图表",
  "柱状图", "折线图", "饼图", "散点图", "雷达图", "热力图", "仪表盘", "漏斗图",
  "K线图", "桑基图", "树图", "旭日图", "关系图", "状态机", "state machine",
  "gauge", "funnel", "sankey",
  "数据可视化", "data visualization", "仪表板", "dashboard", "堆叠图",
  "stacked chart", "环形图", "平行坐标", "箱线图", "boxplot"
skill_id: "<SKILL:.specify/skills/draw-echarts/SKILL.md>"
---

# Apache ECharts Data Visualization Skill

Create data visualizations using Apache ECharts, output as a self-contained HTML file that can be opened directly in any modern browser.

## Core Principles

### 1. Configuration-Driven Design
ECharts uses a declarative `option` object to configure charts. Every visualization starts from understanding the data and mapping it to the appropriate `option` structure. Focus on clear data-to-visual mappings.

### 2. Self-Contained Output
The output must be a **single HTML file** with all ECharts configuration and styles inline. No external dependencies beyond the pinned ECharts script. The file should work by simply opening in a browser.

**Version pinning & offline reproducibility**: always load a **pinned** ECharts version (e.g. `echarts@5.6.0`), never a floating `echarts@5` tag — floating tags break reproducibility (CDN behavior, ECharts changes). For **offline-critical or long-lived deliverables, the local copy must be REAL**: place `vendor/echarts.min.js` into the deliverable (use `scripts/vendor-echarts.sh <dir>` which downloads and verifies the file) — never ship a loader that references a file you did not actually place, and never ship an empty `vendor/` stub (it renders a blank canvas offline). The HTML loader references the vendored copy first and falls back to the pinned CDN via a dynamic `onerror` loader — **no `document.write`** (deprecated, blocking, CSP-hostile). For complex, regenerable deliverables, separate data and the `option` config into a `config.json` (single canonical format) with a generated `config.js` wrapper (see `scripts/sync-config.mjs`), so regeneration only edits the config (see Step 4).

### 3. ECharts Best Practices
Use a pinned ECharts 5.x version (e.g. `echarts@5.6.0`). Leverage built-in features: responsive resize, tooltip, legend, toolbox (save as image, data view). Use `dataset` for data management when data is tabular. See [echarts-guide.md](references/echarts-guide.md) for configuration patterns, including graph/state-machine recipes and label-overlap avoidance.

### 4. Rich Interactivity by Default
ECharts provides built-in interactivity (tooltip, legend toggle, zoom, data highlight). Enable these features by default. Add custom interactions only when explicitly requested.

## Workflow

This skill creates ECharts data visualizations based on user-provided data and requirements. Follow the steps below in order.

### Step 1: Understand Data & Requirements

Analyze the user's input to determine:

1. **Data structure**: What format is the data in? (Array, table, JSON, CSV, markdown table, etc.)
2. **Data dimensions**: How many variables? Categorical vs quantitative? Time-series?
3. **Visualization goal**: What story should the chart tell? (comparison, trend, distribution, relationship, composition, hierarchy, flow)
4. **Special needs**: Theme preference (light/dark)? Animation? Custom tooltip? Toolbox features?
5. **Multi-chart needs**: Does the user need multiple charts on one page? If so, plan grid layout.
6. **Delivery list**: If the task implies multiple views (e.g. an architecture overview, a state machine, a relationship view, a deployment view), enumerate the **full required view list up front** and keep it for coverage checking at delivery time. Every required view must be either delivered or explicitly waived with a stated reason (see "Coverage Discipline" in Step 2).

Data format handling:
- If data is in a markdown table or plain text table, parse it into ECharts-compatible format
- Prefer `dataset.source` for tabular data with multiple series
- If data has Chinese headers, preserve them for axis labels and legend
- If data volume is large (>50 rows), enable `dataZoom` for scrollable exploration

If critical information is missing, ask **one targeted question**.

### Step 2: Choose Chart Type

Match data characteristics and goals to the appropriate ECharts chart type:

| Goal | Data Type | Recommended Charts (type value) |
|------|-----------|---------------------------------|
| 比较 (Comparison) | Categorical | `bar`, `bar` (horizontal) |
| 趋势 (Trend) | Time-series | `line`, `line` (area) |
| 占比 (Composition) | Part-to-whole | `pie`, `treemap`, `sunburst` |
| 分布 (Distribution) | Quantitative | `scatter`, `boxplot`, `heatmap` |
| 关系 (Relationship) | Two+ quantitative | `scatter`, `graph` |
| 层次 (Hierarchy) | Tree/nested | `tree`, `treemap`, `sunburst` |
| 网络 (Network) | Nodes + Links | `graph`, `sankey` |
| 多维 (Multi-dim) | Multiple attributes | `radar`, `parallel` |
| 指标 (KPI) | Single value | `gauge` |
| 流程 (Funnel) | Stage conversion | `funnel` |
| 金融 (Finance) | OHLC data | `candlestick` |

### Graph / Network & State Machine Views

- **Force-directed `graph` is a relationship view, not an architecture diagram.** In force layout, nodes overlap freely and there are no subsystem boundaries or containment/hierarchy semantics. Use force layout for relationship exploration (who talks to whom, clustering). For **component/architecture views with subsystem boundaries**, prefer a fixed layout (`layout: 'none'` with explicit `x`/`y` per node, or `circular` for ring topologies) plus partitioned background zones (a `graphic` rect layer or a background `scatter` series) to visually group subsystems; or explicitly label the deliverable as a "关系视图 / relationship view" instead of presenting it as an architecture diagram.
- **Edge labels occlude in dense graphs.** Do not render all edge labels by default in force layouts with many edges. Default `edgeLabel` to hidden and reveal on hover via `emphasis.edgeLabel: { show: true }`, and/or enable `labelLayout: { hideOverlap: true }` avoidance. Provide a global toggle only when the user explicitly asks for always-on labels.
- **State machines (`graph` with `categories`):** give `[*]` start/end pseudo-nodes a **visible label** (`[*]`, or localized 开始/结束) — never an empty label — and style nodes by state class (steady/transition/exception states) through `categories[].itemStyle` so node fill/border matches the legend and the edge colors.
- **Inferred vs source-described edges (推断边 vs 源描述边):** when the source description does not specify a transition (e.g. "RUNNING 直接 Delete 未详述") and the model completes it, distinguish the inference from described facts visually — recommended: **dashed gray** (`lineStyle: { type: 'dashed', color: '#999' }`) for inferred/completed edges, **solid** colored edges for source-described transitions, **red dashed** for manual-intervention actions (e.g. operator Delete/恢复 on a CRASHED state that only appears in prose, not in the state graph). Add the distinction to the legend or a prominent in-chart footnote (`title.subtext` / `graphic`), not only in a page footer.
- **Correction / arbitration annotation (校正/仲裁标注):** when the model must arbitrate a conflict in the source description (e.g. old vs current architecture, two interpretations of a data path), do not silently correct: (1) present both readings with the chosen one marked and a timestamp where the description contains old and new sections; (2) label partition titles with evolution terms (e.g. "子系统A（已迁移）", "子系统B（演进）") or use a legend split "活跃关系 / 演进关系" instead of a bare "旧" category; (3) cite the correction basis (the source reference/出处) in the delivery notes so the arbitration is reproducible.
- **Explicit omission annotation:** when the model deliberately excludes part of the domain (undefined states, out-of-scope components, unavailable data), annotate the omission on the chart (e.g. `title.subtext`, `graphic` text footnote, or a page note: "PAUSED 态架构未定义故不含") — never omit silently.

### Scope Boundary: When ECharts Is Not the Right Tool

ECharts is a data-viz library, not a diagramming tool. **Deployment diagrams, sequence diagrams, and UML class diagrams** are outside its natural expression. For such views:
- Recommend the sibling diagram skills (draw-plantuml, draw-mermaid) to the user; or
- If the user insists on ECharts, deliver an approximate view (e.g. a `graph` for deployment topology) **and explicitly document the substitution tradeoff** in the delivery notes: what the view shows and what it cannot show (deployment layers, containment, temporal order).

Never silently substitute one view type for another.

### Coverage Discipline (多视图交付清单)

When the task requires a fixed set of views, keep the delivery list from Step 1 and check it **before delivering**: each required view is either delivered, or explicitly waived with a stated reason (out of scope, substituted by another view with tradeoff noted, or better served by a sibling skill). Do not discover missing views at delivery time.

If multiple chart types are needed, create multiple charts in the same HTML document or use ECharts `toolbox` for type switching.

### Step 3: Build ECharts Option

Based on the chosen chart type and data:

1. **Prepare data**: Format data as `dataset.source` (preferred for tabular data) or inline `series.data`
2. **Configure axes**: Set up `xAxis` and `yAxis` with proper types (`category`, `value`, `time`, `log`)
3. **Define series**: Specify chart type, data mapping (`encode` or direct data), and visual styling
4. **Add components**: title, tooltip, legend, toolbox, dataZoom as needed
5. **Apply styling**: Colors, itemStyle, emphasis effects, animation settings
6. **Shared design tokens**: for multi-chart deliverables (especially with a shared dark theme), define ONE design-token object (palette, theme, background, panel color, fonts, edge-style semantics) and reference it from every chart config — do not scatter hex colors per config (cross-chart consistency, single edit point)
7. **Responsive setup**: Add `window.resize` listener to call `chart.resize()`
8. **Annotate omissions**: If the model deliberately omits part of the domain (undefined states, out-of-scope items), add an explicit annotation (`title.subtext`, `graphic` footnote, or page note) — never omit silently

For ECharts configuration patterns, chart recipes, and component options, reference [echarts-guide.md](references/echarts-guide.md).

### Step 4: Assemble HTML Document

Package everything into a self-contained HTML file using the base template structure:

```html
<!DOCTYPE html>
<html lang="zh">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>[Chart Title]</title>
  <script>
    // ECharts loader: vendored local copy first (offline-capable), pinned CDN fallback.
    // The local copy MUST exist — run: bash scripts/vendor-echarts.sh <deliverable-dir>
    // Never use document.write for this loader.
    (function () {
      var s = document.createElement('script');
      s.src = 'vendor/echarts.min.js';
      s.async = false;
      s.onerror = function () {
        var c = document.createElement('script');
        c.src = 'https://cdn.jsdelivr.net/npm/echarts@5.6.0/dist/echarts.min.js';
        c.async = false;
        document.head.appendChild(c);
      };
      document.head.appendChild(s);
    })();
  </script>
  <style>
    /* Inline styles */
  </style>
</head>
<body>
  <div id="chart" style="width: 100%; height: 500px;"></div>
  <script>
    // ECharts initialization and configuration
  </script>
</body>
</html>
```

Key requirements:
- ECharts loaded from **pinned** CDN: `https://cdn.jsdelivr.net/npm/echarts@5.6.0/dist/echarts.min.js` (never a floating `echarts@5` tag); keep the version in a code comment. Where practical, add `integrity`/`crossorigin` (SRI) attributes for the pinned script
- **Offline-critical deliverables MUST ship a real `vendor/echarts.min.js`** (download + verify with `scripts/vendor-echarts.sh <dir>`); the loader references the local copy first and only falls back to CDN — no `document.write`
- For quick online-only charts, the CDN-primary order is acceptable, but never reference `vendor/` unless the file actually exists (an empty `vendor/` stub renders a blank canvas offline)
- All CSS inline in `<style>` block
- All JavaScript inline in `<script>` block
- Data embedded directly in the script (no external file loading) — **except** complex/regenerable deliverables, where data and the `option` config live in a separate `config.json` (canonical) with a generated `config.js` wrapper so regeneration edits only the config file, not the page scaffold
- Responsive: use `window.addEventListener('resize', () => chart.resize())`
- Use [template.html](assets/template.html) as the starting point

For external-config deliverables:
- Keep ONE canonical format (`*.config.json`); generate the `*.config.js` wrapper with `node scripts/sync-config.mjs <file>.config.json` — never hand-sync two copies (they drift)
- Before delivery, run `node scripts/verify-deliverable.mjs <file>.html` for structural checks (vendor presence, loader, config sync, layout)

For multi-chart dashboards:
- Use CSS Grid for layout (e.g., `grid-template-columns: 1fr 1fr`)
- Each chart in its own container `<div>` with unique ID and fixed height
- Initialize separate ECharts instances for each container
- Single resize listener calls `.resize()` on all chart instances
- Add a page title and optional summary section
- **Avoid redundant information**: put shared context (task title, dataset notes, version footnote) in the page-level header/footer once; keep per-chart titles, legends, and footnotes minimal and non-duplicative

For dark theme:
- Use `echarts.init(dom, 'dark')` for built-in dark mode
- Set `body { background: #1a1a2e; }` to match

### Step 5: Save & Verify

1. Save the HTML file to the user's specified path (or suggest a reasonable default like `./output/chart.html`)
2. **Structural verification (program-first, no browser needed)**: run `node scripts/verify-deliverable.mjs <file>.html` — it checks (a) any referenced `vendor/echarts.min.js` actually exists and is non-empty, (b) the ECharts loader does not use `document.write`, (c) `*.config.js`/`*.config.json` pairs are in sync, (d) fixed-layout graph coordinates have no out-of-bounds nodes or overlaps (canvas size from config `meta.canvasWidth/Height`). Fix all reported problems before delivery
3. **Local open verification (渲染证明)**: open the HTML in a browser and confirm the chart actually renders (canvas is not blank, no console errors). For offline-critical deliverables, re-open with network disabled (or `file://` with no CDN access) to prove the vendored copy renders. If a headless browser is available, capture a screenshot, e.g. `chromium --headless --screenshot=out.png --window-size=1440,900 <file>.html`; verify the PNG is not a blank canvas
4. **Static snapshot export**: for offline-critical or review-facing deliverables, export a static PNG/SVG of each chart (toolbox "保存为图片", or `chart.getDataURL({ type: 'png', pixelRatio: 2 })`, or the headless screenshot) and deliver it **alongside the HTML** as render evidence — reviewers without a browser or network can verify the actual visual result offline
5. Provide a brief explanation of:
   - What the visualization shows
   - How to interact with it (tooltip, legend toggle, zoom, toolbox)
   - How to modify the data (where in the code to update values)
   - For arbitration/correction cases: the correction basis (source reference/出处) and the timestamped readings, so the interpretation is reproducible

## Output Requirements

- Output as a **single `.html` file** (self-contained, no external dependencies except the pinned ECharts script; offline-critical output ships a REAL local `vendor/echarts.min.js`, never an empty stub)
- **Offline-critical / review-facing deliverables**: `vendor/echarts.min.js` present and non-empty (verified by `scripts/vendor-echarts.sh` / `scripts/verify-deliverable.mjs`) AND a static PNG/SVG snapshot of each chart delivered alongside the HTML as render evidence
- ECharts version: **pinned 5.x** (e.g. `https://cdn.jsdelivr.net/npm/echarts@5.6.0/dist/echarts.min.js`), never a floating `@5` tag
- Canvas-based rendering by default (use SVG renderer only if requested)
- Responsive: works on both desktop and mobile viewports
- Clean, readable code with comments explaining key sections
- Default language: follow user's preferred language for labels and titles
- Built-in interactivity: tooltip, legend toggle enabled by default
- Toolbox: include save-as-image feature by default

## Reference Documents

### Guides (`references/`)

| Document | Content |
|----------|---------|  
| [echarts-guide.md](references/echarts-guide.md) | ECharts v5 quick reference: option structure, chart types, components, dataset, styling, common chart recipes, plus graph/state-machine/component-view recipes, label-overlap avoidance, and pinned-version/offline fallback patterns |
| [echarts-official-docs.md](references/echarts-official-docs.md) | ECharts official documentation: container sizing, themes, dataset patterns, encode mapping. Load on-demand for deeper understanding |

### Best Practices (`best-practices/`)

**实战沉淀（务必阅读）**：竞技评审与重绘中固化的经验教训，见 [best-practices.md](best-practices/best-practices.md)（最佳实践）与 [pitfalls.md](best-practices/pitfalls.md)（陷阱）——绘制前对照最佳实践，绘制后自查陷阱清单。

### Assets (`assets/`)

| Asset | Purpose |
|-------|---------|  
| [template.html](assets/template.html) | Base HTML template with offline-first ECharts loader (vendored copy + CDN fallback, no `document.write`), responsive setup, and standard initialization pattern |

### Scripts (`scripts/`)

| Script | Purpose |
|--------|---------|
| [vendor-echarts.sh](scripts/vendor-echarts.sh) | Download a pinned `echarts.min.js` into the deliverable's `vendor/` and verify it (non-empty, version marker, JS syntax) — makes offline rendering real, not a stub |
| [sync-config.mjs](scripts/sync-config.mjs) | Generate the `*.config.js` wrapper from the canonical `*.config.json` (or reverse); `--check` mode fails when the pair drifted — no more hand-syncing two copies |
| [check-layout.mjs](scripts/check-layout.mjs) | Coordinate hygiene for fixed-layout graph configs: out-of-bounds nodes/zones and pairwise node overlaps (canvas size from `meta.canvasWidth/Height`) |
| [verify-deliverable.mjs](scripts/verify-deliverable.mjs) | Pre-delivery structural checks per HTML: vendor file presence, no `document.write` loader, config js/json sync, layout checks (program-first; the visual check is Step 5) |

## Quality Checklist

Before delivering the final HTML file, verify:
- [ ] HTML file opens correctly in a browser without errors
- [ ] Browser console shows no JavaScript errors
- [ ] **Local open verification passed**: chart actually renders (canvas non-blank); for offline-critical output, re-opened with network disabled and still renders
- [ ] `scripts/verify-deliverable.mjs` reports no problems (vendor presence, no `document.write`, config sync, layout)
- [ ] **Offline-critical**: `vendor/echarts.min.js` exists and is non-empty (real file, not an empty stub); static PNG/SVG snapshot of each chart delivered alongside the HTML
- [ ] ECharts CDN link is present and correct (pinned version)
- [ ] No external file dependencies (all data is inline) — or config is external with a single canonical format + generated wrapper (no hand-synced js/json pair)
- [ ] Chart container has proper width and height
- [ ] Tooltip displays correctly on hover
- [ ] Legend is present (for multi-series charts) and toggleable
- [ ] Colors are distinguishable and accessible (avoid red/green only)
- [ ] Data values render correctly (spot-check at least 2 data points)
- [ ] Code has comments explaining data format and key options
- [ ] Title and labels match the user's language preference
- [ ] `window.resize` listener is registered for responsive behavior
- [ ] Toolbox with save-as-image is enabled
- [ ] Multi-chart layout (if applicable) is balanced and aligned
- [ ] Multi-chart deliverables share one design-token object (palette/theme/background), not scattered hex colors
- [ ] ECharts version is pinned (no floating `@5` tag); offline-critical output has a real vendored copy
- [ ] All required views from the delivery list are covered, or explicitly waived with a stated reason
- [ ] Dense graph edge labels are not all visible by default (hover-only or `labelLayout` avoidance)
- [ ] State machine `[*]` pseudo-nodes carry a visible label; node colors match the legend categories
- [ ] Inferred/completed edges (model 补全) are visually distinguished from source-described edges (dashed vs solid); manual-intervention edges are explicit (red dashed) and not only in `desc`/tooltip
- [ ] Fixed-layout graph coordinates are recorded (canvas size/grid rules in config `meta`) and pass the overlap/bounds check
- [ ] Corrections/arbitrations of the source description carry a cited basis (出处) in the delivery notes; old vs new readings are labeled with evolution terms ("已迁移"/"演进") rather than a bare "旧" category
- [ ] Deliberately omitted domain parts (undefined states, out-of-scope items) are annotated explicitly
- [ ] Substituted view types (e.g. a relationship view standing in for a deployment diagram) are documented with their tradeoff

## Feedback

**Runtime-mode gate.** If `${SKILL_WORKDIR}/.specify/` does not exist, this skill is
running in standalone mode (a non–Spec Kit deployment, e.g. a global agent skills
directory) — skip this entire Feedback step: no engine call, no feedback entry.

At the end of a substantial run of this skill, perform an agent self-reflection step (never solicit feedback content from the user), following the canonical convention in `.specify/shared/workflow/feedback-step.md`:

1. **Gate on qualification & completion.** Only proceed if this run reached a meaningful wrap-up. Skip trivial/no-op runs; for an aborted run use the abort/partial rule below.
2. **Reflect (no user input).** Review this run against this skill's declared purpose and produce a short review plus ≥1 concrete, skill-specific optimization point. If the run was clean, use exactly: `No significant optimization points identified this run.`
3. **Scope guard.** Keep strictly to this skill's operation; do NOT produce a global/whole-project assessment (that is `/speckit.review`'s job). Entries are `scope: local`.
4. **Dedup guard.** Use a stable `run_id`; if a parent flow already recorded feedback for this same `(unit_id, run_id)`, the engine no-ops.
5. **Persist** via the engine:
   ```bash
   python3 "${SKILL_WORKDIR:-.}/.specify/scripts/python/feedback-utils.py" --action record \
     --unit-id "skill:draw-echarts" --unit-type skill \
     --run-id "<stable-run-id>" --feature "<feature-key-if-any>" \
     --review "<review prose>" --points-file "<points file>"
   ```
6. **Consolidated submission prompt.** If the returned `should_prompt` is `true`, surface a single consolidated prompt inviting the user to submit collected feedback to the Spec Kit developers; on confirmation run `--action mark-submitted`. Below threshold, do not prompt.

**Abort / partial-run rule.** If the run failed before wrap-up, either skip recording or record with `--partial` and a `## Review` beginning `**Partial run** — `.
