---
name: draw-d3js
description: |
  Use D3.js to create interactive data visualizations and output as standalone HTML documents.
  Use when the user mentions "D3", "d3.js", "数据可视化", "data visualization", "交互式图表",
  "interactive chart", "柱状图", "折线图", "散点图", "饼图", "力导向图", "树图", "热力图",
  "bar chart", "line chart", "pie chart", "force graph", "treemap", "heatmap",
  "面积图", "area chart", "气泡图", "bubble chart", "甜甜圈图", "donut chart",
  "数据仪表板", "dashboard", "SVG图表", "svg chart", "数据图形"
skill_id: "<SKILL:.specify/skills/draw-d3js/SKILL.md>"
---

# D3.js Data Visualization Skill

Create interactive data visualizations using D3.js (Data-Driven Documents), output as a self-contained HTML file that can be opened directly in any modern browser.

## Core Principles

### 1. Data-Driven Design
Every visualization must start from data. Choose the chart type that best reveals patterns, trends, or relationships in the user's data. Avoid decorative elements that don't serve the data story.

### 2. Self-Contained Output
Default output is a **single HTML file** with all D3.js code and styles inline, so it works by simply opening in a browser. Two output modes are supported:

- **Single-file mode (default)**: all code, styles, and data inline; D3.js v7 loaded from CDN (`https://d3js.org/d3.v7.min.js`). Best for quick sharing and portability.
- **Split mode (for maintainability / offline reproducibility)**: keep data in a separate local `data.js` or `data.json` next to the HTML, and optionally vendor D3.js locally (download `d3.v7.min.js` beside the HTML and reference it by relative path). Choose split mode when the dataset is large, the chart will be edited repeatedly, or the output must render without CDN access. State the chosen mode in the deliverable explanation.

### 3. D3.js Best Practices
Use D3.js v7 (latest stable). Follow the data-join pattern (`enter/update/exit`), use proper scales and axes, implement responsive SVG with `viewBox`. See [d3js-guide.md](references/d3js-guide.md) for syntax and patterns.

### 4. Progressive Enhancement
Start with a clean, functional visualization. Add interactivity (tooltips, transitions, zoom) only when it serves the user's needs or when explicitly requested.

## Workflow

This skill creates D3.js data visualizations based on user-provided data and requirements. Follow the steps below in order.

### Step 1: Understand Data & Requirements

Analyze the user's input to determine:

1. **Data structure**: What format is the data in? (CSV, JSON, array, table, markdown table, etc.)
2. **Data dimensions**: How many variables? Categorical vs quantitative? Time-series?
3. **Visualization goal**: What story should the chart tell? (comparison, trend, distribution, relationship, composition, hierarchy)
4. **Interactivity needs**: Static or interactive? Tooltips, zoom, filter, animation?
5. **Multi-chart needs**: Does the user need multiple perspectives? If so, plan a dashboard layout.

Data format handling:
- If data is in a markdown table or plain text table, parse it into a JSON array
- If data has Chinese headers, preserve them as labels
- If data volume is large (>100 rows), consider aggregation or sampling before visualization

If critical information is missing, ask **one targeted question**.

### Step 2: Choose Chart Type

Match data characteristics and goals to the appropriate D3.js chart type:

| Goal | Data Type | Recommended Charts |
|------|-----------|--------------------|
| 比较 (Comparison) | Categorical | Bar Chart, Grouped Bar, Lollipop |
| 趋势 (Trend) | Time-series | Line Chart, Area Chart, Multi-line |
| 分布 (Distribution) | Quantitative | Histogram, Box Plot, Violin |
| 关系 (Relationship) | Two+ quantitative | Scatter Plot, Bubble Chart |
| 组成 (Composition) | Part-to-whole | Pie/Donut Chart, Stacked Bar, Treemap |
| 层次 (Hierarchy) | Tree/nested | Tree Layout, Sunburst, Circle Packing |
| 网络 (Network) | Nodes + Links | Force-Directed Graph, Sankey |
| 地理 (Geographic) | Geo-referenced | Choropleth Map, Bubble Map |
| 热度 (Intensity) | Matrix/grid | Heatmap, Calendar Heatmap |
| 时序 (Sequence) | Interactions over time | Sequence Diagram (hand-drawn SVG, no layout library needed) |
| 分层 (Layered model) | Hierarchical resource/domain layers | Tree Layout, Sunburst, Circle Packing |

If multiple perspectives are needed, create multiple visualizations in the same HTML document.

**Multi-chart coverage check**: when the request names a fixed set of diagrams (e.g. "5 类图"), enumerate them up front and map each one to a concrete chart type before writing code, then deliver the complete set. Missing requested diagram types are deliverable defects, not rendering details — a heatmap cannot stand in for a missing sequence diagram or a missing layered-model view.

**Semantic fit**: a heatmap / dependency matrix encodes dependency *intensity* only; it does NOT show topology, deployment layout, or namespace grouping. If the requested diagram is a deployment/architecture view, render a topology or grouped view (fixed-coordinate component layout, zone-grouped force graph, namespace-grouped view) instead of overloading a heatmap with metadata columns (kind/replica/port).

### Step 3: Write D3.js Code

Based on the chosen chart type and data:

1. **Prepare data**: Parse/transform user data into D3-friendly format
2. **Set up SVG**: Define dimensions, margins, and responsive viewBox
3. **Create scales**: Map data domains to visual ranges (x, y, color, size)
4. **Draw axes**: Add labeled axes with proper tick formatting
5. **Binddata & draw elements**: Use the data-join pattern to render visual marks
6. **Add labels & legend**: Title, axis labels, legend for color/size encodings
7. **Add interactivity** (if requested): Tooltips, transitions, hover effects

For D3.js syntax, scale types, layouts, and common patterns, reference [d3js-guide.md](references/d3js-guide.md).

#### Dense Graph & Architecture Diagram Guidance

When the visualization is a component/architecture graph (nodes + links), apply these rules:

- **Layout reproducibility**: a bare `d3.forceSimulation` starts from random positions — every reload yields a different layout. For reproducible output either (a) assign fixed/preset coordinates (`x`/`y` on each node) and pin or skip the simulation, (b) constrain the simulation with per-group `forceX`/`forceY` so each group occupies a stable region, or (c) seed node positions deterministically. State the layout strategy in the deliverable explanation.
- **Dense-graph threshold**: with roughly >15–20 nodes or dense link sets, a free-running force layout tends to cross and overlap. Prefer fixed hand-placed coordinates (readable + reproducible) or a grouped force layout over a pure simulation.
- **Edge labels**: do NOT keep every edge label permanently visible at full opacity in dense graphs — labels overlap nodes and other links. Show labels on hover (per-link label or tooltip), or display labels only for a curated set of key edges. Keep edge label text short (≤ ~8–10 chars) and color edges by type/relation for extra disambiguation.
- **Node semantics**: every node must be a real component of the modeled system. Non-component concepts (queues, budgets, behaviors, states) are NOT nodes — move them into annotation text, a description panel, or a visually distinct non-node annotation layer. Mixing pseudo-nodes into the node set misleads readers about the architecture.
- **Group semantics**: assign groups by semantic role. Infrastructure nodes (storage systems, external services, registries) usually are not part of a subsystem — give them their own group and their own legend color instead of folding them into a subsystem group.
- **Legend**: always pair color swatches (colored dots/rects) with group names in the legend; a text-only legend is insufficient to decode a colored graph.
- **Bilingual node labels**: in architecture graphs, give each node a short English name (e.g. access / gateway) plus a Chinese subtitle (职责说明, ≤ ~12 chars) below it when the audience is mixed-language; English-only short names reduce readability for Chinese reviewers.
- **Edge-label visibility toggle**: default edge labels to hidden (hover reveals), but provide an explicit「常显边标签」toggle (checkbox/button) so the same figure can be printed/exported with labels permanently visible for doc embedding. Hover-only labels with no way to show them statically lose information in static output (screenshots, PDFs).
- **Cross-panel edges**: when the layout uses panels/zones, route cross-panel edges through unified anchor exit/entry ports on panel borders, or use orthogonal (elbow) routing, instead of long straight diagonals that cut through unrelated panels — long cross-panel lines read as clutter in static output.
- **Fixed-coordinate hygiene**: with hand-placed coordinates, record the partition/grid rule in the data file header (e.g. "panel A: x∈[40,360], y∈[40,300]; node slots on a 24px grid") and run a lightweight overlap / panel-bounds validation before render, so later node additions do not silently overlap. See the validation snippet in [d3js-guide.md](references/d3js-guide.md).
- **Inference annotation**: when the diagram includes transitions/states/relations inferred by the author (not stated in the source description) — e.g. a manual recovery path or a direct-delete path — mark them with a distinct style AND a「描述未定义，推断路径」annotation (footnote, legend entry, or edge label); never present inferred semantics as source-described. Similarly, annotate deliberately omitted domain parts instead of omitting silently.

**Data provenance**: when the visualization encodes an external description (architecture doc, spec, requirement), record the source file/version/date in the data file header or a page note (e.g. `// 依据: docs/architecture.md v2.3 (2026-08-01)`). This keeps the figure auditable against its source and surfaces narrative drift between the description and the diagram.

### Step 4: Assemble HTML Document

Package everything into a self-contained HTML file with this structure:

```html
<!DOCTYPE html>
<html lang="zh">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>[Visualization Title]</title>
  <script src="https://d3js.org/d3.v7.min.js"></script>
  <style>
    /* Inline styles for the visualization */
  </style>
</head>
<body>
  <div id="chart"></div>
  <script>
    // D3.js visualization code
  </script>
</body>
</html>
```

Key requirements:
- D3.js loaded from CDN: `https://d3js.org/d3.v7.min.js` (single-file mode) or vendored locally (split mode)
- All CSS inline in `<style>` block
- All JavaScript inline in `<script>` block
- Data embedded inline in single-file mode; in split mode, data lives in a separate local `data.js`/`data.json` referenced by relative path
- A **reproduction appendix** (visible collapsible footer section plus a header comment block): state that the file opens and renders directly with no build step (CDN mode needs network; split/vendored mode works offline), which data files/arrays drive which part of the figure (nodes / links / rels / cells), and the source description version/date the data is based on. Mandatory in split mode — the data is external and maintainers need the entry points.
- **Export capability**: include「导出 SVG / 导出 PNG」buttons (SVG serialization + Canvas conversion) whenever the figure targets embedding into reports or docs. Reusable snippet: [d3js-guide.md](references/d3js-guide.md).
- Responsive design: use `viewBox` or resize listener

Heatmap / dependency-matrix specifics:
- Make the direction convention **visible by default**: annotate rows and columns (e.g. 行 = 依赖方 / 列 = 被依赖方, with an arrow caption), not only in tooltips. Hover tooltips should restate the direction explicitly.
- Rotated column headers (e.g. -32°/-45°) need enough reserved header height — scale it to the longest label. If labels are long, shorten the displayed name and show the full name in the tooltip.
- **Strength scale semantics**: when cells encode an author-assigned strength (e.g. 0–4), define the scale in the data file header AND in a figure caption/legend (e.g. `0=无依赖, 1=弱关联(低频/可选), 2=一般(异步/事件), 3=强(同步调用/数据流), 4=强同步依赖(核心链路)`). Subjective values without a documented scale are not reviewable or reproducible.
- **Wide matrices (>~10–13 columns)**: rotated headers start colliding with narrow cells — group columns by subsystem with group header rows and fold/collapse sections, or paginate into multiple views with a switcher, instead of shrinking cells further.

For multi-chart dashboards:
- Use CSS Grid or Flexbox for layout
- Each chart in its own `<div>` with unique ID
- Share color scales across charts for visual consistency
- Add a page title and optional description section

### Step 5: Save & Verify

1. Save the HTML file to the user's specified path (or suggest a reasonable default like `./output/visualization.html`)
2. Verify the file can be opened in a browser
3. Provide a brief explanation of:
   - What the visualization shows
   - How to interact with it (if interactive)
   - How to modify the data (where in the code to update values)
   - How to reproduce/embed: the file opens directly in a browser and renders with no build step (CDN mode needs network; split/vendored mode works offline); when review embedding matters, optionally attach a headless-browser-exported PNG snapshot of the figure

## Output Requirements

- Output as a **single `.html` file** by default (self-contained; D3.js v7 via CDN). Split mode is allowed when maintainability or offline reproducibility requires it: local `data.js`/`data.json` and/or a vendored local D3.js, all referenced by relative paths
- D3.js version: v7 (via `https://d3js.org/d3.v7.min.js` or vendored locally)
- SVG-based rendering (not Canvas, unless specifically requested for performance)
- Responsive: works on both desktop and mobile viewports
- Clean, readable code with comments explaining key sections
- Default language: follow user's preferred language for labels and titles
- Color palette: use `d3.schemeCategory10` or `d3.schemeTableau10` by default; honor user preferences
- Deliverables encoding an external description annotate the source version/date (data header or page note)
- Inferred diagram semantics (states/transitions/relations not present in the source) are annotated「描述未定义，推断路径」, never presented as source-described
- Figures targeting doc embedding include「导出 SVG/PNG」buttons

## Reference Documents

### Guides (`references/`)

| Document | Content |
|----------|---------|  
| [d3js-guide.md](references/d3js-guide.md) | D3.js v7 quick reference: scales, axes, shapes, layouts, transitions, data-join pattern, and common chart recipes (incl. dense/reproducible force graphs, state machine, hand-drawn sequence diagram, tree/layered model, heatmap direction conventions) |
| [d3js-official-docs.md](references/d3js-official-docs.md) | D3.js official documentation: core concepts, module architecture, data-join philosophy. Load on-demand for deeper understanding |

### Best Practices (`best-practices/`)

**实战沉淀（务必阅读）**：竞技评审与重绘中固化的经验教训，见 [best-practices.md](best-practices/best-practices.md)（最佳实践）与 [pitfalls.md](best-practices/pitfalls.md)（陷阱）——绘制前对照最佳实践，绘制后自查陷阱清单。

### Assets (`assets/`)

| Asset | Purpose |
|-------|---------|  
| [template.html](assets/template.html) | Base HTML template with D3.js CDN, responsive setup, and standard margin convention |

## Quality Checklist

Before delivering the final HTML file, verify:
- [ ] HTML file opens correctly in a browser without errors
- [ ] Browser console shows no JavaScript errors
- [ ] D3.js v7 is present (CDN link or vendored local file) and correct
- [ ] No external dependencies beyond D3.js; split-mode data files are local and relative
- [ ] SVG has proper `viewBox` or responsive sizing
- [ ] Axes have readable labels and proper formatting
- [ ] Colors are distinguishable and accessible (avoid red/green only)
- [ ] Data values render correctly (spot-check at least 2 data points)
- [ ] Code has comments explaining data format and key logic
- [ ] Title and labels match the user's language preference
- [ ] Interactive elements (if any) provide visual feedback
- [ ] Multi-chart layout (if applicable) is balanced and aligned
- [ ] All requested diagram types are delivered (multi-chart coverage check)
- [ ] Graph nodes are all real components — no pseudo-nodes mixed into the node set
- [ ] Graph layout is reproducible (fixed/preset coordinates or grouped/seeded forces)
- [ ] Graph legend pairs color swatches with group names
- [ ] Heatmap/dependency matrix shows its direction convention by default
- [ ] Dense-graph edge labels are not all permanently visible (hover or curated set)
- [ ] Reproduction appendix (data maintenance + render notes) is included in the HTML output
- [ ] Dependency-matrix strength scale is defined (data header + caption) when cells are author-assigned
- [ ] Wide matrices (>~10–13 cols) are grouped/folded or paginated rather than squeezed
- [ ] Dense-graph edge labels have an always-on toggle (not hover-only)
- [ ] Inferred transitions/paths are annotated「描述未定义，推断路径」
- [ ] Data provenance (source description version + date) is recorded for architecture figures
- [ ] Fixed-coordinate graphs record the grid/partition rule and pass an overlap/bounds check
- [ ] Cross-panel edges use anchored or orthogonal routing
- [ ] Export SVG/PNG is available when the figure targets doc embedding

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
     --unit-id "skill:draw-d3js" --unit-type skill \
     --run-id "<stable-run-id>" --feature "<feature-key-if-any>" \
     --review "<review prose>" --points-file "<points file>"
   ```
6. **Consolidated submission prompt.** If the returned `should_prompt` is `true`, surface a single consolidated prompt inviting the user to submit collected feedback to the Spec Kit developers; on confirmation run `--action mark-submitted`. Below threshold, do not prompt.

**Abort / partial-run rule.** If the run failed before wrap-up, either skip recording or record with `--partial` and a `## Review` beginning `**Partial run** — `.
