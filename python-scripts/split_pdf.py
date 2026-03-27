from PyPDF2 import PdfReader, PdfWriter
import os

def split_pdf(input_path, output_dir, pages_per_file=100):
    reader = PdfReader(input_path)
    total_pages = len(reader.pages)
    
    os.makedirs(output_dir, exist_ok=True)
    
    for i in range(0, total_pages, pages_per_file):
        writer = PdfWriter()
        start = i
        end = min(i + pages_per_file, total_pages)
        
        for page_num in range(start, end):
            writer.add_page(reader.pages[page_num])
        
        output_path = os.path.join(output_dir, f"part_{i+1:04d}-{end:04d}.pdf")
        with open(output_path, "wb") as f:
            writer.write(f)
        print(f"Saved: {output_path}")

# 使用示例：每 100 页一个文件
split_pdf("/Users/edy/Downloads/接触镜学.pdf", "./out-pdf", pages_per_file=100)