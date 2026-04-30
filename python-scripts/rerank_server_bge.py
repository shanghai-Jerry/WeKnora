import torch
import gc
import asyncio
import uvicorn
from fastapi import FastAPI
from pydantic import BaseModel
from transformers import AutoModelForSequenceClassification, AutoTokenizer
from typing import List

import os
os.environ['HF_ENDPOINT'] = 'https://hf-mirror.com'
# --- 1. 定义API的请求和响应数据结构 ---

# 请求体结构保持不变
class RerankRequest(BaseModel):
    query: str
    documents: List[str]

# --- 修改开始：定义测试用的响应结构，字段名为 "score" ---

# DocumentInfo 结构保持不变
class DocumentInfo(BaseModel):
    text: str

# 将原来的 GoRankResult 修改为 TestRankResult
# 核心改动：将 "relevance_score" 字段重命名为 "score"
class TestRankResult(BaseModel):
    index: int
    document: DocumentInfo
    score: float  # <--- 【关键修改点】字段名已从 relevance_score 改为 score

# 最终响应体结构，其 "results" 列表包含的是 TestRankResult
class TestFinalResponse(BaseModel):
    results: List[TestRankResult]

# --- 修改结束 ---

# --- 2. 加载模型 (在服务启动时执行一次) ---
print("正在加载模型，请稍候...")
if torch.cuda.is_available():
    device = torch.device("cuda")
elif torch.backends.mps.is_available():
    device = torch.device("mps")
else:
    device = torch.device("cpu")
print(f"使用的设备: {device}")
try:
    # 请确保这里的路径是正确的
    model_path = 'BAAI/bge-reranker-v2-m3'
    # 直接指定模型类和分词器类
    from transformers import XLMRobertaTokenizer, XLMRobertaForSequenceClassification
    tokenizer = XLMRobertaTokenizer.from_pretrained(model_path)
    model = XLMRobertaForSequenceClassification.from_pretrained(model_path)
    model.to(device)
    model.eval()
    print("模型加载成功！")
except Exception as e:
    print(f"模型加载失败: {e}")
    # 在测试环境中，如果模型加载失败，可以考虑退出以避免运行一个无效的服务
    exit()

# --- 3. 创建FastAPI应用 ---
app = FastAPI(
    title="Reranker API (Test Version)",
    description="一个返回 'score' 字段以测试Go客户端兼容性的API服务",
    version="1.0.1"
)

# 定期内存清理任务（每5分钟）
async def periodic_memory_cleanup():
    """定期清理内存，避免频繁调用影响性能"""
    while True:
        await asyncio.sleep(300)  # 5分钟
        if device.type == 'mps':
            torch.mps.empty_cache()
            print("定期清理: MPS缓存已清理")
        gc.collect()
        print("定期清理: Python垃圾回收已完成")

@app.on_event("startup")
async def startup_event():
    asyncio.create_task(periodic_memory_cleanup())

# 内存监控端点
@app.get("/memory")
def memory_status():
    status = {"device": str(device)}
    if device.type == 'mps' and hasattr(torch.mps, 'current_allocated_memory'):
        status["mps_allocated_mb"] = torch.mps.current_allocated_memory() / 1024**2
    return status

# --- 4. 定义API端点 ---
@app.post("/rerank", response_model=TestFinalResponse)
def rerank_endpoint(request: RerankRequest):
    # 轻量级处理：仅删除引用，不调用empty_cache和gc
    pairs = [[request.query, doc] for doc in request.documents]
    
    with torch.no_grad():
        inputs = tokenizer(pairs, padding=True, truncation=True, return_tensors='pt', max_length=1024).to(device)
        scores = model(**inputs, return_dict=True).logits.view(-1, ).float()
        # 立即删除inputs释放引用
        del inputs
    
    results = []
    for i, (text, score_val) in enumerate(zip(request.documents, scores)):
        doc_info = DocumentInfo(text=text)
        test_result = TestRankResult(
            index=i,
            document=doc_info,
            score=score_val.item()
        )
        results.append(test_result)
    
    # 删除scores引用
    del scores
    
    sorted_results = sorted(results, key=lambda x: x.score, reverse=True)
    return {"results": sorted_results}

@app.get("/")
def read_root():
    return {"status": "Reranker API (Test Version) is running"}

# --- 5. 启动服务 ---
if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
    