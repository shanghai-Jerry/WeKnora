# 02 - 关系抽取模板集

> **用途**：从眼科医学文本中抽取实体之间的语义关系，形成知识三元组。提供 5 组按关系类型分类的子模板 + 通用关系抽取模板。每个模板均含 Few-shot 示例。

---

## 模板 2.1：疾病-症状关系抽取（HAS_SYMPTOM）

````
## 任务

从给定的眼科文本中抽取"疾病 → 症状/体征"的关系三元组。

## 关系定义

**HAS_SYMPTOM**：某疾病会表现出的临床症状或体征。

### 抽取规则

1. 只抽取文本中明确陈述的疾病-症状关系。
2. 注意区分"主要症状"和"可能症状"——均在 confidence 中体现。
3. 否定关系不抽取（如"不伴有眼红"）。
4. 如果症状与特定疾病分期相关，在 attributes 中标注 stage。

## 示例

### 示例 1
**输入**：干眼症患者常见症状包括眼干涩感、异物感、畏光、视疲劳，严重时可出现角膜上皮损害导致视力波动。
**输出**：
```json
{
  "triplets": [
    {
      "head": {"name": "干眼症", "type": "Disease"},
      "relation": "HAS_SYMPTOM",
      "tail": {"name": "眼干涩感", "type": "Symptom"},
      "evidence": "干眼症患者常见症状包括眼干涩感",
      "confidence": "high"
    },
    {
      "head": {"name": "干眼症", "type": "Disease"},
      "relation": "HAS_SYMPTOM",
      "tail": {"name": "异物感", "type": "Symptom"},
      "evidence": "异物感",
      "confidence": "high"
    },
    {
      "head": {"name": "干眼症", "type": "Disease"},
      "relation": "HAS_SYMPTOM",
      "tail": {"name": "畏光", "type": "Symptom"},
      "evidence": "畏光",
      "confidence": "high"
    },
    {
      "head": {"name": "干眼症", "type": "Disease"},
      "relation": "HAS_SYMPTOM",
      "tail": {"name": "视疲劳", "type": "Symptom"},
      "evidence": "视疲劳",
      "confidence": "high"
    },
    {
      "head": {"name": "干眼症", "type": "Disease"},
      "relation": "HAS_SYMPTOM",
      "tail": {"name": "视力波动", "type": "Symptom"},
      "evidence": "严重时可出现角膜上皮损害导致视力波动",
      "confidence": "medium",
      "attributes": {"severity": "严重期"}
    }
  ]
}
```

### 示例 2
**输入**：原发性开角型青光眼早期多无症状，随着病情进展可出现视野缺损、周边视野缩窄，晚期可致中心视力丧失。
**输出**：
```json
{
  "triplets": [
    {
      "head": {"name": "原发性开角型青光眼", "type": "Disease"},
      "relation": "HAS_SYMPTOM",
      "tail": {"name": "视野缺损", "type": "Symptom"},
      "evidence": "可出现视野缺损",
      "confidence": "high",
      "attributes": {"stage": "进展期"}
    },
    {
      "head": {"name": "原发性开角型青光眼", "type": "Disease"},
      "relation": "HAS_SYMPTOM",
      "tail": {"name": "中心视力丧失", "type": "Symptom"},
      "evidence": "晚期可致中心视力丧失",
      "confidence": "high",
      "attributes": {"stage": "晚期"}
    }
  ]
}
```

## 输入文本

{text}

## 输出要求

严格按上述 JSON 格式输出三元组数组。若无相关关系，返回 {"triplets": []}。
````

---

## 模板 2.2：疾病-检查关系抽取（DIAGNOSED_BY）

````
## 任务

从给定的眼科文本中抽取"疾病 → 诊断检查方法"的关系三元组。

## 关系定义

**DIAGNOSED_BY**：确诊或评估某疾病所需的检查方法。

### 抽取规则

1. 区分"确诊检查"和"筛查检查"，确诊检查 confidence 为 high，筛查检查标注为 medium。
2. 如果文本提及检查的灵敏度/特异性，记录在 attributes 中。
3. 检查项目使用标准名称（参照 Schema 术语词典）。

## 示例

### 示例 1
**输入**：糖尿病视网膜病变的诊断主要依据眼底检查，包括散瞳眼底照相和荧光素眼底血管造影（FFA）。OCT可用于定量评估黄斑水肿程度和视网膜厚度。对于早期DR，免散瞳眼底照相可作为筛查手段。
**输出**：
```json
{
  "triplets": [
    {
      "head": {"name": "糖尿病视网膜病变", "type": "Disease"},
      "relation": "DIAGNOSED_BY",
      "tail": {"name": "散瞳眼底照相", "type": "Examination"},
      "evidence": "诊断主要依据眼底检查，包括散瞳眼底照相",
      "confidence": "high",
      "attributes": {"purpose": "确诊"}
    },
    {
      "head": {"name": "糖尿病视网膜病变", "type": "Disease"},
      "relation": "DIAGNOSED_BY",
      "tail": {"name": "荧光素眼底血管造影（FFA）", "type": "Examination"},
      "evidence": "荧光素眼底血管造影（FFA）",
      "confidence": "high",
      "attributes": {"purpose": "确诊"}
    },
    {
      "head": {"name": "糖尿病视网膜病变", "type": "Disease"},
      "relation": "DIAGNOSED_BY",
      "tail": {"name": "光学相干断层扫描（OCT）", "type": "Examination"},
      "evidence": "OCT可用于定量评估黄斑水肿程度和视网膜厚度",
      "confidence": "high",
      "attributes": {"purpose": "评估黄斑水肿"}
    },
    {
      "head": {"name": "糖尿病视网膜病变", "type": "Disease"},
      "relation": "DIAGNOSED_BY",
      "tail": {"name": "免散瞳眼底照相", "type": "Examination"},
      "evidence": "对于早期DR，免散瞳眼底照相可作为筛查手段",
      "confidence": "medium",
      "attributes": {"purpose": "筛查"}
    }
  ]
}
```

## 输入文本

{text}

## 输出要求

严格按上述 JSON 格式输出。若无相关关系，返回 {"triplets": []}。
````

---

## 模板 2.3：疾病-治疗关系抽取（TREATED_BY）

````
## 任务

从给定的眼科文本中抽取"疾病 → 治疗手段"的关系三元组。

## 关系定义

**TREATED_BY**：某疾病的治疗方法，包括药物治疗、手术治疗、光学矫正、视觉训练等。

### 抽取规则

1. 区分一线治疗和二线/替代治疗，在 attributes 中标注 treatment_line。
2. 如果文本提及循证等级（如推荐等级I级、A级推荐），记录在 attributes 中。
3. 如果治疗与特定分期相关，标注 stage。

## 示例

### 示例 1
**输入**：近视防控措施包括：低浓度阿托品滴眼液（0.01%-0.05%）作为一线药物，角膜塑形镜（OK镜）可有效延缓近视进展，离焦设计框架眼镜作为替代方案。对于18岁以上成人近视，可考虑角膜屈光手术如LASIK或SMILE。
**输出**：
```json
{
  "triplets": [
    {
      "head": {"name": "近视", "type": "Disease"},
      "relation": "TREATED_BY",
      "tail": {"name": "阿托品滴眼液", "type": "Drug"},
      "evidence": "低浓度阿托品滴眼液（0.01%-0.05%）作为一线药物",
      "confidence": "high",
      "attributes": {
        "treatment_line": "一线治疗",
        "concentration": "0.01%-0.05%",
        "target_population": "青少年近视进展控制"
      }
    },
    {
      "head": {"name": "近视", "type": "Disease"},
      "relation": "TREATED_BY",
      "tail": {"name": "角膜塑形镜（OK镜）", "type": "Treatment"},
      "evidence": "角膜塑形镜（OK镜）可有效延缓近视进展",
      "confidence": "high",
      "attributes": {"treatment_line": "一线治疗"}
    },
    {
      "head": {"name": "近视", "type": "Disease"},
      "relation": "TREATED_BY",
      "tail": {"name": "离焦设计框架眼镜", "type": "Treatment"},
      "evidence": "离焦设计框架眼镜作为替代方案",
      "confidence": "high",
      "attributes": {"treatment_line": "替代方案"}
    },
    {
      "head": {"name": "近视", "type": "Disease"},
      "relation": "TREATED_BY",
      "tail": {"name": "LASIK", "type": "Treatment"},
      "evidence": "可考虑角膜屈光手术如LASIK",
      "confidence": "high",
      "attributes": {
        "treatment_line": "手术治疗",
        "target_population": "18岁以上成人"
      }
    },
    {
      "head": {"name": "近视", "type": "Disease"},
      "relation": "TREATED_BY",
      "tail": {"name": "SMILE", "type": "Treatment"},
      "evidence": "SMILE",
      "confidence": "high",
      "attributes": {
        "treatment_line": "手术治疗",
        "target_population": "18岁以上成人"
      }
    }
  ]
}
```

## 输入文本

{text}
````

---

## 模板 2.4：疾病-病因/风险关系抽取（CAUSED_BY / HAS_RISK_FACTOR / COMPLICATED_BY）

````
## 任务

从给定的眼科文本中抽取与病因学相关的三元组，包括：
- **CAUSED_BY**：疾病的直接病因
- **HAS_RISK_FACTOR**：疾病的危险因素
- **COMPLICATED_BY**：疾病的并发症

### 抽取规则

1. **CAUSED_BY**：文本明确说"A导致/引起B"，且为直接因果关系。
2. **HAS_RISK_FACTOR**：文本说"A是B的危险因素"或"A增加B的发生风险"。
3. **COMPLICATED_BY**：文本说"A的并发症包括B"或"A可并发B"。
4. 如果因果关系不确定（如"可能与...有关"），confidence 标记为 medium。

## 示例

### 示例 1
**输入**：糖尿病是糖尿病视网膜病变的主要病因。长期血糖控制不佳可加速DR进展。增殖期糖尿病视网膜病变可并发牵拉性视网膜脱离和新生血管性青光眼。高度近视（屈光度>-6.00D）是开角型青光眼的独立危险因素。
**输出**：
```json
{
  "triplets": [
    {
      "head": {"name": "糖尿病视网膜病变", "type": "Disease"},
      "relation": "CAUSED_BY",
      "tail": {"name": "糖尿病", "type": "RiskFactor"},
      "evidence": "糖尿病是糖尿病视网膜病变的主要病因",
      "confidence": "high"
    },
    {
      "head": {"name": "增殖期糖尿病视网膜病变", "type": "Disease"},
      "relation": "COMPLICATED_BY",
      "tail": {"name": "牵拉性视网膜脱离", "type": "Disease"},
      "evidence": "增殖期糖尿病视网膜病变可并发牵拉性视网膜脱离",
      "confidence": "high"
    },
    {
      "head": {"name": "增殖期糖尿病视网膜病变", "type": "Disease"},
      "relation": "COMPLICATED_BY",
      "tail": {"name": "新生血管性青光眼", "type": "Disease"},
      "evidence": "新生血管性青光眼",
      "confidence": "high"
    },
    {
      "head": {"name": "原发性开角型青光眼", "type": "Disease"},
      "relation": "HAS_RISK_FACTOR",
      "tail": {"name": "高度近视", "type": "RiskFactor"},
      "evidence": "高度近视（屈光度>-6.00D）是开角型青光眼的独立危险因素",
      "confidence": "high",
      "attributes": {"risk_definition": "屈光度>-6.00D", "independence": "独立危险因素"}
    }
  ]
}
```

## 输入文本

{text}
````

---

## 模板 2.5：禁忌症关系抽取（CONTRAINDICATED）

````
## 任务

从给定的眼科文本中抽取禁忌症关系三元组。

## 关系定义

**CONTRAINDICATED**：某条件下（疾病/风险因素/解剖条件）不能使用某种治疗方法。

### 抽取规则

1. 禁忌分为"绝对禁忌"和"相对禁忌"，在 attributes 中标注 contraindication_type。
2. 条件可以是疾病（Disease）、风险因素（RiskFactor）或解剖条件（Anatomy的属性）。
3. 如果文本提及"慎用"而非"禁用"，标注为相对禁忌。

## 示例

### 示例 1
**输入**：LASIK手术的绝对禁忌证包括角膜厚度不足（中央角膜厚度<480μm）、角膜地形图异常、活动性眼部感染、未控制的自身免疫性疾病。相对禁忌证包括干眼症、角膜内皮细胞密度偏低、暗瞳直径过大。ICL植入术要求前房深度≥2.8mm。
**输出**：
```json
{
  "triplets": [
    {
      "head": {"name": "角膜厚度不足", "type": "RiskFactor"},
      "relation": "CONTRAINDICATED",
      "tail": {"name": "LASIK", "type": "Treatment"},
      "evidence": "角膜厚度不足（中央角膜厚度<480μm）",
      "confidence": "high",
      "attributes": {
        "contraindication_type": "绝对禁忌",
        "threshold": "中央角膜厚度<480μm"
      }
    },
    {
      "head": {"name": "角膜地形图异常", "type": "RiskFactor"},
      "relation": "CONTRAINDICATED",
      "tail": {"name": "LASIK", "type": "Treatment"},
      "evidence": "角膜地形图异常",
      "confidence": "high",
      "attributes": {"contraindication_type": "绝对禁忌"}
    },
    {
      "head": {"name": "眼部感染", "type": "Disease"},
      "relation": "CONTRAINDICATED",
      "tail": {"name": "LASIK", "type": "Treatment"},
      "evidence": "活动性眼部感染",
      "confidence": "high",
      "attributes": {"contraindication_type": "绝对禁忌", "condition": "活动性"}
    },
    {
      "head": {"name": "干眼症", "type": "Disease"},
      "relation": "CONTRAINDICATED",
      "tail": {"name": "LASIK", "type": "Treatment"},
      "evidence": "相对禁忌证包括干眼症",
      "confidence": "high",
      "attributes": {"contraindication_type": "相对禁忌"}
    },
    {
      "head": {"name": "角膜内皮细胞密度偏低", "type": "RiskFactor"},
      "relation": "CONTRAINDICATED",
      "tail": {"name": "LASIK", "type": "Treatment"},
      "evidence": "角膜内皮细胞密度偏低",
      "confidence": "high",
      "attributes": {"contraindication_type": "相对禁忌"}
    },
    {
      "head": {"name": "前房深度不足", "type": "RiskFactor"},
      "relation": "CONTRAINDICATED",
      "tail": {"name": "ICL植入术", "type": "Treatment"},
      "evidence": "ICL植入术要求前房深度≥2.8mm",
      "confidence": "high",
      "attributes": {
        "contraindication_type": "绝对禁忌",
        "threshold": "前房深度<2.8mm"
      }
    }
  ]
}
```

## 输入文本

{text}
````

---

## 模板 2.6：通用关系抽取模板

**适用场景**：需要一次性抽取文本中所有类型的关系，而非针对特定关系类型。

````
## 任务

从给定的眼科医学文本中抽取所有类型的关系三元组。

## 可抽取的关系类型

| 关系 ID | 含义 | 头实体类型 | 尾实体类型 |
|---------|------|-----------|-----------|
| HAS_SYMPTOM | 疾病表现为某症状 | Disease | Symptom |
| LOCATED_IN | 解剖结构位于 | Anatomy | Anatomy |
| DIAGNOSED_BY | 疾病确诊需某检查 | Disease | Examination |
| TREATED_BY | 疾病用某方式治疗 | Disease | Treatment/Drug |
| CAUSED_BY | 疾病因某因素导致 | Disease | Disease/RiskFactor |
| HAS_RISK_FACTOR | 疾病的危险因素 | Disease | RiskFactor |
| COMPLICATED_BY | 疾病的并发症 | Disease | Disease |
| CONTRAINDICATED | 某条件下禁忌某治疗 | RiskFactor/Disease | Treatment |
| PROGRESSES_TO | 疾病进展为另一阶段 | Disease/Stage | Disease/Stage |
| HAS_STAGE | 疾病的分期分级 | Disease | Stage |
| SUBCLASS_OF | 概念分类层级 | Disease | Disease |
| INDICATES | 症状提示需做某检查 | Symptom | Examination |

## 示例

**输入**：年龄相关性黄斑变性（AMD）是老年人群视力丧失的主要原因之一。早期可无明显症状。眼底检查可见黄斑区玻璃膜疣，OCT可显示视网膜色素上皮层改变。湿性AMD可采取抗VEGF药物玻璃体腔注射治疗。
**输出**：
```json
{
  "triplets": [
    {
      "head": {"name": "年龄相关性黄斑变性", "type": "Disease"},
      "relation": "SUBCLASS_OF",
      "tail": {"name": "黄斑变性", "type": "Disease"},
      "evidence": "年龄相关性黄斑变性（AMD）",
      "confidence": "high",
      "attributes": {"aliases": ["AMD"]}
    },
    {
      "head": {"name": "年龄相关性黄斑变性", "type": "Disease"},
      "relation": "HAS_RISK_FACTOR",
      "tail": {"name": "高龄", "type": "RiskFactor"},
      "evidence": "老年人群",
      "confidence": "high"
    },
    {
      "head": {"name": "年龄相关性黄斑变性", "type": "Disease"},
      "relation": "DIAGNOSED_BY",
      "tail": {"name": "眼底检查", "type": "Examination"},
      "evidence": "眼底检查可见黄斑区玻璃膜疣",
      "confidence": "high"
    },
    {
      "head": {"name": "年龄相关性黄斑变性", "type": "Disease"},
      "relation": "DIAGNOSED_BY",
      "tail": {"name": "光学相干断层扫描（OCT）", "type": "Examination"},
      "evidence": "OCT可显示视网膜色素上皮层改变",
      "confidence": "high"
    },
    {
      "head": {"name": "湿性年龄相关性黄斑变性", "type": "Disease"},
      "relation": "SUBCLASS_OF",
      "tail": {"name": "年龄相关性黄斑变性", "type": "Disease"},
      "evidence": "湿性AMD",
      "confidence": "high"
    },
    {
      "head": {"name": "湿性年龄相关性黄斑变性", "type": "Disease"},
      "relation": "TREATED_BY",
      "tail": {"name": "抗VEGF药物玻璃体腔注射", "type": "Treatment"},
      "evidence": "可采取抗VEGF药物玻璃体腔注射治疗",
      "confidence": "high"
    }
  ]
}
```

## 输入文本

{text}

## 输出要求

严格按 JSON 格式输出。每个三元组必须包含 head、relation、tail、evidence、confidence。无相关关系时返回 {"triplets": []}。
````
