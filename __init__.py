bl_info = {
    "name": "Blender Remote Console",
    "author": "Antigravity",
    "version": (1, 0, 0),
    "blender": (4, 0, 0),
    "location": "View3D > Sidebar > Remote Console",
    "description": "Execute Python code and script files in Blender remotely via HTTP curl commands.",
    "warning": "",
    "doc_url": "",
    "category": "Development",
}

import bpy
from . import operators
from . import ui
from .server import server_manager

def auto_start_server():
    """Timer callback to auto-start server on load if enabled."""
    context = bpy.context
    scene = getattr(context, "scene", None)
    wm = getattr(context, "window_manager", None)
    
    auto_start = False
    if scene and hasattr(scene, "remote_console_auto_start"):
        auto_start = scene.remote_console_auto_start
    elif wm and hasattr(wm, "remote_console_auto_start"):
        auto_start = wm.remote_console_auto_start

    if auto_start and not server_manager.is_running:
        host = getattr(scene, "remote_console_host", "127.0.0.1")
        port = getattr(scene, "remote_console_port", 8182)
        token = getattr(scene, "remote_console_token", "")
        success, msg = server_manager.start(host, port, token)
        if success:
            if scene and hasattr(scene, "remote_console_is_running"):
                scene.remote_console_is_running = True
            if wm and hasattr(wm, "remote_console_is_running"):
                wm.remote_console_is_running = True
            print(f"[Remote Console] {msg}")
    return None  # Run once


def register_properties(target):
    target.remote_console_host = bpy.props.StringProperty(
        name="Host",
        description="IP Address to bind the HTTP server",
        default="127.0.0.1"
    )
    target.remote_console_port = bpy.props.IntProperty(
        name="Port",
        description="Port number for the HTTP server",
        default=8182,
        min=1024,
        max=65535
    )
    target.remote_console_token = bpy.props.StringProperty(
        name="Auth Token",
        description="Optional security token required in X-Auth-Token header or ?token= param",
        default="",
        subtype='PASSWORD'
    )
    target.remote_console_is_running = bpy.props.BoolProperty(
        name="Is Running",
        description="Current server running status",
        default=False
    )
    target.remote_console_auto_start = bpy.props.BoolProperty(
        name="Auto Start",
        description="Automatically start HTTP server when Blender opens or addon loads",
        default=False
    )
    target.remote_console_echo_console = bpy.props.BoolProperty(
        name="Echo to Python Console",
        description="Mirror executed remote commands and output into Blender's built-in Python Console window",
        default=True
    )


def unregister_properties(target):
    for prop in ("remote_console_host", "remote_console_port", "remote_console_token", "remote_console_is_running", "remote_console_auto_start", "remote_console_echo_console"):
        if hasattr(target, prop):
            delattr(target, prop)



def register():
    # Register properties on Scene AND WindowManager for maximum stability
    register_properties(bpy.types.Scene)
    register_properties(bpy.types.WindowManager)

    # Register Operators and UI Panels
    operators.register()
    ui.register()

    # Register auto-start timer (runs 1 sec after addon register)
    bpy.app.timers.register(auto_start_server, first_interval=1.0)


def unregister():
    # Stop server if currently running
    if server_manager.is_running:
        server_manager.stop()

    # Unregister Operators and UI Panels
    ui.unregister()
    operators.unregister()

    # Del properties
    unregister_properties(bpy.types.Scene)
    unregister_properties(bpy.types.WindowManager)


if __name__ == "__main__":
    register()
