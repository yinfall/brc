import ast
import io
import sys
import traceback
import math
import os
import bpy

def echo_to_blender_console(text: str, is_input: bool = False):
    """
    Appends text to Blender's built-in Python Console space if open.
    """
    if not text:
        return

    try:
        wm = bpy.context.window_manager
        if not wm:
            return

        for window in wm.windows:
            screen = window.screen
            for area in screen.areas:
                if area.type == 'CONSOLE':
                    for space in area.spaces:
                        if space.type == 'CONSOLE':
                            line_type = 'INPUT' if is_input else 'OUTPUT'
                            for line in text.splitlines():
                                # Override context to trigger operator in console area
                                with bpy.context.temp_override(window=window, area=area, space_data=space):
                                    bpy.ops.console.scrollback_append(text=line, type=line_type)
                            return
    except Exception:
        # Fallback gracefully if context override fails or console is not active
        pass


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

    def execute(self, code: str, echo: bool = True) -> dict:
        """
        Executes code string and returns execution results.

        Args:
            code: Python code string to execute
            echo: If True, echoes code and results to Blender's built-in Python Console

        Returns:
            dict containing execution results
        """
        stdout_capture = io.StringIO()
        stderr_capture = io.StringIO()

        old_stdout = sys.stdout
        old_stderr = sys.stderr

        success = True
        res_val = None

        clean_code = code.strip()

        # Echo input code snippet to Blender Python Console
        if echo and clean_code:
            lines = clean_code.splitlines()
            if len(lines) > 5:
                echo_head = f"[Remote Code ({len(lines)} lines)]: " + lines[0] + " ..."
            else:
                echo_head = f"[Remote]: " + clean_code
            echo_to_blender_console(echo_head, is_input=True)

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

        # Echo output results to Blender Python Console
        if echo and full_output.strip():
            echo_to_blender_console(full_output.strip(), is_input=False)

        return {
            "success": success,
            "stdout": stdout_str,
            "stderr": stderr_str,
            "result_repr": repr(res_val) if res_val is not None else None,
            "output": full_output
        }

# Global singleton executor instance
executor = PythonExecutor()
