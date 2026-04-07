# 06 - 质量控制模板集

> **用途**：对知识抽取结果进行质量校验、冲突检测和人工审核辅助。这些模板可在抽取流水线末尾串联使用，也可独立调用。

---

## 模板 6.1：抽取结果自检模板

````
## 任务

你是一名知识图谱质量控制专家。请对以下从眼科医学文本中抽取的知识三元组进行质量自检。

## 自检维度

### 1. 实体一致性
- [ ] 实体名称是否使用标准医学术语？
- [ ] 实体类型是否正确？（如"视力下降"应为 Symptom 而非 Disease）
- [ ] 同一实体在不同三元组中名称是否一致？

### 2. 关系有效性
- [ ] head 实体类型是否符合该关系的 domain 约束？
- [ ] tail 实体类型是否符合该关系的 range 约束？
- [ ] 关系方向是否正确？（如应该是 Disease→Symptom 而非 Symptom→Disease）

### 3. 忠实性验证
- [ ] evidence 字段是否真实对应原文片段？
- [ ] 三元组是否忠实于原文含义，没有过度推断？
- [ ] 否定句中的关系是否被正确排除？

### 4. 完整性检查
- [ ] 是否遗漏了文本中明确提到的实体或关系？
- [ ] 关键属性（如分期的治疗推荐）是否被提取？

## 本体约束参考

**关系 domain-range 约束**：
| 关系 | head 类型 | tail 类型 |
|------|----------|----------|
| HAS_SYMPTOM | Disease | Symptom |
| LOCATED_IN | Anatomy | Anatomy |
| DIAGNOSED_BY | Disease | Examination |
| TREATED_BY | Disease | Treatment/Drug |
| CAUSED_BY | Disease | Disease/RiskFactor |
| INDICATES | Symptom | Examination |
| CONTRAINDICATED | RiskFactor/Disease | Treatment |
| PROGRESSES_TO | Disease/Stage | Disease/Stage |
| HAS_STAGE | Disease | Stage |
| SUBCLASS_OF | Disease | Disease |
| HAS_RISK_FACTOR | Disease | RiskFactor |
| COMPLICATED_BY | Disease | Disease |

## 输入

### 原始文本
{original_text}

### 抽取结果
{extraction_result}

## 输出格式

```json
{
  "quality_report": {
    "total_triplets": 10,
    "valid_triplets": 8,
    "invalid_triplets": 2
  },
  "issues": [
    {
      "triplet_index": 3,
      "issue_type": "domain_violation | range_violation | unfaithful | inconsistency | missing_entity",
      "description": "问题描述",
      "suggestion": "修正建议",
      "severity": "critical | warning | info"
    }
  ],
  "corrections": [
    {
      "triplet_index": 3,
      "original": {"head": "错误", "relation": "错误", "tail": "错误"},
      "corrected": {"head": "修正", "relation": "修正", "tail": "修正"},
      "reason": "修正原因"
    }
  ],
  "missing_triplets": [
    {
      "head": {"name": "遗漏实体1", "type": "类型"},
      "relation": "关系",
      "tail": {"name": "遗漏实体2", "type": "类型"},
      "evidence": "原文中的证据"
    }
  ]
}
```
````

---

## 模板 6.2：多源冲突消解模板

````
## 任务

当同一知识在不同数据源中出现矛盾时，需要你进行冲突消解。

## 消解原则

| 冲突场景 | 消解策略 |
|---------|---------|
| 指南 vs 文献 | 优先采信指南（更新版本的指南 > 旧指南 > 文献） |
| 高质量研究 vs 低质量研究 | 优先采信高质量研究（RCT > 队列研究 > 病例报告） |
| 时间差异 | 采信更新的信息（2025年文献 > 2020年文献） |
| 中外差异 | 中国指南/数据优先用于中国人群，国际指南作为补充 |
| 完全矛盾 | 双方均保留，标注争议，标记 pending_review |

## 输入

### 知识三元组 A（来源：{source_a}，时间：{date_a}）
{triplet_a}

### 知识三元组 B（来源：{source_b}，时间：{date_b}）
{triplet_b}

### 冲突描述
{conflict_description}

## 输出格式

```json
{
  "resolution": "accept_a | accept_b | merge | both_retain | pending_review",
  "reason": "消解决策的原因说明",
  "accepted_triplet": {
    "head": {"name": "实体名", "type": "类型"},
    "relation": "关系",
    "tail": {"name": "实体名", "type": "类型"},
    "attributes": {
      "evidence_level": "综合评估后的证据等级",
      "sources": ["来源A", "来源B"],
      "disputed": false
    }
  },
  "conflict_note": "如双方保留，此处记录争议说明"
}
```
````

---

## 模板 6.3：人工审核辅助模板

````
## 任务

你是一名眼科知识图谱审核助手。以下抽取结果已被标记为需要人工审核。请为审核人员提供辅助信息，帮助他们快速判断是否接受、修改或拒绝。

## 输入

### 待审核的三元组
{triplet}

### 审核原因
{review_reason}

### 原始文本
{original_text}

## 输出格式

```json
{
  "triplet_summary": "一句话总结该三元组的含义",
  "review_checklist": {
    "is_medically_accurate": {
      "question": "该三元组在医学上是否准确？",
      "hint": "可参考的医学知识：{参考信息}",
      "risk_if_wrong": "如果该三元组错误可能造成的后果"
    },
    "is_faithful_to_source": {
      "question": "该三元组是否忠实于原文？",
      "hint": "原文相关片段：{evidence}",
      "ambiguity_note": "原文是否存在歧义？"
    },
    "is_necessary": {
      "question": "该三元组是否有必要收录到知识图谱中？",
      "hint": "考虑该知识的临床价值"
    }
  },
  "recommended_action": "accept | modify | reject | escalate",
  "suggested_modification": "如果建议修改，提供修改后的三元组"
}
```
````

---

## 模板 6.4：实体消歧/归并模板

````
## 任务

以下实体列表中可能存在同一实体的不同表述。请进行消歧和归并。

## 归并规则

1. **完全相同**：名称完全一致 → 直接去重
2. **标准名 vs 别名**：如"老年性白内障"和"年龄相关性白内障" → 归并为标准名称（参考 Schema 术语词典）
3. **缩写 vs 全称**：如"OCT"和"光学相干断层扫描" → 归并为"光学相干断层扫描（OCT）"
4. **不同粒度**：如"青光眼"和"开角型青光眼" → **不归并**，用 SUBCLASS_OF 关系连接
5. **非同类实体**：如"角膜"（Anatomy）和"角膜炎"（Disease） → **不归并**

## 输入实体列表

{entities_json}

## 输出格式

```json
{
  "merged_entities": [
    {
      "canonical_name": "标准名称",
      "type": "实体类型",
      "aliases": ["别名1", "别名2"],
      "source_entries": [
        {"original_name": "原始名称1", "source": "来源"},
        {"original_name": "原始名称2", "source": "来源"}
      ]
    }
  ],
  "unmerged_entities": [
    {
      "name": "实体名称",
      "type": "实体类型",
      "reason": "不归并的原因（不同粒度/不同概念）"
    }
  ]
}
```
````

---

## 后处理规则（代码层面）

除了 Prompt 模板外，以下规则应在代码层面自动执行：

```python
"""
后处理规则 —— 在 LLM 抽取结果返回后自动应用
"""

def post_process_triplets(triplets: list[dict]) -> list[dict]:
    """对 LLM 抽取结果进行自动后处理"""

    processed = []

    for t in triplets:
        # 规则 1：验证实体类型
        valid_types = {"Anatomy", "Disease", "Symptom", "Examination",
                       "Stage", "Treatment", "Drug", "RiskFactor"}
        if t["head"]["type"] not in valid_types or t["tail"]["type"] not in valid_types:
            continue  # 跳过无效实体类型的三元组

        # 规则 2：验证关系类型
        valid_relations = {
            "HAS_SYMPTOM", "LOCATED_IN", "DIAGNOSED_BY", "TREATED_BY",
            "CAUSED_BY", "INDICATES", "CONTRAINDICATED", "PROGRESSES_TO",
            "HAS_STAGE", "SUBCLASS_OF", "HAS_RISK_FACTOR", "COMPLICATED_BY"
        }
        if t["relation"] not in valid_relations:
            continue

        # 规则 3：检查 head == tail 的自环
        if t["head"]["name"] == t["tail"]["name"]:
            continue

        # 规则 4：domain-range 约束检查
        domain_range = {
            "HAS_SYMPTOM": ("Disease", "Symptom"),
            "DIAGNOSED_BY": ("Disease", "Examination"),
            "TREATED_BY": ("Disease", "Treatment"),
            "INDICATES": ("Symptom", "Examination"),
            "SUBCLASS_OF": ("Disease", "Disease"),
            "LOCATED_IN": ("Anatomy", "Anatomy"),
        }
        if t["relation"] in domain_range:
            expected_domain, expected_range = domain_range[t["relation"]]
            if not (t["head"]["type"].startswith(expected_domain)
                    and t["tail"]["type"].startswith(expected_range)):
                t["confidence"] = "low"  # 降级而非丢弃

        # 规则 5：去除重复三元组
        key = (t["head"]["name"], t["relation"], t["tail"]["name"])
        if not hasattr(post_process_triplets, '_seen'):
            post_process_triplets._seen = set()
        if key in post_process_triplets._seen:
            continue
        post_process_triplets._seen.add(key)

        # 规则 6：确保必填字段存在
        t.setdefault("evidence", "")
        t.setdefault("confidence", "medium")
        t.setdefault("attributes", {})

        processed.append(t)

    return processed
```
