import ast
import os
import sys
import json
import argparse
from typing import Dict, List, Any, Optional, Union, Set


class PythonAnalyzer:
    def __init__(self, root_path: str = None):
        """
        Initialize the analyzer with an optional root path.
        
        Args:
            root_path: The root directory to use when calculating Python module paths.
                      If None, the current working directory is used.
        """
        self.modules = {}
        self.current_file = None
        self.current_module = None
        self.root_path = os.path.abspath(root_path) if root_path else os.getcwd()
        print(f"Using root path: {self.root_path}")

    def analyze_path(self, path: str) -> Dict:
        is_single_file = os.path.isfile(path) and path.endswith('.py')
        
        if is_single_file:
            file_data = self._process_file_and_return(path)
            return file_data
        elif os.path.isdir(path):
            self._process_directory(path)
            
            dir_name = os.path.basename(os.path.normpath(path))
            
            all_files = []
            for module_files in self.modules.values():
                all_files.extend(module_files)
            
            result = {
                "name": dir_name,
                "files": all_files
            }
            
            return result
        else:
            print(f"Error: {path} is not a Python file or directory")
            sys.exit(1)    

    def _get_module_name(self, file_path: str) -> str:
        """
        Calculate the Python module name of a file relative to the root path.
        
        Args:
            file_path: The absolute path to the Python file
            
        Returns:
            The Python module name (e.g. 'package.subpackage.module')
        """
        abs_path = os.path.abspath(file_path)
        
        if abs_path.startswith(self.root_path):
            rel_path = os.path.relpath(abs_path, self.root_path)
        else:
            rel_path = os.path.relpath(abs_path)
        
        directory_path = os.path.dirname(rel_path)
        file_name = os.path.splitext(os.path.basename(rel_path))[0]
        
        if file_name == '__init__':
            return directory_path.replace(os.path.sep, '.')
        else:
            if directory_path:
                return f"{directory_path.replace(os.path.sep, '.')}.{file_name}"
            else:
                return file_name

    def _get_relative_path(self, file_path: str) -> str:
        """
        Get the path relative to the root path.
        
        Args:
            file_path: The absolute path to the file
            
        Returns:
            The path relative to the root path
        """
        abs_path = os.path.abspath(file_path)
        if abs_path.startswith(self.root_path):
            return os.path.relpath(abs_path, self.root_path)
        else:
            return os.path.basename(file_path)

    def _process_file_and_return(self, file_path: str) -> Dict:
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                source_code = f.read()
            
            tree = ast.parse(source_code)
            
            module_name = self._get_module_name(file_path)
            
            self.current_file = file_path
            self.current_module = module_name
            
            file_data = {
                "path": self._get_relative_path(file_path),
                "module": module_name,
                "classes": [],
                "interfaces": [],
                "functions": []
            }
            
            for node in tree.body:
                if isinstance(node, ast.FunctionDef):
                    function = self._extract_function(node)
                    file_data["functions"].append(function)
            
            for node in tree.body:
                if isinstance(node, ast.ClassDef):
                    class_data = self._extract_class(node)
                    
                    if self._is_interface(node):
                        interface = {
                            "name": class_data["name"],
                            "methods": [m["function"] for m in class_data["methods"]]
                        }
                        file_data["interfaces"].append(interface)
                    else:
                        file_data["classes"].append(class_data)
            
            return file_data
            
        except Exception as e:
            print(f"Error processing file {file_path}: {e}")
            return {
                "path": self._get_relative_path(file_path),
                "module": self._get_module_name(file_path),
                "classes": [],
                "interfaces": [],
                "functions": [],
            }

    def _process_directory(self, directory: str) -> None:
        for root, _, files in os.walk(directory):
            for file in files:
                if file.endswith('.py'):
                    file_path = os.path.join(root, file)
                    self._process_file(file_path)

    def _process_file(self, file_path: str) -> None:
        file_data = self._process_file_and_return(file_path)
        
        module_name = file_data["module"]
        if module_name not in self.modules:
            self.modules[module_name] = []
        
        self.modules[module_name].append(file_data)

    def _extract_function(self, node: ast.FunctionDef) -> Dict:
        function = {
            "name": node.name,
            "parameters": self._extract_parameters(node),
            "return_types": self._extract_return_types(node),
            "body": self._get_function_body(node)
        }
        return function

    def _extract_parameters(self, node: ast.FunctionDef) -> List[Dict]:
        parameters = []
        
        for arg in node.args.args:
            param_type = "Any"
            if arg.annotation:
                param_type = self._get_type_as_string(arg.annotation)
            
            parameters.append({
                "name": arg.arg,
                "type": param_type
            })
        
        return parameters

    def _extract_return_types(self, node: ast.FunctionDef) -> List[str]:
        if node.returns:
            return_type = self._get_type_as_string(node.returns)
            if return_type.startswith("Union["):
                union_types = return_type[6:-1].split(", ")
                return union_types
            return [return_type]
        
        return_types = set()
        for subnode in ast.walk(node):
            if isinstance(subnode, ast.Return) and subnode.value:
                if isinstance(subnode.value, ast.Name):
                    return_types.add(subnode.value.id)
                elif isinstance(subnode.value, ast.Constant):
                    return_types.add(type(subnode.value.value).__name__)
        
        if not return_types and not node.returns:
            return ["None"]
        
        return list(return_types)

    def _extract_class(self, node: ast.ClassDef) -> Dict:
        class_data = {
            "name": node.name,
            "fields": [],
            "methods": []
        }
        
        for item in node.body:
            if isinstance(item, ast.AnnAssign):
                if isinstance(item.target, ast.Name):
                    field_type = self._get_type_as_string(item.annotation)
                    class_data["fields"].append({
                        "name": item.target.id,
                        "type": field_type
                    })
            elif isinstance(item, ast.FunctionDef):
                method = {
                    "reciever": node.name,
                    # Keep using "function" as the key to match the JSON tag in the Go struct
                    "function": self._extract_function(item)
                }
                class_data["methods"].append(method)
        
        return class_data

    def _is_interface(self, node: ast.ClassDef) -> bool:
        for base in node.bases:
            if isinstance(base, ast.Name):
                if base.id in ('ABC', 'Protocol', 'Interface'):
                    return True
            elif isinstance(base, ast.Attribute):
                if base.attr in ('ABC', 'Protocol', 'Interface'):
                    return True
        
        has_abstract_methods = False
        for item in node.body:
            if isinstance(item, ast.FunctionDef):
                for decorator in item.decorator_list:
                    if isinstance(decorator, ast.Name) and decorator.id == 'abstractmethod':
                        has_abstract_methods = True
                        break
        
        return has_abstract_methods

    def _get_type_as_string(self, annotation) -> str:
        if isinstance(annotation, ast.Name):
            return annotation.id
        elif isinstance(annotation, ast.Attribute):
            return f"{self._get_name_from_attribute(annotation)}"
        elif isinstance(annotation, ast.Subscript):
            value = self._get_type_as_string(annotation.value)
            if isinstance(annotation.slice, ast.Index):
                slice_value = self._get_type_as_string(annotation.slice.value)
            else:
                slice_value = self._get_type_as_string(annotation.slice)
            return f"{value}[{slice_value}]"
        elif isinstance(annotation, ast.Tuple):
            elts = [self._get_type_as_string(e) for e in annotation.elts]
            return f"Tuple[{', '.join(elts)}]"
        elif isinstance(annotation, ast.Constant):
            return str(annotation.value)
        elif isinstance(annotation, ast.List):
            return f"List"
        elif isinstance(annotation, ast.Dict):
            return f"Dict"
        else:
            return str(annotation)

    def _get_name_from_attribute(self, node: ast.Attribute) -> str:
        if isinstance(node.value, ast.Attribute):
            return f"{self._get_name_from_attribute(node.value)}.{node.attr}"
        elif isinstance(node.value, ast.Name):
            return f"{node.value.id}.{node.attr}"
        return node.attr

    def _get_function_body(self, node: ast.FunctionDef) -> str:
        try:
            with open(self.current_file, 'r', encoding='utf-8') as f:
                source_lines = f.readlines()
            
            start_line = node.lineno
            end_line = node.end_lineno if hasattr(node, 'end_lineno') else self._find_end_of_function(node, source_lines)
            
            body_lines = source_lines[start_line:end_line]
            return ''.join(body_lines).strip()
        except Exception:
            return ""

    def _find_end_of_function(self, node: ast.FunctionDef, source_lines: List[str]) -> int:
        start_line = node.lineno
        indent_level = len(source_lines[start_line - 1]) - len(source_lines[start_line - 1].lstrip())
        
        for i in range(start_line, len(source_lines)):
            line = source_lines[i]
            if line.strip() and len(line) - len(line.lstrip()) <= indent_level:
                return i
        
        return len(source_lines)


def main():
    parser = argparse.ArgumentParser(description='Generate metadata from Python code.')
    parser.add_argument('path', help='Path to a Python file or directory')
    parser.add_argument('-o', '--output', default='metadata.json', help='Output JSON file')
    parser.add_argument('-r', '--root', default=None, 
                       help='Root directory for calculating Python module paths (default: current directory)')
    
    args = parser.parse_args()
    
    analyzer = PythonAnalyzer(root_path=args.root)
    modules = analyzer.analyze_path(args.path)
    
    with open(args.output, 'w', encoding='utf-8') as f:
        json.dump(modules, f, indent=2)
    
    print(f"Metadata has been written to {args.output}")


if __name__ == "__main__":
    main()