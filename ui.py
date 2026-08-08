import bpy
from .server import server_manager

class REMOTE_CONSOLE_PT_main(bpy.types.Panel):
    """Main Panel in 3D Viewport Sidebar (N-Panel)"""
    bl_label = "Remote Console Server"
    bl_idname = "REMOTE_CONSOLE_PT_main"
    bl_space_type = 'VIEW_3D'
    bl_region_type = 'UI'
    bl_category = 'Remote Console'

    @classmethod
    def poll(cls, context):
        return context.area is not None and context.area.type == 'VIEW_3D'

    def draw(self, context):
        layout = self.layout

        # Prefer scene for property bindings, fallback to window_manager
        scene = getattr(context, "scene", None)
        wm = getattr(context, "window_manager", None)
        
        target = None
        if scene and hasattr(scene, "remote_console_host"):
            target = scene
        elif wm and hasattr(wm, "remote_console_host"):
            target = wm

        if not target:
            box = layout.box()
            box.label(text="Addon Properties Not Ready", icon='ERROR')
            box.label(text="Please disable and re-enable Addon in Preferences.")
            return

        try:
            is_running = getattr(target, "remote_console_is_running", False) or server_manager.is_running

            # --- Status Box ---
            box = layout.box()
            row = box.row(align=True)
            if is_running:
                row.label(text=f"Running ({server_manager.host}:{server_manager.port})", icon='CHECKMARK')
                row.operator("wm.remote_console_stop_server", text="Stop Server", icon='CANCEL')
            else:
                row.label(text="Status: Stopped", icon='PAUSE')
                row.operator("wm.remote_console_start_server", text="Start Server", icon='PLAY')


            # --- Network Configuration ---
            config_box = layout.box()
            config_box.label(text="Server Settings", icon='PREFERENCES')

            col = config_box.column(align=True)
            col.enabled = not is_running  # Disable editing config while running
            col.prop(target, "remote_console_host", text="Host")
            col.prop(target, "remote_console_port", text="Port")
            col.prop(target, "remote_console_token", text="Auth Token")

            config_box.prop(target, "remote_console_auto_start", text="Auto Start on Load")
            config_box.prop(target, "remote_console_echo_console", text="Echo to Python Console")


            # --- Quick Usage Examples ---
            usage_box = layout.box()
            usage_box.label(text="curl Quick Commands", icon='CONSOLE')

            row1 = usage_box.row(align=True)
            op1 = row1.operator("wm.remote_console_copy_curl_cmd", text="Copy Short Cmd", icon='COPYDOWN')
            op1.cmd_type = 'SHORT'

            op2 = row1.operator("wm.remote_console_copy_curl_cmd", text="Copy Script Cmd", icon='COPYDOWN')
            op2.cmd_type = 'FILE'

            # Text hints
            host = getattr(target, "remote_console_host", "127.0.0.1")
            port = getattr(target, "remote_console_port", 8182)
            token = getattr(target, "remote_console_token", "")
            token_hdr = f' -H "X-Auth-Token: {token}"' if token else ''

            col_hint = usage_box.column(align=True)
            col_hint.scale_y = 0.8
            col_hint.label(text="Short Command Example:")
            col_hint.label(text=f'curl -X POST http://{host}:{port}{token_hdr} -d "print(bpy.context.scene.name)"')
            col_hint.separator()
            col_hint.label(text="Script File Example:")
            col_hint.label(text=f'curl -X POST http://{host}:{port}{token_hdr} --data-binary @script.py')

            # --- Execution Stats & Tools ---
            stats_box = layout.box()
            stats_box.label(text="Runtime Info", icon='INFO')
            col_stats = stats_box.column(align=True)
            col_stats.scale_y = 0.85
            col_stats.label(text=f"Total Requests: {server_manager.total_requests}")
            col_stats.label(text=f"Last Req Time: {server_manager.last_time}")
            col_stats.label(text=f"Last Status: {server_manager.last_status}")
            if server_manager.last_code:
                col_stats.label(text=f"Last Code: {server_manager.last_code}")

            stats_box.separator()
            stats_box.operator("wm.remote_console_reset_globals", text="Reset Namespace", icon='FILE_REFRESH')

        except Exception as e:
            box = layout.box()
            box.label(text="Error rendering panel:", icon='ERROR')
            box.label(text=str(e))


classes = (
    REMOTE_CONSOLE_PT_main,
)

def register():
    for cls in classes:
        bpy.utils.register_class(cls)

def unregister():
    for cls in reversed(classes):
        bpy.utils.unregister_class(cls)
