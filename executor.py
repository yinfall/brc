import ast
import io
import sys
import traceback
import math
import os
import bpy

class PythonExecutor:
    """
    Executes Python code strings safely within the Blender environment.
    Captures stdout, stderr, and evaluates return expressions like Python Console.
    Maintains persistent state across multiple execution requests.
    """
    def __init__(self):
        self.globals = {}
        self.reset_globals()

    def reset_globals(self):
        """Reset the persistent global namespace."""
        self.globals = {
            '__name__': '__main__',
            '__doc__': None,
            '__package__': None,
            '__builtins__': __builtins__,
            'bpy': bpy,
            'os': os,
            'sys': sys,
            'math': math,
        }

    def execute(self, code: str) -> dict:
        """
        Executes code string and returns execution results.

        Args:
            code: Python code string to execute

        Returns:
            dict containing execution results:
                - success (bool)
                - stdout (str)
                - stderr (str)
                - result_repr (str or None)
                - output (str combined output)
        """
        stdout_capture = io.StringIO()
        stderr_capture = io.StringIO()

        old_stdout = sys.stdout
        old_stderr = sys.stderr

        success = True
        res_val = None

        clean_code = code.strip()

        try:
            sys.stdout = stdout_capture
            sys.stderr = stderr_capture

            parsed = None
            try:
                parsed = ast.parse(clean_code, filename="<remote_console>", mode="exec")
            except SyntaxError:
                exec(clean_code, self.globals)

            if parsed and parsed.body:
                last_stmt = parsed.body[-1]

                if isinstance(last_stmt, ast.Expr):
                    body_exec = parsed.body[:-1]
                    if body_exec:
                        mod_exec = ast.Module(body=body_exec, type_ignores=[])
                        code_exec = compile(mod_exec, filename="<remote_console>", mode="exec")
                        exec(code_exec, self.globals)

                    expr_mod = ast.Expression(body=last_stmt.value)
                    code_eval = compile(expr_mod, filename="<remote_console>", mode="eval")
                    res_val = eval(code_eval, self.globals)
                else:
                    code_exec = compile(parsed, filename="<remote_console>", mode="exec")
                    exec(code_exec, self.globals)

        except Exception:
            success = False
            traceback.print_exc(file=stderr_capture)
        finally:
            sys.stdout = old_stdout
            sys.stderr = old_stderr

        stdout_str = stdout_capture.getvalue()
        stderr_str = stderr_capture.getvalue()

        # Format combined output
        output_parts = []
        if stdout_str:
            output_parts.append(stdout_str)
        if stderr_str:
            output_parts.append(stderr_str)
        if res_val is not None:
            output_parts.append(repr(res_val) + "\n")

        full_output = "".join(output_parts)

        return {
            "success": success,
            "stdout": stdout_str,
            "stderr": stderr_str,
            "result_repr": repr(res_val) if res_val is not None else None,
            "output": full_output
        }

# Global singleton executor instance
executor = PythonExecutor()
