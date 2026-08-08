import bpy
from .server import server_manager
from .executor import executor

def get_settings(context):
    """Get settings object (prefer scene, fallback to window_manager)."""
    scene = getattr(context, "scene", None)
    wm = getattr(context, "window_manager", None)
    if scene and hasattr(scene, "remote_console_host"):
        return scene
    if wm and hasattr(wm, "remote_console_host"):
        return wm
    return scene or wm

def set_running_status(context, is_running):
    scene = getattr(context, "scene", None)
    wm = getattr(context, "window_manager", None)
    if scene and hasattr(scene, "remote_console_is_running"):
        scene.remote_console_is_running = is_running
    if wm and hasattr(wm, "remote_console_is_running"):
        wm.remote_console_is_running = is_running


class REMOTE_CONSOLE_OT_start_server(bpy.types.Operator):
    """Start the Remote Console HTTP server"""
    bl_idname = "wm.remote_console_start_server"
    bl_label = "Start Server"
    bl_description = "Start listening for incoming HTTP requests and curl commands"
    bl_options = {'REGISTER'}

    def execute(self, context):
        settings = get_settings(context)
        host = getattr(settings, "remote_console_host", "127.0.0.1")
        port = getattr(settings, "remote_console_port", 8182)
        token = getattr(settings, "remote_console_token", "")

        success, msg = server_manager.start(host, port, token)

        if success:
            set_running_status(context, True)
            self.report({'INFO'}, msg)
        else:
            set_running_status(context, False)
            self.report({'ERROR'}, msg)

        return {'FINISHED'}


class REMOTE_CONSOLE_OT_stop_server(bpy.types.Operator):
    """Stop the Remote Console HTTP server"""
    bl_idname = "wm.remote_console_stop_server"
    bl_label = "Stop Server"
    bl_description = "Stop listening for HTTP requests"
    bl_options = {'REGISTER'}

    def execute(self, context):
        success, msg = server_manager.stop()
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


class REMOTE_CONSOLE_OT_copy_curl_cmd(bpy.types.Operator):
    """Copy example curl command to clipboard"""
    bl_idname = "wm.remote_console_copy_curl_cmd"
    bl_label = "Copy curl Example"
    bl_description = "Copy a ready-to-use curl command to your OS clipboard"
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
        settings = get_settings(context)
        host = getattr(settings, "remote_console_host", "127.0.0.1")
        port = getattr(settings, "remote_console_port", 8182)
        token = getattr(settings, "remote_console_token", "")

        token_header = f" -H \"X-Auth-Token: {token}\"" if token else ""

        if self.cmd_type == 'SHORT':
            curl_str = f'curl -X POST http://{host}:{port}{token_header} -d "print(bpy.context.scene.name)"'
        else:
            curl_str = f'curl -X POST http://{host}:{port}{token_header} --data-binary @script.py'

        context.window_manager.clipboard = curl_str
        self.report({'INFO'}, f"Copied curl command to clipboard:\n{curl_str}")
        return {'FINISHED'}


classes = (
    REMOTE_CONSOLE_OT_start_server,
    REMOTE_CONSOLE_OT_stop_server,
    REMOTE_CONSOLE_OT_reset_globals,
    REMOTE_CONSOLE_OT_copy_curl_cmd,
)

def register():
    for cls in classes:
        bpy.utils.register_class(cls)

def unregister():
    for cls in reversed(classes):
        bpy.utils.unregister_class(cls)
