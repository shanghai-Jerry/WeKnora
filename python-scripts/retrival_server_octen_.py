import torch
import uvicorn
from fastapi import FastAPI
from pydantic import BaseModel, Field
from typing import List, Optional, Union
from transformers import AutoModel, AutoTokenizer
import torch.nn.functional as F
import time

# --- 1. 定义 API 的请求和响应数据结构 ---

class EmbeddingRequest(BaseModel):
    model: str = "Octen-Embedding-4B"
    input: Union[str, List[str]]
    encoding_format: Optional[str] = "float"

class EmbeddingData(BaseModel):
    object: str = "embedding"
    embedding: List[float]
    index: int

class UsageInfo(BaseModel):
    prompt_tokens: int
    total_tokens: int

class EmbeddingResponse(BaseModel):
    object: str = "list"
    data: List[EmbeddingData]
    model: str
    usage: UsageInfo

# --- 2. 加载模型 ---
print("正在加载模型，请稍候...")
device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
print(f"使用的设备: {device}")

model_name = "Octen/Octen-Embedding-4B"
tokenizer = AutoTokenizer.from_pretrained(model_name, padding_side="left")
model = AutoModel.from_pretrained(model_name)
model.to(device)
model.eval()
print("模型加载成功！")

def count_tokens(texts: List[str]) -> int:
    """计算 token 数量"""
    total = 0
    for text in texts:
        tokens = tokenizer.encode(text, add_special_tokens=False)
        total += len(tokens)
    return total

def encode(texts: List[str]) -> torch.Tensor:
    """生成文本嵌入向量"""
    inputs = tokenizer(texts, padding=True, truncation=True,
                       max_length=8192, return_tensors="pt").to(device)

    with torch.no_grad():
        outputs = model(**inputs)
        embeddings = outputs.last_hidden_state[:, -1, :]
        embeddings = F.normalize(embeddings, p=2, dim=1)

    return embeddings

# --- 3. 创建 FastAPI 应用 ---
app = FastAPI(
    title="Embedding API Server",
    description="OpenAI 兼容的 Embedding API 服务",
    version="1.0.0"
)

# --- 4. 定义 API 端点 ---
@app.post("/embeddings", response_model=EmbeddingResponse)
def embeddings_endpoint(request: EmbeddingRequest):
    input_texts = request.input if isinstance(request.input, list) else [request.input]
    
    embeddings = encode(input_texts)
    
    prompt_tokens = count_tokens(input_texts)
    
    data = []
    for i, embedding in enumerate(embeddings):
        data.append(EmbeddingData(
            object="embedding",
            embedding=embedding.cpu().tolist(),
            index=i
        ))
    
    return EmbeddingResponse(
        object="list",
        data=data,
        model=request.model,
        usage=UsageInfo(
            prompt_tokens=prompt_tokens,
            total_tokens=prompt_tokens
        )
    )

@app.get("/")
def read_root():
    return {
        "status": "Embedding API Server is running",
        "model": model_name,
        "device": str(device)
    }

@app.get("/health")
def health_check():
    return {"status": "healthy"}

# --- 5. 启动服务 ---
if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8889)
