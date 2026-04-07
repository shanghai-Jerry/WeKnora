# 07 - 使用指南

> **用途**：本文档提供眼科视光知识图谱 Prompt 模板体系的完整使用指南，包括模板选择决策树、API 调用示例、Neo4j 导入脚本和 Prompt 集成策略。

---

## 一、快速开始

### 1.1 模板选择决策树

```
你的数据是什么类型？
│
├─ 临床指南/专家共识
│  └─ → 使用 05-source-specific.md 中的「模板 5.1：临床指南抽取模板」
│
├─ 医学教材
│  └─ → 使用 05-source-specific.md 中的「模板 5.2：医学教材抽取模板」
│
├─ 科研文献摘要
│  └─ → 使用 05-source-specific.md 中的「模板 5.3：PubMed 文献抽取模板」
│     或 04-joint-triplet-extraction.md 中的「模板 4.2：文献摘要联合抽取」
│
├─ 药品说明书
│  └─ → 使用 05-source-specific.md 中的「模板 5.4：药品说明书抽取模板」
│     或 04-joint-triplet-extraction.md 中的「模板 4.3：药品说明书联合抽取」
│
├─ 电子病历
│  └─ → 使用 05-source-specific.md 中的「模板 5.5：电子病历抽取模板」
│
└─ 其他/不确定
   └─ → 使用 04-joint-triplet-extraction.md 中的「模板 4.1：通用联合抽取模板」
```

**需要更高精度？** → 使用 01-entity-recognition.md + 02-relation-extraction.md 的分步抽取 + 多 Prompt 集成投票。

---

## 二、API 调用示例

### 2.1 基础调用（单模板）

```python
"""
眼科知识图谱抽取 - 基础调用示例
"""
import json
from openai import OpenAI

client = OpenAI()  # 需设置 OPENAI_API_KEY 环境变量


def extract_from_guideline(text: str) -> dict:
    """从临床指南文本中抽取知识三元组"""

    # System Prompt（来自 00-schema-definition.md）
    system_prompt = """你是一名眼科视光领域的医学知识抽取专家。
核心原则：忠实原文、标准术语、粒度适中、不确定标记、证据溯源。
严格按 JSON 格式输出。"""

    # User Prompt（来自 05-source-specific.md 模板 5.1）
    user_prompt = f"""## 任务
你正在处理一份眼科临床指南/专家共识文本。

## 抽取重点
1. 诊断标准 → DIAGNOSED_BY 关系
2. 分期分级 → HAS_STAGE / PROGRESSES_TO 关系
3. 治疗推荐（含循证等级）→ TREATED_BY 关系

## 指南文本
{text}

## 输出格式
严格按 JSON 格式输出三元组数组，每个三元组包含 head、relation、tail、evidence、confidence。"""

    response = client.chat.completions.create(
        model="gpt-4",  # 或 "deepseek-chat"
        messages=[
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": user_prompt}
        ],
        temperature=0.1,
        response_format={"type": "json_object"}
    )

    return json.loads(response.choices[0].message.content)


# 使用示例
guideline_text = """
干眼症诊疗专家共识（2024）
...
"""

result = extract_from_guideline(guideline_text)
print(json.dumps(result, ensure_ascii=False, indent=2))
```

### 2.2 多 Prompt 集成调用

```python
"""
眼科知识图谱抽取 - 多模板集成调用示例
使用多个异构模板并行抽取 + 投票融合
"""
import json
from openai import OpenAI
from concurrent.futures import ThreadPoolExecutor

client = OpenAI()

# 模板注册表
TEMPLATES = {
    "instruction": """从以下眼科文本中识别所有医学实体并分类为8种类型。输出JSON数组。
实体类型：Anatomy, Disease, Symptom, Examination, Stage, Treatment, Drug, RiskFactor

文本：{text}

输出格式：{{"entities": [{{"name": "名称", "type": "类型"}}]}}
""",

    "few_shot": """参考示例从眼科文本中识别实体：

示例：患者主诉眼痛、畏光，诊断为角膜炎。
输出：{{"entities": [{{"name":"眼痛","type":"Symptom"}},{{"name":"畏光","type":"Symptom"}},{{"name":"角膜炎","type":"Disease"}}]}}

文本：{text}

输出格式：{{"entities": [{{"name": "名称", "type": "类型"}}]}}
""",

    "role_play": """你是一位资深眼科主任医师。从临床文本中精确提取医学实体。
注意区分疾病和症状，精确分期，识别否定表述。

临床文本：{text}

输出格式：{{"entities": [{{"name": "名称", "type": "类型", "clinical_context": "语境"}}]}}
""",

    "constrained": """对以下眼科文本进行精细化实体识别，处理嵌套实体和缩写消歧。
每个实体必须能在原文中找到对应片段。

文本：{text}

输出格式：{{"entities": [{{"name": "名称", "type": "类型", "span": "原文片段", "confidence": "high|medium|low"}}]}}
"""
}


def call_template(template_name: str, text: str) -> dict:
    """调用单个模板"""
    prompt = TEMPLATES[template_name].format(text=text)

    response = client.chat.completions.create(
        model="gpt-4",
        messages=[
            {"role": "system", "content": "你是眼科知识抽取专家。严格按JSON输出。"},
            {"role": "user", "content": prompt}
        ],
        temperature=0.1,
        response_format={"type": "json_object"}
    )

    return json.loads(response.choices[0].message.content)


def ensemble_extract(text: str) -> dict:
    """多模板集成抽取"""

    # 并行调用所有模板
    with ThreadPoolExecutor(max_workers=4) as executor:
        futures = {
            name: executor.submit(call_template, name, text)
            for name in TEMPLATES
        }

    results = {
        name: future.result()
        for name, future in futures.items()
    }

    # 实体投票统计
    entity_votes = {}
    entity_details = {}

    for template_name, result in results.items():
        for entity in result.get("entities", []):
            key = (entity["name"], entity["type"])
            entity_votes[key] = entity_votes.get(key, 0) + 1
            if key not in entity_details:
                entity_details[key] = {"name": entity["name"], "type": entity["type"]}
                # 合并所有属性
            for attr in ["span", "clinical_context", "confidence", "aliases"]:
                if attr in entity and attr not in entity_details[key]:
                    entity_details[key][attr] = entity[attr]

    # 生成最终结果
    final_entities = []
    for (name, etype), votes in entity_votes.items():
        detail = entity_details[(name, etype)]
        confidence = "high" if votes >= 3 else ("medium" if votes >= 2 else "low")
        final_entities.append({
            **detail,
            "confidence": confidence,
            "vote_count": votes,
            "voted_by": [k for k, v in results.items()
                        if any(e["name"] == name and e["type"] == etype
                               for e in v.get("entities", []))]
        })

    return {
        "final_entities": sorted(final_entities, key=lambda x: x["vote_count"], reverse=True),
        "per_template_results": results,
        "ensemble_stats": {
            "total_unique_entities": len(entity_votes),
            "templates_used": len(TEMPLATES),
            "consensus_entities": sum(1 for v in entity_votes.values() if v >= 3)
        }
    }


# 使用示例
text = "患者右眼视力下降1月，OCT示黄斑区视网膜水肿，诊断为湿性年龄相关性黄斑变性。"
result = ensemble_extract(text)
print(json.dumps(result, ensure_ascii=False, indent=2))
```

### 2.3 完整 Pipeline

```python
"""
完整知识抽取 Pipeline
输入：原始文本 + 数据源类型
输出：质量校验后的三元组列表（可直接导入 Neo4j）
"""
import json
from openai import OpenAI

client = OpenAI()


def full_extraction_pipeline(text: str, source_type: str) -> dict:
    """
    完整抽取流水线：
    1. 联合抽取（实体 + 关系 + 属性）
    2. 质量自检
    3. 后处理（去重、约束检查）
    4. 返回可导入的结果
    """

    # === Step 1: 联合抽取 ===
    extraction_prompt = f"""从以下眼科文本中抽取知识三元组。

实体类型：Anatomy, Disease, Symptom, Examination, Stage, Treatment, Drug, RiskFactor
关系类型：HAS_SYMPTOM, LOCATED_IN, DIAGNOSED_BY, TREATED_BY, CAUSED_BY,
         HAS_RISK_FACTOR, COMPLICATED_BY, CONTRAINDICATED, PROGRESSES_TO,
         HAS_STAGE, SUBCLASS_OF, INDICATES

文本：
{text}

输出格式：
{{"triplets": [{{"head": {{"name": "名称", "type": "类型"}}, "relation": "关系", "tail": {{"name": "名称", "type": "类型"}}, "evidence": "原文证据", "confidence": "high|medium|low", "attributes": {{}}}}]}}"""

    response = client.chat.completions.create(
        model="gpt-4",
        messages=[
            {"role": "system", "content": "你是眼科知识抽取专家。忠实原文，严格JSON输出。"},
            {"role": "user", "content": extraction_prompt}
        ],
        temperature=0.1,
        response_format={"type": "json_object"}
    )

    raw_result = json.loads(response.choices[0].message.content)
    triplets = raw_result.get("triplets", [])

    # === Step 2: 后处理 ===
    triplets = post_process_triplets(triplets)

    # === Step 3: 添加元数据 ===
    final_result = {
        "triplets": triplets,
        "metadata": {
            "source_type": source_type,
            "model": "gpt-4",
            "pipeline": "full_extraction",
            "timestamp": datetime.now().isoformat()
        }
    }

    return final_result


def post_process_triplets(triplets: list[dict]) -> list[dict]:
    """后处理：约束检查 + 去重"""
    valid_types = {"Anatomy", "Disease", "Symptom", "Examination",
                   "Stage", "Treatment", "Drug", "RiskFactor"}
    valid_relations = {"HAS_SYMPTOM", "LOCATED_IN", "DIAGNOSED_BY", "TREATED_BY",
                       "CAUSED_BY", "INDICATES", "CONTRAINDICATED", "PROGRESSES_TO",
                       "HAS_STAGE", "SUBCLASS_OF", "HAS_RISK_FACTOR", "COMPLICATED_BY"}

    seen = set()
    processed = []

    for t in triplets:
        # 跳过无效类型
        if (t["head"]["type"] not in valid_types
                or t["tail"]["type"] not in valid_types
                or t["relation"] not in valid_relations):
            continue

        # 跳过自环
        if t["head"]["name"] == t["tail"]["name"]:
            continue

        # 去重
        key = (t["head"]["name"], t["relation"], t["tail"]["name"])
        if key in seen:
            continue
        seen.add(key)

        # 确保必填字段
        t.setdefault("evidence", "")
        t.setdefault("confidence", "medium")
        t.setdefault("attributes", {})

        processed.append(t)

    return processed
```

---

## 三、Neo4j 导入脚本

### 3.1 创建图模式

```cypher
// 创建节点标签和属性约束
CREATE CONSTRAINT IF NOT EXISTS FOR (d:Disease) REQUIRE d.name IS UNIQUE;
CREATE CONSTRAINT IF NOT EXISTS FOR (a:Anatomy) REQUIRE a.name IS UNIQUE;
CREATE CONSTRAINT IF NOT EXISTS FOR (s:Symptom) REQUIRE s.name IS UNIQUE;
CREATE CONSTRAINT IF NOT EXISTS FOR (e:Examination) REQUIRE e.name IS UNIQUE;
CREATE CONSTRAINT IF NOT EXISTS FOR (st:Stage) REQUIRE st.name IS UNIQUE;
CREATE CONSTRAINT IF NOT EXISTS FOR (t:Treatment) REQUIRE t.name IS UNIQUE;
CREATE CONSTRAINT IF NOT EXISTS FOR (dr:Drug) REQUIRE dr.name IS UNIQUE;
CREATE CONSTRAINT IF NOT EXISTS FOR (r:RiskFactor) REQUIRE r.name IS UNIQUE;

// 创建关系类型（Neo4j 会自动创建，这里仅作文档说明）
// HAS_SYMPTOM, LOCATED_IN, DIAGNOSED_BY, TREATED_BY,
// CAUSED_BY, HAS_RISK_FACTOR, COMPLICATED_BY, CONTRAINDICATED,
// PROGRESSES_TO, HAS_STAGE, SUBCLASS_OF, INDICATES
```

### 3.2 Python 导入脚本

```python
"""
将抽取结果导入 Neo4j
"""
from neo4j import GraphDatabase


class OphthalmologyKGImporter:
    def __init__(self, uri: str, user: str, password: str):
        self.driver = GraphDatabase.driver(uri, auth=(user, password))

    def close(self):
        self.driver.close()

    def import_triplets(self, extraction_result: dict):
        """导入抽取结果到 Neo4j"""

        triplets = extraction_result["triplets"]
        metadata = extraction_result["metadata"]

        with self.driver.session() as session:
            for triplet in triplets:
                session.execute_write(
                    self._create_triplet,
                    triplet,
                    metadata
                )

    @staticmethod
    def _create_triplet(tx, triplet: dict, metadata: dict):
        head = triplet["head"]
        relation = triplet["relation"]
        tail = triplet["tail"]
        evidence = triplet.get("evidence", "")
        confidence = triplet.get("confidence", "medium")
        attributes = triplet.get("attributes", {})

        # 创建 head 节点
        tx.run(f"""
            MERGE (h:{head['type']} {{name: $name}})
            SET h.confidence = $confidence,
                h.source_type = $source_type
            """,
            name=head["name"],
            confidence=confidence,
            source_type=metadata.get("source_type", "")
        )

        # 创建 tail 节点
        tx.run(f"""
            MERGE (t:{tail['type']} {{name: $name}})
            SET t.confidence = $confidence,
                t.source_type = $source_type
            """,
            name=tail["name"],
            confidence=confidence,
            source_type=metadata.get("source_type", "")
        )

        # 创建关系
        attr_str = ""
        if attributes:
            attr_pairs = [f"r.{k} = ${k}" for k in attributes.keys()]
            attr_str = ", " + ", ".join(attr_pairs)

        query = f"""
            MATCH (h:{head['type']} {{name: $head_name}})
            MATCH (t:{tail['type']} {{name: $tail_name}})
            MERGE (h)-[r:{relation}]->(t)
            SET r.evidence = $evidence,
                r.confidence = $confidence,
                r.source_type = $source_type,
                r.timestamp = $timestamp
                {attr_str}
        """

        params = {
            "head_name": head["name"],
            "tail_name": tail["name"],
            "evidence": evidence,
            "confidence": confidence,
            "source_type": metadata.get("source_type", ""),
            "timestamp": metadata.get("timestamp", ""),
            **attributes
        }

        tx.run(query, **params)

    def import_drug_attributes(self, drug_result: dict):
        """导入药物属性（来自药品说明书抽取）"""

        drug = drug_result.get("drug_entity", {})
        attrs = drug_result.get("drug_attributes", {})

        if not drug:
            return

        with self.driver.session() as session:
            # 更新药物节点属性
            attr_pairs = []
            params = {"name": drug["name"]}
            for k, v in attrs.items():
                if isinstance(v, (str, int, float, bool)):
                    params[k] = v
                    attr_pairs.append(f"d.{k} = ${k}")
                elif isinstance(v, list):
                    params[k] = v
                    attr_pairs.append(f"d.{k} = ${k}")

            if attr_pairs:
                attr_str = ", ".join(attr_pairs)
                session.run(f"""
                    MERGE (d:Drug {{name: $name}})
                    SET {attr_str}
                """, **params)

            # 导入药物关系三元组
            for triplet in drug_result.get("triplets", []):
                session.execute_write(self._create_triplet, triplet,
                                      {"source_type": "drug_label"})


# 使用示例
if __name__ == "__main__":
    importer = OphthalmologyKGImporter(
        uri="bolt://localhost:7687",
        user="neo4j",
        password="your_password"
    )

    # 读取抽取结果
    with open("extraction_result.json", "r") as f:
        result = json.load(f)

    # 导入
    importer.import_triplets(result)
    importer.close()

    print(f"成功导入 {len(result['triplets'])} 个三元组")
```

---

## 四、Prompt 集成策略说明

### 4.1 策略选择矩阵

| 场景 | 推荐策略 | 原因 |
|------|---------|------|
| MVP/快速原型 | 单模板（联合抽取） | 成本低，速度快 |
| 高精度要求 | 多模板集成（4模板投票） | 降低幻觉和遗漏 |
| 生产环境 | 分步抽取（NER → RE → Attr） | 可控性强，便于调试 |
| 大规模批处理 | 单模板 + 后处理规则 | 平衡精度和成本 |

### 4.2 成本优化建议

| 优化手段 | 效果 | 适用场景 |
|---------|------|---------|
| 使用 GPT-4o-mini 替代 GPT-4 | 成本降低 90% | 低精度要求场景 |
| Few-shot 示例数量控制在 3 个 | Token 数减少 30% | 所有场景 |
| 文本预分割（每段 < 2000 字） | 避免超长文本，提高抽取质量 | 长文档 |
| 批量 API 调用（batch endpoint） | 成本降低 50% | 大规模离线处理 |
| 缓存相同文本的结果 | 避免重复调用 | 有重复数据的场景 |

### 4.3 评估指标

建议定期对抽取结果进行人工评估，关注以下指标：

| 指标 | 计算方式 | 目标值 |
|------|---------|--------|
| 实体识别 Precision | 正确实体数 / 抽取实体总数 | > 90% |
| 实体识别 Recall | 正确实体数 / 文本中实体总数 | > 85% |
| 关系抽取 Precision | 正确三元组数 / 抽取三元组总数 | > 85% |
| 关系抽取 Recall | 正确三元组数 / 应抽取三元组总数 | > 80% |
| 幻觉率 | 编造三元组数 / 总抽取数 | < 5% |
| 证据覆盖率 | 有 evidence 的三元组 / 总三元组 | > 95% |

---

## 五、目录索引

| 文件 | 内容 | 适用场景 |
|------|------|---------|
| `00-schema-definition.md` | 本体 Schema（实体/关系/属性/术语词典） | **所有模板的前置依赖** |
| `01-entity-recognition.md` | 4种 NER 模板 + 集成策略 | 纯实体识别任务 |
| `02-relation-extraction.md` | 6种关系抽取模板 | 纯关系抽取任务 |
| `03-attribute-extraction.md` | 3种属性抽取模板 | 纯属性填充任务 |
| `04-joint-triplet-extraction.md` | 3种联合抽取模板 + Pipeline | 端到端抽取 |
| `05-source-specific.md` | 5种数据源专用模板 | 特定数据源处理 |
| `06-quality-control.md` | 4种 QC 模板 + 后处理规则 | 质量保障 |
| `07-usage-guide.md` | 本文档 | 使用参考 |
| `examples/` | 完整输入输出示例 | 效果参考和调试 |
