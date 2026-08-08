import http.server
import socketserver
import threading
import queue
import urllib.parse
import json
import time
import bpy
from .executor import executor

REQUEST_QUEUE = queue.Queue()

class RemoteConsoleHTTPHandler(http.server.BaseHTTPRequestHandler):
    """HTTP Request Handler for processing Remote Console calls."""

    server_manager = None  # Reference to RemoteConsoleServerManager instance

    def log_message(self, format, *args):
        # Disable default stderr HTTP logging to keep console clean
        pass

    def _check_auth(self, query_params) -> bool:
        """Verify authentication token if configured."""
        required_token = self.server_manager.auth_token if self.server_manager else ""
        if not required_token:
            return True

        # Check HTTP Headers
        header_token = self.headers.get('X-Auth-Token') or self.headers.get('X-Token')
        if not header_token and 'Authorization' in self.headers:
            auth_val = self.headers.get('Authorization', '')
            if auth_val.startswith('Bearer '):
                header_token = auth_val[7:].strip()

        if header_token == required_token:
            return True

        # Check Query Parameters
        param_token = query_params.get('token', [''])[0]
        if param_token == required_token:
            return True

        return False

    def do_GET(self):
        self._handle_request(is_get=True)

    def do_POST(self):
        self._handle_request(is_get=False)

    def _handle_request(self, is_get=False):
        parsed_url = urllib.parse.urlparse(self.path)
        query_params = urllib.parse.parse_qs(parsed_url.query)

        # Check Auth Token
        if not self._check_auth(query_params):
            self.send_response(401)
            self.send_header("Content-Type", "text/plain; charset=utf-8")
            self.end_headers()
            self.wfile.write(b"Error 401: Unauthorized. Invalid or missing authentication token.\n")
            return

        code = ""
        if is_get:
            code = query_params.get('cmd', query_params.get('code', ['']))[0]
        else:
            content_length = int(self.headers.get('Content-Length', 0))
            raw_body = self.rfile.read(content_length).decode('utf-8', errors='replace')
            content_type = self.headers.get('Content-Type', '')

            if 'application/json' in content_type:
                try:
                    data = json.loads(raw_body)
                    code = data.get('cmd') or data.get('code') or ''
                except Exception:
                    code = raw_body
            elif 'application/x-www-form-urlencoded' in content_type:
                form_params = urllib.parse.parse_qs(raw_body)
                if 'cmd' in form_params or 'code' in form_params:
                    code = form_params.get('cmd', form_params.get('code', ['']))[0]
                else:
                    code = raw_body
            else:
                # Raw plain text or binary py script payload
                code = raw_body

        if not code.strip():
            self.send_response(400)
            self.send_header("Content-Type", "text/plain; charset=utf-8")
            self.end_headers()
            self.wfile.write(b"Error 400: No Python code provided in request.\n")
            return

        # Queue execution to Blender main thread
        result_event = threading.Event()
        response_holder = {}

        REQUEST_QUEUE.put({
            'code': code,
            'event': result_event,
            'response': response_holder
        })

        # Update stats
        if self.server_manager:
            self.server_manager.record_request(code)

        # Wait for Blender main thread execution (Timeout 60 seconds)
        completed = result_event.wait(timeout=60.0)

        if not completed:
            self.send_response(504)
            self.send_header("Content-Type", "text/plain; charset=utf-8")
            self.end_headers()
            self.wfile.write(b"Error 504: Execution timed out on Blender main thread.\n")
            if self.server_manager:
                self.server_manager.record_result(False)
            return

        exec_res = response_holder.get('result', {})
        success = exec_res.get('success', False)
        output = exec_res.get('output', '')

        if self.server_manager:
            self.server_manager.record_result(success)

        status_code = 200 if success else 500
        self.send_response(status_code)
        self.send_header("Content-Type", "text/plain; charset=utf-8")
        self.end_headers()
        self.wfile.write(output.encode('utf-8', errors='replace'))


class RemoteConsoleServerManager:
    """Manages HTTP Server lifecycle and main-thread execution timer."""

    def __init__(self):
        self.httpd = None
        self.server_thread = None
        self.is_running = False
        self.host = "127.0.0.1"
        self.port = 8182
        self.auth_token = ""

        # Runtime Stats
        self.total_requests = 0
        self.successful_requests = 0
        self.failed_requests = 0
        self.last_code = ""
        self.last_status = "N/A"
        self.last_time = "N/A"

    def record_request(self, code: str):
        self.total_requests += 1
        snippet = code.strip().splitlines()[0] if code.strip() else ""
        if len(snippet) > 40:
            snippet = snippet[:37] + "..."
        self.last_code = snippet
        self.last_time = time.strftime("%H:%M:%S")

    def record_result(self, success: bool):
        if success:
            self.successful_requests += 1
            self.last_status = "SUCCESS"
        else:
            self.failed_requests += 1
            self.last_status = "ERROR"

    def start(self, host: str, port: int, token: str) -> tuple[bool, str]:
        """Start the HTTP server on specified host and port."""
        if self.is_running:
            return True, f"Server is already running on {self.host}:{self.port}"

        self.host = host
        self.port = port
        self.auth_token = token

        try:
            # Dual-stack socket server setup
            class ThreadedHTTPServer(socketserver.ThreadingMixIn, http.server.HTTPServer):
                daemon_threads = True
                allow_reuse_address = True

            RemoteConsoleHTTPHandler.server_manager = self
            self.httpd = ThreadedHTTPServer((self.host, self.port), RemoteConsoleHTTPHandler)

            self.server_thread = threading.Thread(target=self.httpd.serve_forever, daemon=True)
            self.server_thread.start()

            self.is_running = True

            # Register main thread timer if not already registered
            if not bpy.app.timers.is_registered(process_queue_timer):
                bpy.app.timers.register(process_queue_timer, persistent=True)

            return True, f"Server started successfully on {self.host}:{self.port}"

        except Exception as e:
            self.is_running = False
            self.httpd = None
            return False, f"Failed to start server: {str(e)}"

    def stop(self) -> tuple[bool, str]:
        """Stop the HTTP server and unregister timer."""
        if not self.is_running:
            return True, "Server is not running."

        try:
            if self.httpd:
                self.httpd.shutdown()
                self.httpd.server_close()
                self.httpd = None

            self.is_running = False

            if bpy.app.timers.is_registered(process_queue_timer):
                bpy.app.timers.unregister(process_queue_timer)

            return True, "Server stopped successfully."
        except Exception as e:
            self.is_running = False
            return False, f"Error stopping server: {str(e)}"


def process_queue_timer():
    """Blender main thread timer function to execute queued python requests."""
    while not REQUEST_QUEUE.empty():
        try:
            req = REQUEST_QUEUE.get_nowait()
            code = req['code']
            event = req['event']
            response_holder = req['response']

            # Query Echo to Python Console setting
            context = bpy.context
            scene = getattr(context, "scene", None)
            wm = getattr(context, "window_manager", None)
            echo = True
            if scene and hasattr(scene, "remote_console_echo_console"):
                echo = scene.remote_console_echo_console
            elif wm and hasattr(wm, "remote_console_echo_console"):
                echo = wm.remote_console_echo_console

            # Execute code inside Blender main thread
            res = executor.execute(code, echo=echo)
            response_holder['result'] = res
            event.set()
        except queue.Empty:
            break
        except Exception as e:
            print(f"[Remote Console] Error processing queue: {e}")

    # Return delay in seconds for next timer execution (50ms)
    return 0.05



# Global singleton manager instance
server_manager = RemoteConsoleServerManager()
