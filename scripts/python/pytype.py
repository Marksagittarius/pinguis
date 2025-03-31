import os
import subprocess
import argparse
import sys

def run_pytype(target):
    print(f"Running pytype on: {target}")
    try:
        subprocess.run(["pytype", target], check=True)
        return True
    except subprocess.CalledProcessError as e:
        print(f"Error running pytype: {e}", file=sys.stderr)
        return False

def merge_type_info(target):
    pytype_pyi_dir = os.path.join(".pytype", "pyi")
    
    if not os.path.exists(pytype_pyi_dir):
        print(f"Warning: {pytype_pyi_dir} directory not found")
        return
    
    if os.path.isfile(target) and target.endswith(".py"):
        files = [target]
    else:
        files = [
            os.path.join(root, file)
            for root, _, files in os.walk(target)
            for file in files if file.endswith(".py")
        ]
    
    success_count = 0
    failed_count = 0
    
    for py_file in files:
        rel_path = os.path.relpath(py_file)
        pyi_file = os.path.join(pytype_pyi_dir, rel_path + "i")
        
        if os.path.exists(pyi_file):
            print(f"Merging type info for: {rel_path}")
            try:
                subprocess.run(["merge-pyi", "-i", py_file, pyi_file], check=True)
                success_count += 1
            except subprocess.CalledProcessError as e:
                print(f"Error merging {py_file}: {e}", file=sys.stderr)
                failed_count += 1
    
    print(f"Merged type info for {success_count} files ({failed_count} failed)")

def clean_pytype():
    """Remove the .pytype directory."""
    if os.path.exists(".pytype"):
        print("Cleaning up .pytype directory")
        subprocess.run(["rm", "-rf", ".pytype"], check=True)

def main():
    parser = argparse.ArgumentParser(description="Automatically add type annotations using pytype")
    parser.add_argument("target", help="Python file or directory to process")
    parser.add_argument("--keep", action="store_true", help="Keep .pytype directory after completion")
    parser.add_argument("-v", "--verbose", action="store_true", help="Show verbose output")
    
    args = parser.parse_args()
    
    if not os.path.exists(args.target):
        print(f"Error: {args.target} not found", file=sys.stderr)
        return 1
    
    if not run_pytype(args.target):
        return 1
    
    merge_type_info(args.target)
    
    if not args.keep:
        clean_pytype()
    
    return 0

if __name__ == "__main__":
    sys.exit(main())
