# 05 - 数据源专用模板

> **用途**：针对不同类型的眼科数据源（临床指南、教材、文献、药品说明书、电子病历）设计差异化的 Prompt 模板，每种数据源的语言特征和知识密度不同，需要定制化处理策略。

---

## 模板 5.1：临床指南抽取模板

````## 任务

你正在处理一份**眼科临床指南/专家共识**文本。指南文本具有以下特征：

- 权威性高：包含循证医学推荐（A级/B级/C级）
- 结构化强：通常包含定义、诊断标准、分期分级、治疗推荐
- 语言规范：使用标准医学术语

## 抽取重点

1. **诊断标准**：疾病的诊断条件和检查要求 → DIAGNOSED_BY 关系
2. **分期分级**：疾病的严重程度分期 → HAS_STAGE / PROGRESSES_TO 关系
3. **治疗推荐**：各分期的推荐治疗方案（含循证等级）→ TREATED_BY 关系
4. **循证等级**：每条治疗推荐标注推荐等级 → attributes.evidence_level

## 特别注意

- 指南中的推荐语句（如"推荐使用XXX"、"建议XXX"）表示高质量治疗关系
- "可选择"、"可考虑"表示替代方案，treatment_line 标注为"替代方案"
- 如果指南明确标注了推荐等级（I级/II级/III级，A级/B级/C级），记录在 attributes.evidence_level 中

## 指南文本

{text}

## 输出格式

```json
{
  "guideline_info": {
    "name": "指南名称（如能从文本推断）",
    "organization": "发布机构（如能从文本推断）",
    "year": "发布年份（如能从文本推断）"
  },
  "triplets": [
    {
      "head": {"name": "实体名", "type": "实体类型"},
      "relation": "关系类型",
      "tail": {"name": "实体名", "type": "实体类型"},
      "evidence": "原文证据",
      "confidence": "high|medium|low",
      "attributes": {
        "evidence_level": "I级/II级/III级 或 A级/B级/C级",
        "recommendation_strength": "强推荐/弱推荐/专家共识",
        "applicable_stage": "适用的疾病分期"
      }
    }
  ]
}
```
````

---

## 模板 5.2：医学教材抽取模板

````## 任务

你正在处理一份**眼科医学教材**的文本段落。教材文本具有以下特征：

- 系统性强：按疾病系统分类，知识体系完整
- 基础与临床结合：包含解剖、病理生理、临床表现、诊断、治疗
- 教学导向：包含典型的知识要点

## 抽取重点

1. **疾病定义和病理**：疾病的基本定义和病理机制 → CAUSED_BY 关系 + 疾病属性
2. **解剖关系**：涉及的解剖结构和层次关系 → LOCATED_IN 关系
3. **临床表现**：症状和体征的完整列表 → HAS_SYMPTOM 关系
4. **诊断流程**：从症状到检查到诊断的逻辑链 → DIAGNOSED_BY + INDICATES 关系
5. **鉴别诊断**：需要与哪些疾病鉴别 → 关系类型使用 PROGRESSES_TO 或在 attributes 中标注

## 教材文本

{text}

## 输出格式

```json
{
  "text_type": "textbook",
  "topic": "本段落的主要主题（一句话概括）",
  "triplets": [
    {
      "head": {"name": "实体名", "type": "实体类型"},
      "relation": "关系类型",
      "tail": {"name": "实体名", "type": "实体类型"},
      "evidence": "原文证据",
      "confidence": "high|medium|low",
      "attributes": {}
    }
  ],
  "knowledge_points": [
    "知识点1（原文提炼的核心教学要点）",
    "知识点2"
  ]
}
```
````

---

## 模板 5.3：PubMed 文献抽取模板

````## 任务

你正在处理一篇**眼科科研文献的摘要**。文献摘要通常包含 Background、Methods、Results、Conclusion 四个部分。

## 抽取策略

| 文献部分 | 抽取策略 | 应关注 |
|---------|---------|--------|
| Background | 抽取疾病背景知识 | 流行病学、疾病定义 |
| Methods | **跳过** | 不抽取研究方法、样本量 |
| Results | 抽取研究发现 | 治疗效果、发病率、检查灵敏度等数据 |
| Conclusion | 抽取核心结论 | 治疗推荐、新发现、临床意义 |

## 特别规则

1. **效果数据**：如果 Results 中提到具体数据（如"有效率85%"、"眼压降低30%"），记录在 attributes 中。
2. **P值和统计显著性**：不提取 P 值本身，但 P<0.05 的发现 confidence 标记为 high。
3. **新发现 vs 已知**：文献可能报道新发现（如新的治疗靶点），如实抽取，不因与常识矛盾而跳过。
4. **研究类型**：识别研究类型（RCT、队列研究、病例报告等），记录在 metadata.research_type 中。
5. **样本限制**：注意研究的人群和条件限制，记录在 attributes.study_population 中。

## 文献摘要

{text}

## 输出格式

```json
{
  "metadata": {
    "research_type": "RCT | 队列研究 | 病例对照 | 横断面研究 | 病例报告 | 综述",
    "study_population": "研究人群描述",
    "source_type": "literature"
  },
  "triplets": [
    {
      "head": {"name": "实体名", "type": "实体类型"},
      "relation": "关系类型",
      "tail": {"name": "实体名", "type": "实体类型"},
      "evidence": "原文证据",
      "confidence": "high|medium|low",
      "attributes": {
        "efficacy": "效果数据（如有）",
        "study_population": "适用人群",
        "evidence_level": "基于研究类型自动评估"
      }
    }
  ]
}
```
````

---

## 模板 5.4：药品说明书抽取模板

````## 任务

你正在处理一份**眼科药品说明书**。说明书通常包含以下章节：

【药品名称】【成分】【适应症】【用法用量】【不良反应】【禁忌】【注意事项】【孕妇及哺乳期妇女用药】【儿童用药】【老年用药】【药物相互作用】【药理毒理】【贮藏】

## 抽取策略

对每个章节的抽取策略如下：

| 章节 | 抽取内容 | 输出位置 |
|------|---------|---------|
| 【药品名称】 | 药物名称和别名 | drug_entity |
| 【成分】 | 有效成分、浓度 | drug_attributes |
| 【适应症】 | 适应症疾病列表 → 反向 TREATED_BY | triplets |
| 【用法用量】 | 用法、用量、疗程 | drug_attributes |
| 【不良反应】 | 不良反应 → Drug HAS_SYMPTOM/COMPLICATED_BY Disease | triplets |
| 【禁忌】 | 禁忌症 → CONTRAINDICATED | triplets |
| 【注意事项】 | 重要使用提示 | drug_attributes.precautions |
| 【药物相互作用】 | 与其他药物的相互作用 | triplets 或 attributes |
| 【药理毒理】 | 作用机制 | drug_attributes.mechanism |

## 特别注意

1. **频率标注**：不良反应通常标注频率（常见/偶见/罕见），记录在 attributes 中。
2. **慎用 vs 禁用**："禁用"为绝对禁忌，"慎用"为相对禁忌。
3. **人群限制**：孕妇、儿童、老年用药限制记录在 attributes.population_limitations 中。

## 说明书文本

{text}

## 输出格式

```json
{
  "drug_entity": {
    "name": "药品通用名",
    "type": "Drug",
    "aliases": ["商品名"]
  },
  "drug_attributes": {
    "ingredients": [],
    "concentration": "",
    "drug_class": "",
    "dosage_form": "",
    "usage": "",
    "mechanism": "",
    "storage": "",
    "precautions": [],
    "population_limitations": {}
  },
  "triplets": [
    {
      "head": {"name": "实体名", "type": "实体类型"},
      "relation": "关系类型",
      "tail": {"name": "实体名", "type": "实体类型"},
      "evidence": "原文证据",
      "confidence": "high|medium|low",
      "attributes": {}
    }
  ]
}
```
````

---

## 模板 5.5：电子病历抽取模板

````## 任务

你正在处理一份**眼科电子病历**的脱敏文本。病历文本具有以下特征：

- 信息密度高：包含主诉、现病史、既往史、检查、诊断、治疗
- 书写口语化：可能使用非标准术语（如"左眼看不清"）
- 时间性强：包含时间线信息

## 抽取重点

1. **主诉**：患者的核心症状 → Symptom 实体
2. **诊断信息**：明确的临床诊断 → Disease 实体
3. **检查结果**：检查项目和参数 → Examination 实体（不提取数值）
4. **治疗方案**：处方和治疗方案 → TREATED_BY / Drug 实体
5. **既往史和全身疾病**：相关全身疾病 → RiskFactor 实体

## 特别注意

1. **脱敏**：文本已脱敏，但仍不应提取任何可能的个人信息标识。
2. **否定处理**：病历中常见否定描述（如"未见视网膜脱离"），negated 设为 true。
3. **口语化术语**：如"左眼看不清"标准化为"左眼视力下降"，"眼压高"标准化为"眼压升高"。
4. **不提取的内容**：具体的数值（如"眼压16mmHg"只提取"眼压测量"，不提取"16mmHg"）、时间信息、住院号等。
5. **病历中的不确定性**：如"疑似"、"不除外"等表述，confidence 标记为 medium。

## 病历文本

{text}

## 输出格式

```json
{
  "text_type": "ehr",
  "clinical_profile": {
    "chief_complaint": "主诉（一句话总结）",
    "diagnosis": ["主要诊断列表"],
    "laterality": "左眼/右眼/双眼（如涉及）"
  },
  "entities": [
    {
      "name": "标准名称",
      "type": "实体类型",
      "original_text": "原文表述",
      "negated": false,
      "clinical_context": "主诉/现病史/既往史/检查/诊断/治疗"
    }
  ],
  "triplets": [
    {
      "head": {"name": "实体名", "type": "实体类型"},
      "relation": "关系类型",
      "tail": {"name": "实体名", "type": "实体类型"},
      "evidence": "原文证据",
      "confidence": "high|medium|low"
    }
  ]
}
```
````

---

## 数据源 → 模板映射速查表

| 数据源 | 推荐模板 | 抽取策略 | 输出结构 |
|--------|---------|---------|---------|
| 临床指南/共识 | 模板 5.1 | 循证关系为主，标注推荐等级 | `guideline_info` + `triplets` |
| 医学教材 | 模板 5.2 | 系统性知识，知识点提炼 | `topic` + `triplets` + `knowledge_points` |
| PubMed 文献 | 模板 5.3 | 研究发现，效果数据 | `metadata.research_type` + `triplets` |
| 药品说明书 | 模板 5.4 | 药物属性+关系全覆盖 | `drug_entity` + `drug_attributes` + `triplets` |
| 电子病历 | 模板 5.5 | 临床实体+否定处理 | `clinical_profile` + `entities` + `triplets` |
| 科普文章 | 通用联合抽取（模板4.1） | 症状-疾病-预防 | `triplets`（confidence 偏 low） |
