# dequantize_to_hf.py
from awq import AutoAWQForCausalLM
from transformers import AutoTokenizer

model_path = "./antangelmed-int4"
output_path = "./antangelmed-fp16"

# 加载 AWQ 模型
model = AutoAWQForCausalLM.from_quantized(model_path, device="cpu")
tokenizer = AutoTokenizer.from_pretrained(model_path, trust_remote_code=True)

# 保存为标准 HF FP16 格式
model.save_pretrained(output_path, safe_serialization=True)
tokenizer.save_pretrained(output_path)