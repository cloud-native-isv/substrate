---
preset_id: skills-arena
name: 锦标赛竞技场持续竞技团队
pattern: continuous
summary: 多个技能在同一任务上竞技 + 双轮裁判 + 败方重绘的长期运营团队:每个周期产出一份排位与改进清单,驱动技能持续变强。
when_to_use: 需要长期维持一个「打擂」机制——多个同类技能(如多个绘图/生成技能)周期性地同题竞技、双轮评审定排位,并把评审发现回流为技能改进;而不是单次收敛后停止。
signals:
  - 竞技场
  - 竞技
  - 打擂
  - 排位
  - 重绘
  - 多技能同题
  - 同题竞技
  - 裁判
  - 评审团
  - 锦标赛竞技场
  - arena
  - tournament arena
  - skill arena
  - rank skills
  - compare skills
  - shootout
  - judge panel
  - redraw
inputs:
  - name: competing_skills
    required: true
    description: 参赛技能清单(≥2,如 draw-echarts / draw-d3js / draw-mermaid)——每周期全员同题
  - name: task_source
    required: true
    description: 竞技任务来源(题库目录/选题规则);每周期由它抽取或轮转同一道题,保证可比
  - name: quality_dimensions
    required: false
    description: 裁判评分维度与权重(和为 1.0);缺省由 goal 推导后与用户确认
  - name: redraw_rounds
    required: false
    description: 败方重绘轮数,默认 1——末位按裁判意见修订后重出一次,重绘分计入档案但不推翻本轮排位
  - name: cadence
    required: false
    description: 运营节奏(如每周/每两周一个竞技周期)
members:
  - role: team-supervisor
    stage: optimizer
    type: Meta
    lifecycle: persistent
    responsibility: 每周期选题、派发参赛与评审、汇总双轮裁判结果定排位、维护跨周期 STATE.md 排位档案与 run-log;触发技能改进回流
  - role: contestant-dispatcher
    stage: executor
    type: Worker
    lifecycle: temporary
    responsibility: 对每个参赛技能执行同一道题,产物按技能分目录落 run workspace;记录成功/失败,失败视为该技能本周期 correctness 扣分而非弃权
  - role: judge-panel
    stage: evaluator
    type: Worker
    lifecycle: temporary
    responsibility: 双轮裁判——第一轮按 quality_dimensions 独立盲评逐件打分;第二轮交叉对比并处理异议,产出最终排序与逐件意见(供重绘与回流)
  - role: skill-improver
    stage: optimizer
    type: Meta
    lifecycle: temporary
    responsibility: 把裁判意见转写为败方技能的具体改进建议(对接 improve-skills 流程);重绘即按该建议修订后重出;不直接改技能源文件,改进落真实路径须人批准
config:
  summary:
    enabled: true
    every: 5
    interactive: false
  maturity: L1
  cadence: weekly
  redraw_rounds: 1
provenance: 自 2026-08-07 viz-skill-arena 真实运行蒸馏(反馈批次 20260807T051529Z:创建时 preset 匹配为 low、从零推导成本高,故沉淀为 continuous 竞技形状)
---

## Goal Skeleton

长期运营一个技能竞技场:`<competing_skills>` 每个周期(`<cadence>`)在来自 `<task_source>` 的同一道题上同题竞技,
经双轮裁判按 `<quality_dimensions>` 定排位;末位按裁判意见重绘 `<redraw_rounds>` 轮,
裁判发现回流为技能改进建议。排位档案跨周期累积,机制本身不因单次结果停止。

## Static Structure

| Role | Stage | Type | Lifecycle | Responsibility |
|------|-------|------|-----------|----------------|
| team-supervisor | optimizer | Meta | persistent | 选题、派发、排位裁决、状态脊柱、改进回流触发 |
| contestant-dispatcher | executor | Worker | temporary | 每技能同题执行,产物分目录,失败显式记录 |
| judge-panel | evaluator | Worker | temporary | 双轮裁判:独立盲评 → 交叉对比定排序 |
| skill-improver | optimizer | Meta | temporary | 裁判意见 → 败方技能改进建议 + 重绘修订 |

## Dynamic Structure

每一个周期(cycle):

```
READ        supervisor 读 constraints.md + budget + kill-switch;从 STATE.md 取上期排位与待办改进
SELECT      从 task_source 抽题(轮转或固定打擂题);声明本周期竞技范围
EXECUTE     contestant-dispatcher 逐技能执行同一道题 → 产物按技能分目录落 run workspace
JUDGE       judge-panel 第一轮独立盲评逐件打分;第二轮交叉对比、处理异议 → 最终排序 + 逐件意见
REDRAW      skill-improver 把末位意见转写为修订;重绘一轮,重绘分入档案(不推翻本轮排位)
REPORT      本周期排位、得分、重绘对比、改进建议清单写入 runs/<ts>-report.md;STATE.md 更新排位档案
SUMMARIZE   按 summary 配置触发 goal 总结(每 N 周期)
```

## Instantiation

1. 确认参赛技能 ≥2 且同类可比(不同类技能同题竞技没有裁判基准)。
2. 与用户确认 `quality_dimensions` 及权重(和为 1.0)——裁判维度不清的竞技场只会产出噪声排位。
3. 确认 `task_source` 可稳定供题;题库枯竭即机制空转,需在 constraints.md 写入补题责任。
4. 落 `.specify/teams/<slug>/team.md`,frontmatter 加 `preset: skills-arena`;`config` 填 cadence/redraw_rounds;连续团队三件套(constraints.md / STATE.md / run-log.jsonl)初始化。
5. 竞技产物与评分转储一律写 `.specify/teams/.work/<slug>/`;只有排位档案与报告落 tracked 团队目录。

## Constraints & Hard Rules

- **同题是竞技前提**:每周期所有参赛技能必须执行同一道题;题目不一致的分数不可比,排位无效。
- **双轮裁判不可并省**:只有独立盲评会锚定先入之见;只有交叉对比会漏掉维度间矛盾。两轮都跑。
- **重绘不翻案**:重绘分只入档案用于趋势观察;本周期排位以正赛为准,否则激励变成"先摆烂后重绘"。
- **改进回流须经人批准**:skill-improver 只产出建议与重绘修订;技能源文件的真实改动走 improve-skills 的正常批准路径,竞技场不直接写技能。
- **失败显式记录**:参赛技能执行失败计 correctness 扣分并公示,不得静默弃权或剔除。

## Known Pitfalls

- 裁判维度权重和不为 1.0 → 跨周期排位不可比。
- 题库枯竭后继续"打擂"→ 周期空转、STATE 漂移;补题责任必须写进 constraints。
- 把重绘当成第二正赛 → 排位被翻案噪声淹没(见硬规则第三条)。
- 参赛技能清单长期不变 → 竞技场退化为自嗨;周期性评估是否增补新技能。
