import bpy
import os
from .client import client_manager
from .executor import executor

def get_settings(context):
    """Get settings object (prefer scene, fallback to window_manager)."""
    scene = getattr(context, "scene", None)
    wm = getattr(context, "window_manager", None)
    if scene and hasattr(scene, "remote_console_is_running"):
        return scene
    if wm and hasattr(wm, "remote_console_is_running"):
        return wm
    return scene or wm

def set_running_status(context, is_running):
    scene = getattr(context, "scene", None)
    wm = getattr(context, "window_manager", None)
    if scene and hasattr(scene, "remote_console_is_running"):
        scene.remote_console_is_running = is_running
    if wm and hasattr(wm, "remote_console_is_running"):
        wm.remote_console_is_running = is_running


class REMOTE_CONSOLE_OT_start_client(bpy.types.Operator):
    """Connect to the BRC Daemon"""
    bl_idname = "wm.remote_console_start_client"
    bl_label = "Connect to Daemon"
    bl_description = "Connect to the background brc daemon to accept commands"
    bl_options = {'REGISTER'}

    def execute(self, context):
        success, msg = client_manager.start()

        if success:
            set_running_status(context, True)
            self.report({'INFO'}, msg)
        else:
            set_running_status(context, False)
            self.report({'ERROR'}, msg)

        return {'FINISHED'}


class REMOTE_CONSOLE_OT_stop_client(bpy.types.Operator):
    """Disconnect from the BRC Daemon"""
    bl_idname = "wm.remote_console_stop_client"
    bl_label = "Disconnect"
    bl_description = "Disconnect from the daemon"
    bl_options = {'REGISTER'}

    def execute(self, context):
        success, msg = client_manager.stop()
        set_running_status(context, False)

        if success:
            self.report({'INFO'}, msg)
        else:
            self.report({'ERROR'}, msg)

        return {'FINISHED'}


class REMOTE_CONSOLE_OT_reset_globals(bpy.types.Operator):
    """Reset the Python Console namespace"""
    bl_idname = "wm.remote_console_reset_globals"
    bl_label = "Reset Console Globals"
    bl_description = "Clear all user-defined variables and reset Python namespace to initial state"
    bl_options = {'REGISTER'}

    def execute(self, context):
        executor.reset_globals()
        self.report({'INFO'}, "Remote Console namespace reset successfully.")
        return {'FINISHED'}


class REMOTE_CONSOLE_OT_copy_brc_cmd(bpy.types.Operator):
    """Copy example brc command to clipboard"""
    bl_idname = "wm.remote_console_copy_brc_cmd"
    bl_label = "Copy brc Example"
    bl_description = "Copy a ready-to-use brc command to your OS clipboard"
    bl_options = {'REGISTER'}

    cmd_type: bpy.props.EnumProperty(
        name="Type",
        items=[
            ('SHORT', "Short Command", "Simple one-line command example"),
            ('FILE', "Script File", "Full .py file execution example"),
        ],
        default='SHORT'
    )

    def execute(self, context):
        pid = os.getpid()
        if self.cmd_type == 'SHORT':
            brc_str = f'brc -s {pid} exec "print(bpy.context.scene.name)"'
        else:
            brc_str = f'brc -s {pid} exec script.py'

        context.window_manager.clipboard = brc_str
        self.report({'INFO'}, f"Copied command to clipboard:\n{brc_str}")
        return {'FINISHED'}


class REMOTE_CONSOLE_OT_clear_logs(bpy.types.Operator):
    """Clear all terminal execution logs"""
    bl_idname = "wm.remote_console_clear_logs"
    bl_label = "Clear Terminal Logs"
    bl_description = "Clear all terminal execution logs and history"
    bl_options = {'REGISTER'}

    def execute(self, context):
        wm = getattr(context, "window_manager", None)
        scene = getattr(context, "scene", None)
        for target in (wm, scene):
            if target and hasattr(target, "remote_console_logs"):
                target.remote_console_logs.clear()
                target.remote_console_active_log_index = 0
        self.report({'INFO'}, "Terminal logs cleared.")
        return {'FINISHED'}


class REMOTE_CONSOLE_OT_copy_logs(bpy.types.Operator):
    """Copy all terminal logs to clipboard"""
    bl_idname = "wm.remote_console_copy_logs"
    bl_label = "Copy Terminal Logs"
    bl_description = "Copy all terminal output lines to your clipboard"
    bl_options = {'REGISTER'}

    def execute(self, context):
        wm = getattr(context, "window_manager", None)
        scene = getattr(context, "scene", None)
        logs = None
        if wm and hasattr(wm, "remote_console_logs") and len(wm.remote_console_logs) > 0:
            logs = wm.remote_console_logs
        elif scene and hasattr(scene, "remote_console_logs") and len(scene.remote_console_logs) > 0:
            logs = scene.remote_console_logs

        if not logs:
            self.report({'WARNING'}, "Terminal log is empty.")
            return {'CANCELLED'}

        lines = [item.text for item in logs]
        full_text = "\n".join(lines)
        context.window_manager.clipboard = full_text
        self.report({'INFO'}, f"Copied {len(lines)} log lines to clipboard.")
        return {'FINISHED'}


classes = (
    REMOTE_CONSOLE_OT_start_client,
    REMOTE_CONSOLE_OT_stop_client,
    REMOTE_CONSOLE_OT_reset_globals,
    REMOTE_CONSOLE_OT_copy_brc_cmd,
    REMOTE_CONSOLE_OT_clear_logs,
    REMOTE_CONSOLE_OT_copy_logs,
)

def register():
    for cls in classes:
        bpy.utils.register_class(cls)

def unregister():
    for cls in reversed(classes):
        bpy.utils.unregister_class(cls)
