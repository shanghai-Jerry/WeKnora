# 04 - 三元组联合抽取模板集

> **用途**：单次 LLM 调用同时完成实体识别、关系抽取和属性填充，输出完整的知识三元组。适用于端到端的自动化构建流水线。

---

## 模板 4.1：通用联合抽取模板（适合指南段落）

````
## 任务

你是眼科视光知识图谱的自动化知识抽取系统。请从以下医学文本中**同时**完成：

1. **实体识别**：识别所有眼科相关实体（8种类型）
2. **关系抽取**：识别实体之间的语义关系（12种关系类型）
3. **属性填充**：提取实体的关键属性信息

## 实体类型（8种）

| ID | 名称 | 说明 |
|----|------|------|
| Anatomy | 解剖结构 | 眼球及附件结构（角膜、视网膜、晶状体、视神经等） |
| Disease | 疾病/异常 | 眼科疾病（青光眼、白内障、干眼症、近视等） |
| Symptom | 症状/体征 | 临床症状和体征（视力下降、眼痛、飞蚊症等） |
| Examination | 检查方法 | 诊断检查手段（OCT、视野检查、验光等） |
| Stage | 诊断/分期 | 疾病分级分期（轻度干眼、增殖期糖网等） |
| Treatment | 治疗干预 | 治疗手段（LASIK、角膜塑形镜、抗VEGF等） |
| Drug | 药物/器械 | 药物和器械（阿托品滴眼液、人工晶状体等） |
| RiskFactor | 风险因素 | 危险因素（糖尿病、高龄、家族史等） |

## 关系类型（12种）

HAS_SYMPTOM | LOCATED_IN | DIAGNOSED_BY | TREATED_BY | CAUSED_BY | HAS_RISK_FACTOR | COMPLICATED_BY | CONTRAINDICATED | PROGRESSES_TO | HAS_STAGE | SUBCLASS_OF | INDICATES

## 核心规则

1. 只抽取文本中明确陈述的知识，不推断、不补充。
2. 否定句中的关系不抽取（如"不伴眼痛"不生成 HAS_SYMPTOM）。
3. 每个三元组必须附带 evidence（原文证据片段）。
4. 不确定的内容 confidence 设为 "low"。
5. 实体使用标准医学术语名称。

## 示例

**输入**：干眼症是由于泪液分泌不足或蒸发过强导致泪膜不稳定的眼表疾病。患者常主诉眼干涩、异物感和畏光。诊断基于症状评估、泪液分泌试验（Schirmer I）和泪膜破裂时间（BUT）。轻度干眼以人工泪液治疗为主，中重度可加用环孢素滴眼液抗炎治疗。

**输出**：
```json
{
  "triplets": [
    {
      "head": {"name": "干眼症", "type": "Disease"},
      "relation": "CAUSED_BY",
      "tail": {"name": "泪液分泌不足", "type": "RiskFactor"},
      "evidence": "由于泪液分泌不足或蒸发过强导致泪膜不稳定",
      "confidence": "high"
    },
    {
      "head": {"name": "干眼症", "type": "Disease"},
      "relation": "CAUSED_BY",
      "tail": {"name": "泪液蒸发过强", "type": "RiskFactor"},
      "evidence": "由于泪液分泌不足或蒸发过强导致泪膜不稳定",
      "confidence": "high"
    },
    {
      "head": {"name": "干眼症", "type": "Disease"},
      "relation": "HAS_SYMPTOM",
      "tail": {"name": "眼干涩", "type": "Symptom"},
      "evidence": "患者常主诉眼干涩、异物感和畏光",
      "confidence": "high"
    },
    {
      "head": {"name": "干眼症", "type": "Disease"},
      "relation": "HAS_SYMPTOM",
      "tail": {"name": "异物感", "type": "Symptom"},
      "evidence": "患者常主诉眼干涩、异物感和畏光",
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
      "relation": "DIAGNOSED_BY",
      "tail": {"name": "泪液分泌试验（Schirmer I）", "type": "Examination"},
      "evidence": "诊断基于症状评估、泪液分泌试验（Schirmer I）和泪膜破裂时间（BUT）",
      "confidence": "high"
    },
    {
      "head": {"name": "干眼症", "type": "Disease"},
      "relation": "DIAGNOSED_BY",
      "tail": {"name": "泪膜破裂时间（BUT）", "type": "Examination"},
      "evidence": "泪膜破裂时间（BUT）",
      "confidence": "high"
    },
    {
      "head": {"name": "轻度干眼", "type": "Stage"},
      "relation": "HAS_STAGE",
      "tail": null,
      "evidence": "轻度干眼以人工泪液治疗为主",
      "confidence": "high",
      "attributes": {"parent_disease": "干眼症", "treatment": "人工泪液"}
    },
    {
      "head": {"name": "轻度干眼", "type": "Stage"},
      "relation": "SUBCLASS_OF",
      "tail": {"name": "干眼症", "type": "Disease"},
      "evidence": "轻度干眼",
      "confidence": "high"
    },
    {
      "head": {"name": "干眼症", "type": "Disease"},
      "relation": "TREATED_BY",
      "tail": {"name": "人工泪液", "type": "Drug"},
      "evidence": "轻度干眼以人工泪液治疗为主",
      "confidence": "high",
      "attributes": {"applicable_stage": "轻度"}
    },
    {
      "head": {"name": "干眼症", "type": "Disease"},
      "relation": "TREATED_BY",
      "tail": {"name": "环孢素滴眼液", "type": "Drug"},
      "evidence": "中重度可加用环孢素滴眼液抗炎治疗",
      "confidence": "high",
      "attributes": {"applicable_stage": "中重度", "mechanism": "抗炎治疗"}
    }
  ]
}
```

## 输入文本

{text}

## 输出要求

严格按上述 JSON 格式输出三元组数组。若无知识可抽取，返回 {"triplets": []}。
````

---

## 模板 4.2：文献摘要联合抽取

````
## 任务

从眼科科研文献的摘要中抽取知识三元组。文献摘要通常包含：研究背景、目的、方法、结果和结论。

## 文献摘要的特点

文献摘要有一些不同于临床指南的语言特征，抽取时需要特别注意：

1. **研究性结论**：关注文献的**主要发现**和**结论**，忽略研究方法和统计细节。
2. **数据关联**：如"研究发现A与B显著相关（P<0.05）"，抽取为 HAS_RISK_FACTOR 或 CAUSED_BY。
3. **效果评估**：如"治疗有效率达85%"，在 attributes 中记录 efficacy 数据。
4. **样本描述**：不抽取样本量、研究设计等信息。
5. **新知识标注**：文献通常包含前沿发现，如果结论与已知知识矛盾，仍如实抽取，confidence 标记为 medium。

## 示例

**输入**：Purpose: To evaluate the efficacy and safety of low-dose atropine (0.01%) for myopia control in Chinese children. Methods: A randomized, double-masked, placebo-controlled trial was conducted with 220 children aged 6-12 years. Results: After 2 years, the atropine group showed significantly less myopia progression (-0.55±0.32 D vs -1.12±0.48 D, P<0.001). No serious adverse events were reported. The most common side effect was mild photophobia (12.7%). Conclusion: Low-dose atropine 0.01% is effective and safe for myopia control in Chinese children, with a favorable safety profile.
**输出**：
```json
{
  "triplets": [
    {
      "head": {"name": "近视", "type": "Disease"},
      "relation": "TREATED_BY",
      "tail": {"name": "硫酸阿托品滴眼液", "type": "Drug"},
      "evidence": "atropine group showed significantly less myopia progression",
      "confidence": "high",
      "attributes": {
        "concentration": "0.01%",
        "target_population": "中国6-12岁儿童",
        "treatment_duration": "2年",
        "efficacy": "年均近视进展 -0.55±0.32 D（对照组 -1.12±0.48 D）",
        "evidence_level": "RCT（I级证据）"
      }
    },
    {
      "head": {"name": "硫酸阿托品滴眼液", "type": "Drug"},
      "relation": "HAS_SYMPTOM",
      "tail": {"name": "畏光", "type": "Symptom"},
      "evidence": "The most common side effect was mild photophobia (12.7%)",
      "confidence": "high",
      "attributes": {
        "adverse_reaction_rate": "12.7%",
        "severity": "轻度"
      }
    }
  ]
}
```

## 输入文本

{text}
````

---

## 模板 4.3：药品说明书联合抽取

````
## 任务

从眼科药品说明书中抽取完整的药物知识，包括药物属性、适应症关系、禁忌症关系和不良反应关系。

## 药品说明书的特点

1. **结构化程度高**：说明书通常有明确的章节（成分、适应症、用法用量、不良反应、禁忌等）。
2. **信息密度大**：一段文本可能包含多种关系，需要仔细拆分。
3. **标准术语**：说明书使用的术语较规范，便于标准化。

## 抽取优先级

| 优先级 | 抽取内容 | 输出方式 |
|--------|---------|---------|
| 1 | 药物属性（成分、剂型、规格） | `entity.attributes` |
| 2 | 适应症关系（Drug → Disease via TREATED_BY 反向） | `triplets` |
| 3 | 禁忌症关系 | `triplets` (CONTRAINDICATED) |
| 4 | 不良反应 | `triplets` (HAS_SYMPTOM) |
| 5 | 用法用量 | `entity.attributes.usage` |

## 示例

**输入**：
【药品名称】通用名：左氧氟沙星滴眼液；商品名：可乐必妥®
【成分】活性成分：左氧氟沙星 0.5%（5mg/mL）
【适应症】本品适用于治疗敏感菌引起的细菌性结膜炎、角膜炎、睑缘炎、泪囊炎等眼部感染。
【用法用量】推荐剂量：每次1-2滴，每日3-4次。疗程一般为7-14天。
【不良反应】常见（≥1%）：局部刺激感、异物感、结膜充血。偶见（0.1%-1%）：点状角膜炎、眼睑水肿、瘙痒。
【禁忌】对喹诺酮类药物过敏者禁用。孕妇及哺乳期妇女慎用。
【注意事项】使用时勿接触软性隐形眼镜。

**输出**：
```json
{
  "drug_entity": {
    "name": "左氧氟沙星滴眼液",
    "type": "Drug",
    "aliases": ["可乐必妥", "Cravit"]
  },
  "drug_attributes": {
    "ingredients": ["左氧氟沙星"],
    "concentration": "0.5%（5mg/mL）",
    "drug_class": "喹诺酮类抗生素",
    "usage": "每次1-2滴，每日3-4次",
    "treatment_duration": "7-14天",
    "precautions": ["使用时勿接触软性隐形眼镜"]
  },
  "triplets": [
    {
      "head": {"name": "细菌性结膜炎", "type": "Disease"},
      "relation": "TREATED_BY",
      "tail": {"name": "左氧氟沙星滴眼液", "type": "Drug"},
      "evidence": "适用于治疗敏感菌引起的细菌性结膜炎",
      "confidence": "high"
    },
    {
      "head": {"name": "角膜炎", "type": "Disease"},
      "relation": "TREATED_BY",
      "tail": {"name": "左氧氟沙星滴眼液", "type": "Drug"},
      "evidence": "角膜炎",
      "confidence": "high"
    },
    {
      "head": {"name": "睑缘炎", "type": "Disease"},
      "relation": "TREATED_BY",
      "tail": {"name": "左氧氟沙星滴眼液", "type": "Drug"},
      "evidence": "睑缘炎",
      "confidence": "high"
    },
    {
      "head": {"name": "泪囊炎", "type": "Disease"},
      "relation": "TREATED_BY",
      "tail": {"name": "左氧氟沙星滴眼液", "type": "Drug"},
      "evidence": "泪囊炎",
      "confidence": "high"
    },
    {
      "head": {"name": "喹诺酮类药物过敏", "type": "RiskFactor"},
      "relation": "CONTRAINDICATED",
      "tail": {"name": "左氧氟沙星滴眼液", "type": "Drug"},
      "evidence": "对喹诺酮类药物过敏者禁用",
      "confidence": "high",
      "attributes": {"contraindication_type": "绝对禁忌"}
    },
    {
      "head": {"name": "妊娠", "type": "RiskFactor"},
      "relation": "CONTRAINDICATED",
      "tail": {"name": "左氧氟沙星滴眼液", "type": "Drug"},
      "evidence": "孕妇及哺乳期妇女慎用",
      "confidence": "high",
      "attributes": {"contraindication_type": "相对禁忌"}
    },
    {
      "head": {"name": "左氧氟沙星滴眼液", "type": "Drug"},
      "relation": "HAS_SYMPTOM",
      "tail": {"name": "局部刺激感", "type": "Symptom"},
      "evidence": "常见（≥1%）：局部刺激感",
      "confidence": "high",
      "attributes": {"adverse_reaction_frequency": "≥1%"}
    },
    {
      "head": {"name": "左氧氟沙星滴眼液", "type": "Drug"},
      "relation": "HAS_SYMPTOM",
      "tail": {"name": "结膜充血", "type": "Symptom"},
      "evidence": "结膜充血",
      "confidence": "high",
      "attributes": {"adverse_reaction_frequency": "≥1%"}
    },
    {
      "head": {"name": "左氧氟沙星滴眼液", "type": "Drug"},
      "relation": "HAS_SYMPTOM",
      "tail": {"name": "点状角膜炎", "type": "Disease"},
      "evidence": "偶见（0.1%-1%）：点状角膜炎",
      "confidence": "high",
      "attributes": {"adverse_reaction_frequency": "0.1%-1%"}
    }
  ]
}
```

## 输入文本

{text}
````

---

## 端到端 Pipeline 设计

```python
"""
联合抽取 Pipeline 示例
"""
import json
from openai import OpenAI

client = OpenAI()

SYSTEM_PROMPT = """
你是一名眼科视光领域的医学知识抽取专家。
你的任务是从给定的眼科医学文本中抽取结构化知识三元组。
核心原则：忠实原文、标准术语、粒度适中、不确定标记、证据溯源。
严格按 JSON 格式输出。
"""


def extract_triplets(text: str, source_type: str, template: str = "joint") -> dict:
    """
    使用联合抽取模板从文本中提取知识三元组

    Args:
        text: 输入文本
        source_type: 数据源类型 (guideline/literature/drug_label/textbook/ehr)
        template: 模板类型 (joint/literature/drug_label)

    Returns:
        抽取结果字典（三元组 + 元数据）
    """
    # 根据数据源类型选择模板
    templates = {
        "joint": JOINT_EXTRACTION_TEMPLATE,
        "literature": LITERATURE_EXTRACTION_TEMPLATE,
        "drug_label": DRUG_LABEL_EXTRACTION_TEMPLATE,
    }
    prompt = templates[template].replace("{text}", text)

    response = client.chat.completions.create(
        model="gpt-4",
        messages=[
            {"role": "system", "content": SYSTEM_PROMPT},
            {"role": "user", "content": prompt}
        ],
        temperature=0.1,  # 低温度减少随机性
        response_format={"type": "json_object"}
    )

    result = json.loads(response.choices[0].message.content)

    # 添加元数据
    result["metadata"] = {
        "source_type": source_type,
        "model": "gpt-4",
        "extraction_strategy": template,
        "timestamp": datetime.now().isoformat()
    }

    return result
```
