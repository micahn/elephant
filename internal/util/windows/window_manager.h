#ifndef WINDOW_MANAGER_H
#define WINDOW_MANAGER_H

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
    int id;
    char *title;
    char *app_id;
    void *handle; // zwlr_foreign_toplevel_handle_v1*
} window_info_t;

typedef struct {
    window_info_t *windows;
    int count;
    int capacity;
} window_list_t;

// Initialize the window manager
int init_window_manager();

// Get list of all windows
window_list_t* get_window_list();

// Focus a window by ID
int focus_window(int window_id);

// Clean up resources
void cleanup_window_manager();

// Free window list
void free_window_list(window_list_t *list);

#ifdef __cplusplus
}
#endif

#endif // WINDOW_MANAGER_H
