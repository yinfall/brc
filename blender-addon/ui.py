import bpy
import os
from .client import client_manager
import bpy.types

_icon_cache = {}
def get_icon(icon_name, fallback='NONE'):
    if icon_name in _icon_cache:
        return _icon_cache[icon_name]
    try:
        if icon_name in bpy.types.UILayout.bl_rna.functions["label"].parameters["icon"].enum_items:
            _icon_cache[icon_name] = icon_name
        else:
            _icon_cache[icon_name] = fallback
    except Exception:
        _icon_cache[icon_name] = fallback
    return _icon_cache[icon_name]


class REMOTE_CONSOLE_UL_log_list(bpy.types.UIList):
    """UIList for scrollable terminal output logs."""
    def draw_item(self, context, layout, data, item, icon, active_data, active_propname, index):
        if self.layout_type in {'DEFAULT', 'COMPACT'}:
            row = layout.row(align=True)

            if item.log_type == 'INPUT':
                row.label(text=item.text, icon='CONSOLE')
            elif item.log_type == 'ERROR':
                row.label(text=item.text, icon='ERROR')
            elif item.log_type == 'INFO':
                row.label(text=item.text, icon='INFO')
            else:
                row.label(text=item.text, icon='BLANK1')

            if item.timestamp:
                sub = row.row()
                sub.alignment = 'RIGHT'
                sub.label(text=item.timestamp)


class REMOTE_CONSOLE_PT_main(bpy.types.Panel):
    """Main Panel in 3D Viewport Sidebar (N-Panel)"""
    bl_label = "Remote Console Session"
    bl_idname = "REMOTE_CONSOLE_PT_main"
    bl_space_type = 'VIEW_3D'
    bl_region_type = 'UI'
    bl_category = 'Remote Console'

    @classmethod
    def poll(cls, context):
        return context.area is not None and context.area.type == 'VIEW_3D'

    def draw(self, context):
        layout = self.layout

        scene = getattr(context, "scene", None)
        wm = getattr(context, "window_manager", None)
        
        target = None
        if scene and hasattr(scene, "remote_console_is_running"):
            target = scene
        elif wm and hasattr(wm, "remote_console_is_running"):
            target = wm

        if not target:
            box = layout.box()
            box.label(text="Addon Properties Not Ready", icon='ERROR')
            box.label(text="Please disable and re-enable Addon in Preferences.")
            return

        try:
            is_running = getattr(target, "remote_console_is_running", False) or client_manager.is_running
            error_msg = getattr(target, "remote_console_error_msg", "")
            pid = os.getpid()

            # --- Status Box ---
            box = layout.box()
            row = box.row(align=True)
            if is_running:
                row.label(text=f"Connected (PID: {pid})", icon=get_icon('CHECKMARK', 'FILE_TICK'))
                row.operator("wm.remote_console_stop_client", text="Disconnect", icon='CANCEL')
            else:
                row.label(text="Status: Disconnected", icon='PAUSE')
                row.operator("wm.remote_console_start_client", text="Connect to Daemon", icon='PLAY')

            if not is_running and error_msg:
                err_box = box.box()
                err_box.label(text=error_msg, icon='ERROR')

            # --- Quick Usage Examples ---
            usage_box = layout.box()
            usage_box.label(text="brc Quick Commands", icon='CONSOLE')

            row1 = usage_box.row(align=True)
            op1 = row1.operator("wm.remote_console_copy_brc_cmd", text="Copy Short Cmd", icon=get_icon('COPYDOWN', 'NONE'))
            op1.cmd_type = 'SHORT'

            op2 = row1.operator("wm.remote_console_copy_brc_cmd", text="Copy Script Cmd", icon=get_icon('COPYDOWN', 'NONE'))
            op2.cmd_type = 'FILE'

            col_hint = usage_box.column(align=True)
            col_hint.scale_y = 0.8
            col_hint.label(text="Short Command Example:")
            col_hint.label(text=f'brc -s {pid} exec "print(bpy.context.scene.name)"')
            col_hint.separator()
            col_hint.label(text="Script File Example:")
            col_hint.label(text=f'brc -s {pid} exec script.py')

            # --- Runtime Info & Scrollable Terminal Output ---
            stats_box = layout.box()
            
            row_stats_hdr = stats_box.row(align=True)
            row_stats_hdr.label(text="Runtime Terminal", icon='CONSOLE')
            row_stats_hdr.operator("wm.remote_console_copy_logs", text="", icon=get_icon('COPYDOWN', 'NONE'))
            row_stats_hdr.operator("wm.remote_console_clear_logs", text="", icon=get_icon('TRASH', 'X'))

            row_info = stats_box.row(align=True)
            row_info.scale_y = 0.85
            row_info.label(text=f"Total: {client_manager.total_requests}")
            row_info.label(text=f"Last: {client_manager.last_time}")
            row_info.label(text=f"Status: {client_manager.last_status}")

            stats_box.template_list(
                "REMOTE_CONSOLE_UL_log_list",
                "",
                target,
                "remote_console_logs",
                target,
                "remote_console_active_log_index",
                rows=7,
                maxrows=12
            )


            stats_box.separator()
            stats_box.operator("wm.remote_console_reset_globals", text="Reset Namespace", icon='FILE_REFRESH')

        except Exception as e:
            box = layout.box()
            box.label(text="Error rendering panel:", icon='ERROR')
            box.label(text=str(e))


classes = (
    REMOTE_CONSOLE_UL_log_list,
    REMOTE_CONSOLE_PT_main,
)

def register():
    for cls in classes:
        bpy.utils.register_class(cls)

def unregister():
    for cls in reversed(classes):
        bpy.utils.unregister_class(cls)
