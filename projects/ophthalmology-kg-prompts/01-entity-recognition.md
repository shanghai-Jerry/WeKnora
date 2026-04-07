# 01 - 实体识别模板集

> **用途**：从眼科医学文本中识别并分类命名实体。提供 3 种 Prompt 策略模板，可独立使用，也可组合进行多 Prompt 集成投票。

---

## 模板 1.1：指令式 NER 模板

**适用场景**：通用场景，结构化文本（指南、教材段落），需要快速抽取时。

````
## 任务定义

从给定的眼科医学文本中识别所有医学实体，并按预定义的 8 种实体类型进行分类。

## 实体类型

| 类型 ID | 类型名称 | 定义 | 示例 |
|---------|---------|------|------|
| Anatomy | 解剖结构 | 眼球及附件的解剖结构 | 角膜、视网膜、晶状体、视神经、黄斑区 |
| Disease | 疾病/异常 | 眼科疾病、视功能异常 | 开角型青光眼、干眼症、近视、白内障 |
| Symptom | 症状/体征 | 患者主观症状或客观体征 | 视力下降、眼痛、飞蚊症、畏光、视野缺损 |
| Examination | 检查方法 | 眼科诊断和评估检查 | OCT、视野检查、裂隙灯检查、验光 |
| Stage | 诊断/分期 | 疾病的分期分级分型 | 增殖期糖尿病视网膜病变、轻度干眼 |
| Treatment | 治疗干预 | 手术、药物、光学矫正、康复 | LASIK、角膜塑形镜、抗VEGF治疗 |
| Drug | 药物/器械 | 眼科用药和器械产品 | 阿托品滴眼液、人工泪液、多焦点IOL |
| RiskFactor | 风险因素 | 增加眼病风险的因素 | 糖尿病、高度近视家族史、长时间近距离用眼 |

## 抽取规则

1. 只识别文本中明确出现的实体，不要推断。
2. 一个实体只能归属一种类型，选择最精确的类型。
3. 使用标准医学术语作为实体名称。
4. 忽略否定句中的实体（如"不伴有眼痛"中不提取"眼痛"作为当前疾病症状）。
5. 忽略非医学实体（医院名、医生姓名、时间日期等）。

## 输入文本

{text}

## 输出格式

严格按以下 JSON 格式输出：

```json
{
  "entities": [
    {
      "name": "实体标准名称",
      "type": "实体类型ID",
      "span": "原文中的完整片段",
      "aliases": ["别名1"]
    }
  ]
}
```

如果文本中无相关实体，返回 {"entities": []}。
````

---

## 模板 1.2：Few-shot NER 模板

**适用场景**：复杂文本、新型文献，需要通过示例引导模型理解领域术语边界时。

````
## 任务

从眼科医学文本中识别命名实体并分类。请参考以下示例学习抽取模式，然后处理输入文本。

## 实体类型（8种）

Anatomy（解剖结构）、Disease（疾病）、Symptom（症状/体征）、Examination（检查方法）、Stage（分期分级）、Treatment（治疗干预）、Drug（药物/器械）、RiskFactor（风险因素）

## 示例

### 示例 1
**输入**：患者，男，68岁，双眼视力逐渐下降3年，加重半年。既往有2型糖尿病病史15年。眼科检查：双眼晶状体混浊，核性II级。眼压右眼16mmHg，左眼18mmHg。OCT示双眼黄斑区视网膜厚度正常。
**输出**：
```json
{
  "entities": [
    {"name": "视力下降", "type": "Symptom", "span": "视力逐渐下降"},
    {"name": "2型糖尿病", "type": "RiskFactor", "span": "2型糖尿病"},
    {"name": "晶状体", "type": "Anatomy", "span": "晶状体"},
    {"name": "年龄相关性白内障", "type": "Disease", "span": "晶状体混浊，核性II级"},
    {"name": "眼压", "type": "Examination", "span": "眼压"},
    {"name": "光学相干断层扫描（OCT）", "type": "Examination", "span": "OCT"},
    {"name": "黄斑区", "type": "Anatomy", "span": "黄斑区"},
    {"name": "视网膜", "type": "Anatomy", "span": "视网膜"},
    {"name": "核性白内障II级", "type": "Stage", "span": "核性II级"}
  ]
}
```

### 示例 2
**输入**：原发性开角型青光眼（POAG）是最常见的青光眼类型。早期通常无症状，眼压逐渐升高。诊断依赖于视野检查和视神经纤维层分析。一线治疗包括前列腺素类降眼压药物（如拉坦前列素滴眼液），目标眼压需个体化设定。
**输出**：
```json
{
  "entities": [
    {"name": "原发性开角型青光眼", "type": "Disease", "span": "原发性开角型青光眼（POAG）", "aliases": ["POAG"]},
    {"name": "青光眼", "type": "Disease", "span": "青光眼类型"},
    {"name": "眼压升高", "type": "Symptom", "span": "眼压逐渐升高"},
    {"name": "视野检查", "type": "Examination", "span": "视野检查"},
    {"name": "视神经纤维层分析", "type": "Examination", "span": "视神经纤维层分析"},
    {"name": "拉坦前列素滴眼液", "type": "Drug", "span": "拉坦前列素滴眼液"},
    {"name": "眼压", "type": "Examination", "span": "眼压"},
    {"name": "降眼压", "type": "Treatment", "span": "前列腺素类降眼压药物"}
  ]
}
```

### 示例 3
**输入**：角膜塑形镜（OK镜）是一种特殊设计的硬性透气性接触镜，通过夜间佩戴暂时改变角膜形态，用于控制青少年近视进展。禁忌证包括角膜地形图异常、干眼症、活动性眼部感染等。
**输出**：
```json
{
  "entities": [
    {"name": "角膜塑形镜（OK镜）", "type": "Treatment", "span": "角膜塑形镜（OK镜）", "aliases": ["OK镜"]},
    {"name": "硬性透气性接触镜", "type": "Drug", "span": "硬性透气性接触镜"},
    {"name": "角膜", "type": "Anatomy", "span": "角膜"},
    {"name": "近视", "type": "Disease", "span": "近视"},
    {"name": "青少年", "type": "RiskFactor", "span": "青少年"},
    {"name": "角膜地形图", "type": "Examination", "span": "角膜地形图"},
    {"name": "干眼症", "type": "Disease", "span": "干眼症"},
    {"name": "眼部感染", "type": "Disease", "span": "活动性眼部感染"}
  ]
}
```

## 输入文本

{text}

## 输出要求

请按上述示例的格式输出 JSON，仅输出 JSON，不要添加任何解释。
````

---

## 模板 1.3：角色扮演式 NER 模板

**适用场景**：需要严格临床思维的场景，如病历文本，模拟眼科医生的诊断思维。

````
你是一位资深眼科主任医师，正在为下级医生整理病例资料。你需要从临床文本中精确提取所有相关的医学实体。

作为一名临床专家，你要特别注意：

1. **区分疾病和症状**："视力下降"是症状，"青光眼"是疾病。不要混淆。
2. **精确分期**：当文本描述了疾病严重程度时，提取为 Stage 类型（如"轻度干眼"而非仅仅"干眼"）。
3. **识别否定**：如"无视盘水肿"则不应提取"视盘水肿"。
4. **解剖定位**：注意实体涉及的解剖部位，如"黄斑区视网膜"应分别识别"黄斑区"和"视网膜"两个解剖实体。
5. **检查项目 vs 参数**："眼压测量"是检查方法（Examination），"眼压16mmHg"中的"16mmHg"是参数值而非实体。

## 临床文本

{text}

## 输出格式

```json
{
  "entities": [
    {
      "name": "标准术语名称",
      "type": "实体类型",
      "clinical_context": "该实体在文本中的临床语境简述（如：患者主诉/检查发现/既往史）",
      "negated": false
    }
  ],
  "clinical_summary": "一句话总结文本的核心临床信息"
}
```

注意：如果某个实体在文本中是被否定的（如"无视网膜脱离"），则 negated 设为 true，但仍列出该实体。
````

---

## 模板 1.4：约束消歧式 NER 模板

**适用场景**：用于对其他模板的输出进行校验和补充，特别适合处理模糊边界和嵌套实体。

````
## 任务

对以下眼科文本进行精细化的实体识别。你需要处理以下特殊情况：

### 需要处理的问题

1. **嵌套实体**："糖尿病视网膜病变"包含"糖尿病"（RiskFactor）和"视网膜"（Anatomy），请分别提取，同时将整体提取为 Disease。
2. **缩写消歧**："OCT"在眼科指光学相干断层扫描，不是普通CT。请根据上下文消歧。
3. **复合实体拆分**："左眼视力0.5"中，"左眼"是部位修饰，"视力"是检查项目，"0.5"是参数值（不提取）。
4. **同义词归并**："老年性白内障"和"年龄相关性白内障"是同一疾病的不同名称，请归并为标准名称并记录别名。
5. **数量与程度**："眼压升高"中"升高"是定性描述，整体作为 Symptom 提取，不要单独提取"升高"。

### 约束

- 每个提取的实体必须能在原文中找到对应片段（span）。
- 不确定的实体标注 confidence 为 "low"。
- 原文中没有但你知道的医学知识，不要补充。

## 输入文本

{text}

## 输出格式

```json
{
  "entities": [
    {
      "name": "标准名称",
      "type": "类型",
      "span": "原文片段",
      "aliases": ["别名"],
      "nested_entities": [
        {"name": "子实体名", "type": "子实体类型"}
      ],
      "confidence": "high | medium | low",
      "disambiguation_note": "消歧说明（如有）"
    }
  ]
}
```
````

---

## 集成使用建议

当对抽取精度要求较高时，建议将上述 4 种模板对**同一段文本**并行调用，然后通过以下策略集成结果：

### 投票策略

```python
def ensemble_entities(results: list[dict]) -> list[dict]:
    """
    多模板实体投票集成
    - 被>=3个模板识别的实体 → confidence: high
    - 被2个模板识别的实体 → confidence: medium
    - 仅被1个模板识别的实体 → confidence: low（需人工审核）
    """
    entity_counter = {}
    for result in results:
        for entity in result["entities"]:
            key = (entity["name"], entity["type"])
            entity_counter[key] = entity_counter.get(key, 0) + 1

    final_entities = []
    for (name, etype), count in entity_counter.items():
        confidence = "high" if count >= 3 else ("medium" if count >= 2 else "low")
        final_entities.append({
            "name": name,
            "type": etype,
            "confidence": confidence,
            "vote_count": count
        })

    return sorted(final_entities, key=lambda x: x["vote_count"], reverse=True)
```

### 冲突处理

| 冲突类型 | 处理方式 |
|---------|---------|
| 同一文本片段被标注为不同类型 | 优先采纳角色扮演模板（模板1.3）的结果 |
| 同一实体在不同模板中名称不同 | 术语词典归并（参照 Schema 术语对照表） |
| 某模板多提取了其他模板未识别的实体 | 记录但标记 confidence: low，留待人工审核 |
