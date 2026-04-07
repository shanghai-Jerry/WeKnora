# 03 - 属性抽取模板集

> **用途**：从眼科医学文本中抽取实体的结构化属性信息（如 ICD 编码、分期体系、药物用法、检查参数等）。分为 3 个子模板，分别针对疾病、药物和检查实体。

---

## 模板 3.1：疾病属性抽取

````
## 任务

从给定的眼科文本中抽取**疾病实体**的结构化属性信息。

## 需抽取的属性字段

| 属性名 | 类型 | 说明 | 是否必填 |
|--------|------|------|---------|
| `name` | string | 疾病标准名称 | ✅ |
| `aliases` | array | 别名列表 | ❌ |
| `icd_code` | string | ICD-10 编码 | ❌（如有则提取） |
| `definition` | string | 疾病定义 | ❌ |
| `prevalence` | string | 发病率/患病率 | ❌ |
| `onset_age` | string | 好发年龄 | ❌ |
| `chronicity` | string | 急/慢性分类 | ❌ |
| `inheritance` | string | 遗传性描述 | ❌ |
| `severity_levels` | array | 分级/分期体系 | ❌ |
| `prognosis` | string | 预后描述 | ❌ |
| `key_clinical_features` | array | 主要临床特征（要点列表） | ❌ |

## 示例

### 示例 1
**输入**：原发性开角型青光眼（POAG，ICD-10: H40.1）是一种慢性进展性视神经病变，以视神经萎缩和特征性视野缺损为特征。全球患病率约2%-3%，40岁以上人群患病率显著增高。早期通常无症状，是导致不可逆性盲的主要原因之一。眼压升高是主要危险因素，但部分患者眼压在正常范围内（正常眼压性青光眼）。治疗目标为降低眼压至目标水平，延缓视神经损伤进展。预后取决于诊断时机和治疗的依从性。
**输出**：
```json
{
  "entity": {
    "name": "原发性开角型青光眼",
    "type": "Disease",
    "aliases": ["POAG", "开角型青光眼"]
  },
  "attributes": {
    "icd_code": "H40.1",
    "definition": "一种慢性进展性视神经病变，以视神经萎缩和特征性视野缺损为特征",
    "prevalence": "全球约2%-3%",
    "onset_age": "40岁以上显著增高",
    "chronicity": "慢性进展性",
    "key_clinical_features": [
      "视神经萎缩",
      "特征性视野缺损",
      "眼压升高"
    ],
    "prognosis": "取决于诊断时机和治疗依从性；不可逆性盲的主要原因之一"
  },
  "evidence_map": {
    "icd_code": "ICD-10: H40.1",
    "definition": "原发性开角型青光眼（POAG）是一种慢性进展性视神经病变，以视神经萎缩和特征性视野缺损为特征",
    "prevalence": "全球患病率约2%-3%",
    "onset_age": "40岁以上人群患病率显著增高",
    "chronicity": "慢性进展性视神经病变",
    "prognosis": "预后取决于诊断时机和治疗的依从性"
  }
}
```

### 示例 2
**输入**：干眼症（ICD-10: H04.12）是由泪液质或量异常或泪液动力学异常引起的泪膜不稳定和/或眼表损害。根据严重程度分为轻度、中度和重度。轻度以主观症状为主，无或仅有轻度眼表体征；中度有明显的眼表体征；重度出现角膜损害、视力受影响。我国干眼症发病率约21%-30%，女性和老年人多见。
**输出**：
```json
{
  "entity": {
    "name": "干眼症",
    "type": "Disease",
    "aliases": ["干眼", "干眼综合征"]
  },
  "attributes": {
    "icd_code": "H04.12",
    "definition": "由泪液质或量异常或泪液动力学异常引起的泪膜不稳定和/或眼表损害",
    "prevalence": "我国约21%-30%",
    "severity_levels": ["轻度", "中度", "重度"],
    "key_clinical_features": [
      "主观症状（干涩、异物感、畏光等）",
      "泪膜不稳定",
      "眼表损害",
      "角膜损害（重度）"
    ]
  },
  "evidence_map": {
    "icd_code": "ICD-10: H04.12",
    "definition": "由泪液质或量异常或泪液动力学异常引起的泪膜不稳定和/或眼表损害",
    "severity_levels": "根据严重程度分为轻度、中度和重度",
    "prevalence": "我国干眼症发病率约21%-30%"
  }
}
```

## 输入文本

{text}

## 输出要求

严格按上述 JSON 格式输出。仅提取文本中明确提及的属性，未提及的字段省略。如果文本中没有疾病实体，返回 {"entity": null, "attributes": {}}。
````

---

## 模板 3.2：药物属性抽取

````
## 任务

从给定的眼科文本中抽取**药物实体**的结构化属性信息。

## 需抽取的属性字段

| 属性名 | 类型 | 说明 | 是否必填 |
|--------|------|------|---------|
| `name` | string | 药物标准名称 | ✅ |
| `aliases` | array | 商品名/缩写 | ❌ |
| `ingredients` | array | 有效成分 | ❌ |
| `drug_class` | string | 药物分类（如前列腺素类似物、抗生素等） | ❌ |
| `dosage_form` | string | 剂型（滴眼液、眼膏、口服等） | ❌ |
| `concentration` | string | 浓度/规格 | ❌ |
| `usage` | string | 用法用量 | ❌ |
| `mechanism` | string | 作用机制 | ❌ |
| `indications` | array | 适应症 | ❌ |
| `contraindications` | array | 禁忌症 | ❌ |
| `adverse_reactions` | array | 不良反应 | ❌ |
| `insurance_category` | string | 医保类别 | ❌ |
| `storage` | string | 储存条件 | ❌ |

## 示例

### 示例 1
**输入**：0.01%硫酸阿托品滴眼液是目前循证证据最充分的近视防控药物。其作用机制为拮抗巩膜上的M1受体，减缓眼轴增长。推荐用法：每晚睡前1滴，双眼用药。常见不良反应包括畏光、近视力模糊（尤其在阅读时）、脸红。禁忌证包括青光眼或青光眼倾向者、Down综合征、对阿托品过敏者。需避光冷藏保存。
**输出**：
```json
{
  "entity": {
    "name": "硫酸阿托品滴眼液",
    "type": "Drug",
    "aliases": ["阿托品滴眼液"]
  },
  "attributes": {
    "ingredients": ["硫酸阿托品"],
    "drug_class": "M受体拮抗剂（抗胆碱能药）",
    "dosage_form": "滴眼液",
    "concentration": "0.01%",
    "usage": "每晚睡前1滴，双眼用药",
    "mechanism": "拮抗巩膜上的M1受体，减缓眼轴增长",
    "indications": ["近视进展控制"],
    "contraindications": [
      "青光眼或青光眼倾向",
      "Down综合征",
      "对阿托品过敏"
    ],
    "adverse_reactions": [
      "畏光",
      "近视力模糊",
      "脸红"
    ],
    "storage": "避光冷藏"
  },
  "evidence_map": {
    "concentration": "0.01%硫酸阿托品滴眼液",
    "usage": "每晚睡前1滴，双眼用药",
    "mechanism": "拮抗巩膜上的M1受体，减缓眼轴增长",
    "contraindications": "禁忌证包括青光眼或青光眼倾向者、Down综合征、对阿托品过敏者",
    "adverse_reactions": "常见不良反应包括畏光、近视力模糊（尤其在阅读时）、脸红",
    "storage": "需避光冷藏保存"
  }
}
```

### 示例 2
**输入**：拉坦前列素滴眼液（适利达®）属于前列腺素类降眼压药物，是目前治疗开角型青光眼的一线用药。浓度为0.005%，每日1次，晚间滴用。通过增加葡萄膜巩膜外流降低眼压。常见不良反应为虹膜色素加深、睫毛增长、结膜充血。活动性眼部炎症、葡萄膜炎患者禁用。
**输出**：
```json
{
  "entity": {
    "name": "拉坦前列素滴眼液",
    "type": "Drug",
    "aliases": ["适利达", "Xalatan"]
  },
  "attributes": {
    "ingredients": ["拉坦前列素"],
    "drug_class": "前列腺素类降眼压药物",
    "dosage_form": "滴眼液",
    "concentration": "0.005%",
    "usage": "每日1次，晚间滴用",
    "mechanism": "增加葡萄膜巩膜外流降低眼压",
    "indications": ["开角型青光眼"],
    "contraindications": [
      "活动性眼部炎症",
      "葡萄膜炎"
    ],
    "adverse_reactions": [
      "虹膜色素加深",
      "睫毛增长",
      "结膜充血"
    ]
  },
  "evidence_map": {
    "drug_class": "属于前列腺素类降眼压药物",
    "concentration": "浓度为0.005%",
    "usage": "每日1次，晚间滴用",
    "mechanism": "通过增加葡萄膜巩膜外流降低眼压",
    "contraindications": "活动性眼部炎症、葡萄膜炎患者禁用",
    "adverse_reactions": "常见不良反应为虹膜色素加深、睫毛增长、结膜充血"
  }
}
```

## 输入文本

{text}

## 输出要求

严格按 JSON 格式输出。仅提取文本中明确提及的属性。无药物实体时返回 {"entity": null, "attributes": {}}。
````

---

## 模板 3.3：检查方法属性抽取

````
## 任务

从给定的眼科文本中抽取**检查方法实体**的结构化属性信息。

## 需抽取的属性字段

| 属性名 | 类型 | 说明 | 是否必填 |
|--------|------|------|---------|
| `name` | string | 检查标准名称 | ✅ |
| `aliases` | array | 缩写/别名 | ❌ |
| `category` | string | 检查类别（影像检查/视功能检查/器械检查/实验室检查） | ❌ |
| `purpose` | string | 检查目的和临床意义 | ❌ |
| `equipment` | string | 所需设备 | ❌ |
| `sensitivity` | string | 灵敏度 | ❌ |
| `specificity` | string | 特异性 | ❌ |
| `normal_range` | string | 正常参考值 | ❌ |
| `duration` | string | 检查所需时长 | ❌ |
| `preparation` | array | 检查前准备（如散瞳） | ❌ |
| `limitations` | array | 检查局限性 | ❌ |

## 示例

### 示例 1
**输入**：光学相干断层扫描（OCT）是一种高分辨率、非接触性的眼部影像学检查技术，利用低相干光干涉原理获取视网膜横断面图像。频域OCT（SD-OCT）轴向分辨率可达5-7μm，广泛用于黄斑疾病、青光眼和视网膜疾病的诊断与随访。检查无需散瞳，约5-10分钟完成。可定量测量视网膜神经纤维层（RNFL）厚度，正常RNFL厚度≥80μm。OCT对黄斑水肿的检测灵敏度高，但对视网膜深层结构显示有限。
**输出**：
```json
{
  "entity": {
    "name": "光学相干断层扫描（OCT）",
    "type": "Examination",
    "aliases": ["OCT", "SD-OCT"]
  },
  "attributes": {
    "category": "影像检查",
    "purpose": "获取视网膜横断面图像，用于黄斑疾病、青光眼和视网膜疾病的诊断与随访",
    "equipment": "频域OCT设备",
    "sensitivity": "对黄斑水肿检测灵敏度高",
    "normal_range": {
      "RNFL厚度": "≥80μm"
    },
    "duration": "5-10分钟",
    "preparation": ["无需散瞳"],
    "limitations": [
      "对视网膜深层结构显示有限"
    ],
    "technical_specs": {
      "resolution": "轴向分辨率5-7μm（SD-OCT）"
    }
  },
  "evidence_map": {
    "purpose": "广泛用于黄斑疾病、青光眼和视网膜疾病的诊断与随访",
    "equipment": "频域OCT",
    "sensitivity": "OCT对黄斑水肿的检测灵敏度高",
    "normal_range": "正常RNFL厚度≥80μm",
    "duration": "约5-10分钟完成",
    "preparation": "检查无需散瞳",
    "limitations": "对视网膜深层结构显示有限"
  }
}
```

### 示例 2
**输入**：Goldmann压平眼压计是眼压测量的金标准。测量前需表面麻醉（0.5%丙美卡因）。正常眼压范围为10-21mmHg。测量时患者取坐位，注视前方目标。每次测量约3-5分钟。局限性在于需要患者配合，角膜病变（如角膜瘢痕）可能影响测量准确性。
**输出**：
```json
{
  "entity": {
    "name": "Goldmann压平眼压计",
    "type": "Examination",
    "aliases": ["Goldmann眼压计", "压平眼压计"]
  },
  "attributes": {
    "category": "器械检查",
    "purpose": "眼压测量（金标准）",
    "equipment": "裂隙灯-mounted Goldmann压平眼压计",
    "normal_range": "10-21mmHg",
    "duration": "3-5分钟",
    "preparation": [
      "表面麻醉（0.5%丙美卡因）"
    ],
    "limitations": [
      "需要患者配合",
      "角膜病变（如角膜瘢痕）可能影响测量准确性"
    ]
  },
  "evidence_map": {
    "purpose": "Goldmann压平眼压计是眼压测量的金标准",
    "normal_range": "正常眼压范围为10-21mmHg",
    "duration": "每次测量约3-5分钟",
    "preparation": "测量前需表面麻醉（0.5%丙美卡因）",
    "limitations": "角膜病变（如角膜瘢痕）可能影响测量准确性"
  }
}
```

## 输入文本

{text}

## 输出要求

严格按 JSON 格式输出。仅提取文本中明确提及的属性。无检查实体时返回 {"entity": null, "attributes": {}}。
````
