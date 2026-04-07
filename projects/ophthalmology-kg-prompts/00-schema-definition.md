# 00 - 共享本体 Schema 定义

> **用途**：本文档定义了眼科视光知识图谱的完整本体 Schema，包括实体类型、关系类型、属性体系和术语约束规则。所有 Prompt 模板均引用本 Schema 作为输出约束的基础。

---

## 一、系统提示词（System Prompt）

以下 System Prompt 应作为所有抽取模板的**共享前缀**，通过 `system` 角色注入：

```
你是一名眼科视光领域的医学知识抽取专家。你的任务是从给定的眼科医学文本中，按照预定义的本体 Schema 抽取结构化的知识三元组（实体-关系-实体）和属性信息。

## 核心原则

1. **忠实原文**：只抽取文本中明确提及的内容，严禁推断或补充原文未表达的知识。
2. **标准术语**：优先使用标准医学术语（对齐 ICD-10 / SNOMED CT），保留原文中的专有名词。
3. **粒度适中**：抽取粒度以"一个完整知识点"为单位，避免过度拆分或过度合并。
4. **不确定标记**：对模糊或不确定的抽取结果，将 confidence 标记为 "low"，而非猜测。
5. **无内容不编造**：如果文本中没有与目标抽取任务相关的内容，返回空数组，不要编造三元组。
6. **证据溯源**：每个三元组必须附带 evidence 字段，引用原文中支持该三元组的片段。

## 输出格式

严格按 JSON 格式输出，不要添加任何额外解释或 markdown 标记。确保输出是合法的 JSON。
```

---

## 二、实体类型定义

### 2.1 实体类型枚举

| 实体 ID | 中文名称 | 英文标识 | 定义 | 示例 |
|---------|---------|---------|------|------|
| `Anatomy` | 解剖结构 | `Anatomy` | 眼球及其附件的解剖学结构 | 角膜、视网膜、晶状体、视神经、泪腺、睫状体、虹膜、巩膜、脉络膜、黄斑区、视盘、眼睑、结膜、泪道 |
| `Disease` | 疾病/异常 | `Disease` | 眼科疾病、视功能异常 | 开角型青光眼、干眼症、糖尿病视网膜病变、近视、白内障、黄斑变性、斜视、弱视、葡萄膜炎 |
| `Symptom` | 症状/体征 | `Symptom` | 患者主观症状或临床客观体征 | 视力下降、眼痛、眼红、飞蚊症、畏光、流泪、眼干、眼痒、视野缺损、虹视、复视、眼球震颤 |
| `Examination` | 检查方法 | `Examination` | 用于诊断和评估的眼科检查手段 | OCT、视野检查、裂隙灯显微镜检查、眼压测量、角膜地形图、验光、荧光素眼底血管造影(FFA)、B超、角膜内皮计数、对比敏感度检查 |
| `Stage` | 诊断/分期 | `Stage` | 疾病的分期、分级、分型 | 轻度干眼、增殖期糖尿病视网膜病变、早期年龄相关性白内障、高度近视（>600度） |
| `Treatment` | 治疗干预 | `Treatment` | 药物治疗、手术治疗、光学矫正、康复训练等 | LASIK、ICL植入术、白内障超声乳化术、角膜塑形镜验配、视觉训练、玻璃体切割术、抗VEGF治疗、激光光凝 |
| `Drug` | 药物/器械 | `Drug` | 眼科用药和医疗器械产品 | 0.01%阿托品滴眼液、左氧氟沙星滴眼液、拉坦前列素滴眼液、人工泪液、多焦点IOL、离焦镜片 |
| `RiskFactor` | 风险因素 | `RiskFactor` | 增加眼病发生风险的因素 | 糖尿病、高血压、高度近视家族史、长时间近距离用眼、紫外线暴露、年龄>60岁、吸烟、全身免疫性疾病 |

### 2.2 实体类型约束（JSON Schema）

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "OphthalmologyEntity",
  "type": "object",
  "required": ["name", "type"],
  "properties": {
    "name": {
      "type": "string",
      "description": "实体标准名称"
    },
    "type": {
      "type": "string",
      "enum": ["Anatomy", "Disease", "Symptom", "Examination", "Stage", "Treatment", "Drug", "RiskFactor"],
      "description": "实体类型，必须为上述8种之一"
    },
    "aliases": {
      "type": "array",
      "items": {"type": "string"},
      "description": "别名列表（可选）"
    },
    "icd_code": {
      "type": "string",
      "description": "ICD-10 编码（仅 Disease 类型）"
    }
  }
}
```

---

## 三、关系类型定义

### 3.1 关系类型枚举

| 关系 ID | 中文名称 | 定义 | Domain（头实体类型） | Range（尾实体类型） | 示例 |
|---------|---------|------|-------------------|-------------------|------|
| `HAS_SYMPTOM` | 表现为 | 疾病的临床症状或体征 | `Disease` | `Symptom` | 开角型青光眼 → 视野缺损 |
| `LOCATED_IN` | 位于 | 解剖结构的层次/位置关系 | `Anatomy` | `Anatomy` | 视网膜 → 眼球后段 |
| `DIAGNOSED_BY` | 确诊需 | 疾病的诊断检查方法 | `Disease` | `Examination` | 糖尿病视网膜病变 → OCT |
| `TREATED_BY` | 治疗方式 | 疾病的治疗手段 | `Disease` | `Treatment`, `Drug` | 近视 → 角膜塑形镜验配 |
| `CAUSED_BY` | 病因为 | 疾病的致病因素 | `Disease` | `Disease`, `RiskFactor` | 糖尿病视网膜病变 → 糖尿病 |
| `INDICATES` | 检查指征 | 症状/体征提示需要做的检查 | `Symptom` | `Examination` | 眼压升高 → 青光眼排查 |
| `CONTRAINDICATED` | 禁忌 | 某条件下不能使用某治疗 | `RiskFactor`, `Disease` | `Treatment` | 角膜偏薄 → LASIK |
| `PROGRESSES_TO` | 进展为 | 疾病的自然进展路径 | `Disease`, `Stage` | `Disease`, `Stage` | 轻度干眼 → 中度干眼 |
| `HAS_STAGE` | 分期为 | 疾病的分期/分级体系 | `Disease` | `Stage` | 糖尿病视网膜病变 → 增殖期 |
| `SUBCLASS_OF` | 子类属于 | 概念的分类层级关系 | `Disease` | `Disease` | 开角型青光眼 → 青光眼 |
| `HAS_RISK_FACTOR` | 风险因素 | 疾病的危险因素 | `Disease` | `RiskFactor` | 近视 → 长时间近距离用眼 |
| `COMPLICATED_BY` | 并发症 | 疾病的并发症 | `Disease` | `Disease` | 糖尿病视网膜病变 → 牵拉性视网膜脱离 |

### 3.2 关系类型约束

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "OphthalmologyRelation",
  "type": "string",
  "enum": [
    "HAS_SYMPTOM",
    "LOCATED_IN",
    "DIAGNOSED_BY",
    "TREATED_BY",
    "CAUSED_BY",
    "INDICATES",
    "CONTRAINDICATED",
    "PROGRESSES_TO",
    "HAS_STAGE",
    "SUBCLASS_OF",
    "HAS_RISK_FACTOR",
    "COMPLICATED_BY"
  ]
}
```

---

## 四、属性体系定义

### 4.1 疾病属性

| 属性名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| `icd_code` | string | ICD-10 编码 | "H40.1" |
| `snomed_ct_id` | string | SNOMED CT 概念ID | "256印象89002" |
| `prevalence` | string | 发病率/患病率 | "我国40岁以上人群约2.3%" |
| `onset_age` | string | 好发年龄 | "多发于40岁以上" |
| `severity_levels` | array | 严重程度分级 | ["轻度", "中度", "重度"] |
| `chronicity` | string | 急慢性分类 | "慢性进展性" |
| `inheritance` | string | 遗传性 | "多基因遗传" |
| `prognosis` | string | 预后 | "早期治疗可控制" |

### 4.2 药物属性

| 属性名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| `ingredients` | array | 成分 | ["阿托品"] |
| `dosage_form` | string | 剂型 | "滴眼液" |
| `concentration` | string | 浓度/规格 | "0.01%" |
| `usage` | string | 用法用量 | "每日1次，每晚1滴" |
| `indications` | array | 适应症 | ["近视进展控制"] |
| `contraindications` | array | 禁忌症 | ["青光眼", "过敏体质"] |
| `adverse_reactions` | array | 不良反应 | ["畏光", "近视力模糊"] |
| `insurance_category` | string | 医保类别 | "医保甲类" |

### 4.3 检查方法属性

| 属性名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| `purpose` | string | 检查目的 | "评估视网膜神经纤维层厚度" |
| `sensitivity` | string | 灵敏度 | "95%" |
| `specificity` | string | 特异性 | "90%" |
| `normal_range` | string | 正常值范围 | "RNFL厚度 ≥ 80μm" |
| `equipment` | string | 所需设备 | "频域OCT" |
| `duration` | string | 检查时长 | "约10分钟" |

---

## 五、领域术语词典

### 5.1 标准术语对照表（部分）

以下词典用于实体标准化，抽取时应优先使用"标准名称"列：

| 原文中可能出现的形式 | 标准名称 | 实体类型 |
|---------------------|---------|---------|
| 老花、老花眼、老视 | 老视（年龄相关性调节力下降） | `Disease` |
| 近视眼、近视 | 近视 | `Disease` |
| 飞蚊症、玻璃体混浊 | 玻璃体混浊（飞蚊症） | `Disease` |
| 白内障、老年性白内障、年龄相关性白内障 | 年龄相关性白内障 | `Disease` |
| 青光眼、原发性开角型青光眼、POAG | 原发性开角型青光眼 | `Disease` |
| 糖网、DR、糖尿病视网膜病变 | 糖尿病视网膜病变 | `Disease` |
| AMD、老年黄斑变性、ARMD | 年龄相关性黄斑变性 | `Disease` |
| 干眼、干眼症、干眼综合征 | 干眼症 | `Disease` |
| OCT、光学相干断层扫描 | 光学相干断层扫描（OCT） | `Examination` |
| FFA、荧光造影、FFA | 荧光素眼底血管造影（FFA） | `Examination` |
| 验光、屈光检查 | 验光（屈光检查） | `Examination` |
| 激光手术、飞秒、LASIK、SMILE | 根据具体术式标准化为对应 `Treatment` | `Treatment` |
| 角膜塑形镜、OK镜 | 角膜塑形镜（OK镜） | `Treatment` |
| IOL、人工晶体、人工晶状体 | 人工晶状体（IOL） | `Drug` |
| 眼压、IOP | 眼压（IOP） | `Symptom` |

### 5.2 ICD-10 眼科编码前缀

抽取疾病实体时，如文本中包含 ICD 编码，应提取并存储：

| 编码范围 | 疾病类别 |
|---------|---------|
| H00-H06 | 眼睑、泪器、眼眶疾病 |
| H10-H13 | 结膜疾病 |
| H15-H22 | 巩膜、角膜、虹膜睫状体、晶状体疾病 |
| H25-H28 | 白内障及其他晶状体疾病 |
| H30-H36 | 脉络膜、视网膜疾病 |
| H40-H42 | 青光眼 |
| H43-H45 | 玻璃体、眼球疾病 |
| H46-H48 | 视神经及视路疾病 |
| H49-H52 | 眼外肌、双眼视觉、屈光疾病 |
| H53-H54 | 视觉障碍及盲 |

---

## 六、三元组输出 Schema

所有抽取模板的输出均应遵循以下统一格式：

```json
{
  "triplets": [
    {
      "head": {
        "name": "实体名称",
        "type": "实体类型"
      },
      "relation": "关系类型",
      "tail": {
        "name": "实体名称",
        "type": "实体类型"
      },
      "attributes": {},
      "evidence": "支持该三元组的原文片段",
      "confidence": "high | medium | low"
    }
  ],
  "metadata": {
    "source_type": "guideline | literature | drug_label | textbook | ehr | popular_science",
    "source_id": "数据源标识（如文献DOI、指南名称）",
    "model": "模型名称",
    "timestamp": "ISO8601时间戳"
  }
}
```

### 字段说明

| 字段 | 必填 | 说明 |
|------|------|------|
| `triplets[].head.name` | ✅ | 头实体标准名称 |
| `triplets[].head.type` | ✅ | 头实体类型（8选1） |
| `triplets[].relation` | ✅ | 关系类型（12选1） |
| `triplets[].tail.name` | ✅ | 尾实体标准名称 |
| `triplets[].tail.type` | ✅ | 尾实体类型（8选1） |
| `triplets[].attributes` | ❌ | 实体或关系的附加属性 |
| `triplets[].evidence` | ✅ | 原文证据片段 |
| `triplets[].confidence` | ✅ | 置信度：high/medium/low |

### 置信度判定标准

| 级别 | 判定条件 |
|------|---------|
| `high` | 文本明确、直接陈述了该三元组关系 |
| `medium` | 文本间接暗示了该关系，需要一定的医学推理 |
| `low` | 关系存在较大模糊性或需要跨段落推断 |

---

## 七、负样本规则（不应抽取的内容）

以下类型的内容**不应**被抽取为实体或三元组：

1. **非医学实体**：医院名称、医生姓名、患者个人信息、科室名称
2. **时间信息**：具体日期、住院天数（除非与疾病分期直接相关）
3. **数量信息**：样本量、研究经费等
4. **方法论描述**：研究方法、统计学方法（如"采用t检验"）
5. **通用描述**："常见病"、"多发病"等无具体指代的泛称
6. **非眼科内容**：与眼科视光无关的全身性疾病描述（除非作为风险因素）
7. **否定句中的实体**：文本明确否定的关系不应抽取（如"不伴有眼痛"则不抽取"疾病→眼痛"）
