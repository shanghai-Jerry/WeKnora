# pip install mlx_lm==0.31.1
python -m mlx_lm.convert --hf-path Octen/Octen-Embedding-4B --mlx-path ./octen-embedding-mlx --quantize --q-bits 4
