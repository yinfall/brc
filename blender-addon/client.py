import socket
import threading
import queue
import json
import time
import os
import bpy
from .executor import executor

REQUEST_QUEUE = queue.Queue()

class DaemonClient:
    def __init__(self):
        self.host = "127.0.0.1"
        self.port = 8082
        self.is_running = False
        self.sock = None
        self.thread = None

        # Stats
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

    def start(self) -> tuple:
        if self.is_running:
            return True, f"Client is already running."

        import shutil
        import subprocess

        # Look in PATH, default installation directory ~/.brc/bin, or standard paths
        candidates = [
            shutil.which("brc"),
            shutil.which("brc.exe"),
            os.path.expanduser("~/.brc/bin/brc"),
            os.path.expanduser("~/.brc/bin/brc.exe"),
        ]
        brc_path = next((p for p in candidates if p and os.path.isfile(p)), None)
        if not brc_path:
            return False, "brc executable not found in system PATH or ~/.brc/bin"

        self.host = "127.0.0.1"
        self.port = 8082
        
        connected = False
        try:
            self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            self.sock.settimeout(0.5)
            self.sock.connect((self.host, self.port))
            connected = True
        except ConnectionRefusedError:
            pass
        except Exception as e:
            return False, f"Socket error: {str(e)}"

        if not connected:
            try:
                subprocess.Popen([brc_path, "daemon"], 
                                 stdout=subprocess.DEVNULL, 
                                 stderr=subprocess.DEVNULL,
                                 creationflags=getattr(subprocess, 'CREATE_NO_WINDOW', 0))
                
                for _ in range(20):
                    time.sleep(0.1)
                    try:
                        self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                        self.sock.settimeout(0.5)
                        self.sock.connect((self.host, self.port))
                        connected = True
                        break
                    except:
                        pass
                if not connected:
                    return False, "Failed to connect after starting daemon"
            except Exception as e:
                return False, f"Failed to start daemon: {str(e)}"

        try:
            self.sock.settimeout(None)
            
            # Send register
            pid = os.getpid()
            reg_msg = {"type": "register", "pid": pid}
            self.sock.sendall((json.dumps(reg_msg) + "\n").encode('utf-8'))
            
            self.is_running = True
            
            self.thread = threading.Thread(target=self._listen_loop, daemon=True)
            self.thread.start()
            
            if not bpy.app.timers.is_registered(process_queue_timer):
                bpy.app.timers.register(process_queue_timer, persistent=True)
                
            return True, f"Connected to Daemon on {self.host}:{self.port}"
        except Exception as e:
            self.is_running = False
            if self.sock:
                self.sock.close()
                self.sock = None
            return False, f"Failed to initialize connection: {str(e)}"

    def stop(self) -> tuple:
        if not self.is_running:
            return True, "Client is not running."
            
        self.is_running = False
        if self.sock:
            try:
                self.sock.close()
            except:
                pass
            self.sock = None
            
        if bpy.app.timers.is_registered(process_queue_timer):
            bpy.app.timers.unregister(process_queue_timer)
            
        return True, "Disconnected from Daemon."

    def _listen_loop(self):
        try:
            sock_file = self.sock.makefile('r', encoding='utf-8')
            while self.is_running:
                line = sock_file.readline()
                if not line:
                    break
                try:
                    msg = json.loads(line)
                    if msg.get("type") == "exec":
                        code = msg.get("code", "")
                        req_id = msg.get("id", "")
                        
                        self.record_request(code)
                        
                        result_event = threading.Event()
                        response_holder = {}
                        
                        REQUEST_QUEUE.put({
                            'code': code,
                            'id': req_id,
                            'event': result_event,
                            'response': response_holder
                        })
                        
                        # Wait for execution or timeout
                        completed = result_event.wait(timeout=60.0)
                        if not completed:
                            self.record_result(False)
                            res_msg = {
                                "type": "result",
                                "id": req_id,
                                "success": False,
                                "stderr": "Error 504: Execution timed out on Blender main thread.\n",
                                "stdout": ""
                            }
                            self.sock.sendall((json.dumps(res_msg) + "\n").encode('utf-8'))
                            continue
                            
                        exec_res = response_holder.get('result', {})
                        success = exec_res.get('success', False)
                        
                        self.record_result(success)
                        
                        res_msg = {
                            "type": "result",
                            "id": req_id,
                            "success": success,
                            "stdout": exec_res.get('stdout', ''),
                            "stderr": exec_res.get('stderr', ''),
                            "result_repr": exec_res.get('result_repr')
                        }
                        
                        self.sock.sendall((json.dumps(res_msg) + "\n").encode('utf-8'))
                except json.JSONDecodeError:
                    continue
                except Exception as e:
                    print(f"[Remote Console] Error processing message: {e}")
        except Exception as e:
            if self.is_running:
                print(f"[Remote Console] Daemon connection lost: {e}")
        finally:
            self.is_running = False
            if self.sock:
                self.sock.close()
                self.sock = None
                
            # If auto-start is true, we might want to try reconnecting later, but for now just stop
            context = bpy.context
            scene = getattr(context, "scene", None)
            wm = getattr(context, "window_manager", None)
            if scene and hasattr(scene, "remote_console_is_running"):
                scene.remote_console_is_running = False
            if wm and hasattr(wm, "remote_console_is_running"):
                wm.remote_console_is_running = False


def append_terminal_log(code: str, result: dict):
    try:
        context = bpy.context
        wm = getattr(context, "window_manager", None)
        scene = getattr(context, "scene", None)
        
        targets = []
        if wm and hasattr(wm, "remote_console_logs"):
            targets.append(wm)
        if scene and hasattr(scene, "remote_console_logs") and scene is not wm:
            targets.append(scene)

        if not targets:
            return

        current_time = time.strftime("%H:%M:%S")
        clean_code = code.strip()
        lines = clean_code.splitlines() if clean_code else []
        stdout_text = result.get("stdout", "")
        res_repr = result.get("result_repr")
        stderr_text = result.get("stderr", "")

        for target in targets:
            logs = target.remote_console_logs

            for idx, line in enumerate(lines):
                item = logs.add()
                item.timestamp = current_time if idx == 0 else ""
                item.log_type = 'INPUT'
                item.text = f">>> {line}" if idx == 0 else f"... {line}"

            if stdout_text:
                for line in stdout_text.rstrip("\n").splitlines():
                    item = logs.add()
                    item.timestamp = ""
                    item.log_type = 'OUTPUT'
                    item.text = f"    {line}"

            if res_repr is not None:
                item = logs.add()
                item.timestamp = ""
                item.log_type = 'OUTPUT'
                item.text = f" => {res_repr}"

            if stderr_text:
                for line in stderr_text.rstrip("\n").splitlines():
                    item = logs.add()
                    item.timestamp = ""
                    item.log_type = 'ERROR'
                    item.text = f" !  {line}"

            max_logs = 200
            while len(logs) > max_logs:
                logs.remove(0)

            target.remote_console_active_log_index = max(0, len(logs) - 1)
    except Exception as e:
        print(f"[Remote Console] Error updating terminal log: {e}")


def process_queue_timer():
    while not REQUEST_QUEUE.empty():
        try:
            req = REQUEST_QUEUE.get_nowait()
            code = req['code']
            event = req['event']
            response_holder = req['response']

            res = executor.execute(code)
            response_holder['result'] = res
            event.set()

            append_terminal_log(code, res)
        except queue.Empty:
            break
        except Exception as e:
            print(f"[Remote Console] Error processing queue: {e}")

    return 0.05

client_manager = DaemonClient()
