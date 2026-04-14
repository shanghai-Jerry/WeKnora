from PyPDF2 import PdfReader, PdfWriter
import os
import argparse

SIZE_LIMIT_MB = 50
SIZE_LIMIT_BYTES = SIZE_LIMIT_MB * 1024 * 1024
PAGES_PER_FILE = 100


def split_pdf(input_path, output_dir, pages_per_file=PAGES_PER_FILE):
    reader = PdfReader(input_path)
    total_pages = len(reader.pages)

    os.makedirs(output_dir, exist_ok=True)

    base_name = os.path.splitext(os.path.basename(input_path))[0]

    split_count = 0
    for i in range(0, total_pages, pages_per_file):
        writer = PdfWriter()
        start = i
        end = min(i + pages_per_file, total_pages)

        for page_num in range(start, end):
            writer.add_page(reader.pages[page_num])

        output_path = os.path.join(output_dir, f"{base_name}_{start+1:04d}-{end:04d}.pdf")
        with open(output_path, "wb") as f:
            writer.write(f)
        print(f"Saved: {output_path}")
        split_count += 1

    return split_count


def process_directory(input_dir, output_dir):
    if not os.path.isdir(input_dir):
        print(f"Error: {input_dir} is not a valid directory")
        return

    os.makedirs(output_dir, exist_ok=True)

    pdf_files = [f for f in os.listdir(input_dir) if f.lower().endswith('.pdf')]

    if not pdf_files:
        print(f"No PDF files found in {input_dir}")
        return

    print(f"Found {len(pdf_files)} PDF files in {input_dir}")

    processed_count = 0
    skipped_count = 0

    for pdf_file in pdf_files:
        input_path = os.path.join(input_dir, pdf_file)
        file_size = os.path.getsize(input_path)
        file_size_mb = file_size / (1024 * 1024)

        print(f"\nChecking: {pdf_file} ({file_size_mb:.2f} MB)")

        if file_size > SIZE_LIMIT_BYTES:
            print(f"  File exceeds {SIZE_LIMIT_MB}MB, splitting...")
            split_count = split_pdf(input_path, output_dir)
            print(f"  Split into {split_count} files")
            processed_count += 1
        else:
            print(f"  File size is under {SIZE_LIMIT_MB}MB, skipping")
            skipped_count += 1

    print(f"\n=== Summary ===")
    print(f"Total PDF files: {len(pdf_files)}")
    print(f"Split: {processed_count}")
    print(f"Skipped: {skipped_count}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Split large PDF files (>50MB) into smaller files (50 pages each)"
    )
    parser.add_argument(
        "-i", "--input",
        required=True,
        help="Input directory containing PDF files"
    )
    parser.add_argument(
        "-o", "--output",
        required=True,
        help="Output directory for split PDF files"
    )
    parser.add_argument(
        "-s", "--size-limit",
        type=int,
        default=SIZE_LIMIT_MB,
        help=f"Size limit in MB (default: {SIZE_LIMIT_MB})"
    )
    parser.add_argument(
        "-p", "--pages",
        type=int,
        default=PAGES_PER_FILE,
        help=f"Pages per split file (default: {PAGES_PER_FILE})"
    )

    args = parser.parse_args()

    SIZE_LIMIT_BYTES = args.size_limit * 1024 * 1024

    if args.pages != PAGES_PER_FILE:
        PAGES_PER_FILE = args.pages

    process_directory(args.input, args.output)
