bl_info = {
    "name": "Blender Remote Console",
    "author": "Antigravity",
    "version": (0, 0, 2),
    "blender": (4, 0, 0),
    "location": "View3D > Sidebar > Remote Console",
    "description": "Execute Python code in Blender remotely from external CLI/terminal.",
    "warning": "",
    "doc_url": "https://github.com/yinfall/blender-remote-console",
    "tracker_url": "https://github.com/yinfall/blender-remote-console/issues",
    "category": "Development",
    "license": "GPL-3.0-or-later",
}

import bpy
from . import operators
from . import ui
from .client import client_manager


class RemoteConsoleLogItem(bpy.types.PropertyGroup):
    """Property group representing a single terminal log line."""
    text: str = bpy.props.StringProperty(name="Text", default="")
    log_type: str = bpy.props.EnumProperty(
        name="Type",
        items=[
            ('INPUT', "Input", "Command input", 'CONSOLE', 0),
            ('OUTPUT', "Output", "Standard output", 'BLANK1', 1),
            ('ERROR', "Error", "Error output", 'ERROR', 2),
            ('INFO', "Info", "Info message", 'INFO', 3),
        ],
        default='OUTPUT'
    )
    timestamp: str = bpy.props.StringProperty(name="Time", default="")


def auto_start_client():
    """Timer callback to auto-start client on load and keep it connected."""
    if not client_manager.is_running:
        success, msg = client_manager.start()
        
        context = bpy.context
        scene = getattr(context, "scene", None)
        wm = getattr(context, "window_manager", None)
        
        if scene and hasattr(scene, "remote_console_is_running"):
            scene.remote_console_is_running = success
            if hasattr(scene, "remote_console_error_msg"):
                scene.remote_console_error_msg = "" if success else msg
        if wm and hasattr(wm, "remote_console_is_running"):
            wm.remote_console_is_running = success
            if hasattr(wm, "remote_console_error_msg"):
                wm.remote_console_error_msg = "" if success else msg
    
    return 2.0  # Retry/Check every 2 seconds


def register_properties(target):
    target.remote_console_is_running = bpy.props.BoolProperty(
        name="Is Running",
        description="Current server running status",
        default=False
    )
    target.remote_console_error_msg = bpy.props.StringProperty(
        name="Error Message",
        description="Error message if daemon connection fails",
        default=""
    )
    target.remote_console_logs = bpy.props.CollectionProperty(
        type=RemoteConsoleLogItem,
        name="Terminal Logs"
    )
    target.remote_console_active_log_index = bpy.props.IntProperty(
        name="Active Log Index",
        default=0
    )


def unregister_properties(target):
    for prop in (
        "remote_console_is_running",
        "remote_console_error_msg",
        "remote_console_logs",
        "remote_console_active_log_index",
    ):
        if hasattr(target, prop):
            delattr(target, prop)


def register():
    # Register PropertyGroup first
    bpy.utils.register_class(RemoteConsoleLogItem)

    # Register properties on Scene AND WindowManager for maximum stability
    register_properties(bpy.types.Scene)
    register_properties(bpy.types.WindowManager)

    # Register Operators and UI Panels
    operators.register()
    ui.register()

    # Register auto-start timer (runs 1 sec after addon register, persistent across file loads)
    bpy.app.timers.register(auto_start_client, first_interval=1.0, persistent=True)


def unregister():
    # Stop client if currently running
    if client_manager.is_running:
        client_manager.stop()

    # Unregister Operators and UI Panels
    ui.unregister()
    operators.unregister()

    # Del properties
    unregister_properties(bpy.types.Scene)
    unregister_properties(bpy.types.WindowManager)

    # Unregister PropertyGroup
    bpy.utils.unregister_class(RemoteConsoleLogItem)


if __name__ == "__main__":
    register()
