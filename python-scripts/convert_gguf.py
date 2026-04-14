# 克隆最新 llama.cpp（必须 ≥ v3.0）
# it clone https://github.com/ggerganov/llama.cpp
# cd llama.cpp && make clean && make -j LLAMA_METAL=1

# 转 FP16 GGUF
# python convert_hf_to_gguf.py ./antangelmed-fp16 --outfile antangelmed-f16.gguf --fp16

# 量化为 Q4_K_M（最佳平衡）
# ./quantize antangelmed-f16.gguf antangelmed-q4_k_m.gguf q4_k_m