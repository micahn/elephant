#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <wayland-client.h>
#include <wayland-client-protocol.h>
#include "wlr-foreign-toplevel-management-unstable-v1-client-protocol.h"
#include "window_manager.h"

static struct {
    struct wl_display *display;
    struct wl_registry *registry;
    struct zwlr_foreign_toplevel_manager_v1 *toplevel_manager;
    struct wl_seat *seat;
    window_list_t window_list;
    int next_id;
    int initialized;
} app_state = {0};

typedef struct {
    struct zwlr_foreign_toplevel_handle_v1 *handle;
    char *title;
    char *app_id;
    int id;
} toplevel_state_t;

static void update_window(int id, const char *title, const char *app_id);

static void add_window(toplevel_state_t *state) {
    // Check if window already exists
    for (int i = 0; i < app_state.window_list.count; i++) {
        if (app_state.window_list.windows[i].id == state->id) {
            // Window already exists, just update it
            update_window(state->id, state->title, state->app_id);
            return;
        }
    }

    // Add new window
    if (app_state.window_list.count >= app_state.window_list.capacity) {
        app_state.window_list.capacity = app_state.window_list.capacity == 0 ? 10 : app_state.window_list.capacity * 2;
        app_state.window_list.windows = realloc(app_state.window_list.windows,
                                               app_state.window_list.capacity * sizeof(window_info_t));
    }

    window_info_t *window = &app_state.window_list.windows[app_state.window_list.count];
    window->id = state->id;
    window->title = state->title ? strdup(state->title) : NULL;
    window->app_id = state->app_id ? strdup(state->app_id) : NULL;
    window->handle = state->handle;

    app_state.window_list.count++;
}

static void remove_window(int id) {
    for (int i = 0; i < app_state.window_list.count; i++) {
        if (app_state.window_list.windows[i].id == id) {
            free(app_state.window_list.windows[i].title);
            free(app_state.window_list.windows[i].app_id);

            // Move last window to this position
            if (i < app_state.window_list.count - 1) {
                app_state.window_list.windows[i] = app_state.window_list.windows[app_state.window_list.count - 1];
            }
            app_state.window_list.count--;
            break;
        }
    }
}

static void update_window(int id, const char *title, const char *app_id) {
    for (int i = 0; i < app_state.window_list.count; i++) {
        if (app_state.window_list.windows[i].id == id) {
            if (title) {
                free(app_state.window_list.windows[i].title);
                app_state.window_list.windows[i].title = strdup(title);
            }
            if (app_id) {
                free(app_state.window_list.windows[i].app_id);
                app_state.window_list.windows[i].app_id = strdup(app_id);
            }
            break;
        }
    }
}

static void toplevel_handle_title(void *data,
                                 struct zwlr_foreign_toplevel_handle_v1 *handle,
                                 const char *title) {
    (void)handle;
    toplevel_state_t *state = data;

    free(state->title);
    state->title = strdup(title);
    update_window(state->id, title, NULL);
}

static void toplevel_handle_app_id(void *data,
                                  struct zwlr_foreign_toplevel_handle_v1 *handle,
                                  const char *app_id) {
    (void)handle;
    toplevel_state_t *state = data;

    free(state->app_id);
    state->app_id = strdup(app_id);
    update_window(state->id, NULL, app_id);
}

static void toplevel_handle_done(void *data,
                                struct zwlr_foreign_toplevel_handle_v1 *handle) {
    (void)handle;
    toplevel_state_t *state = data;
    add_window(state);
}

static void toplevel_handle_closed(void *data,
                                  struct zwlr_foreign_toplevel_handle_v1 *handle) {
    (void)handle;
    toplevel_state_t *state = data;

    remove_window(state->id);
    zwlr_foreign_toplevel_handle_v1_destroy(state->handle);
    free(state->title);
    free(state->app_id);
    free(state);
}

static void toplevel_handle_output_enter(void *data,
                                        struct zwlr_foreign_toplevel_handle_v1 *handle,
                                        struct wl_output *output) {
    (void)data; (void)handle; (void)output;
}

static void toplevel_handle_output_leave(void *data,
                                        struct zwlr_foreign_toplevel_handle_v1 *handle,
                                        struct wl_output *output) {
    (void)data; (void)handle; (void)output;
}

static void toplevel_handle_state(void *data,
                                 struct zwlr_foreign_toplevel_handle_v1 *handle,
                                 struct wl_array *state) {
    (void)data; (void)handle; (void)state;
}

static void toplevel_handle_parent(void *data,
                                  struct zwlr_foreign_toplevel_handle_v1 *handle,
                                  struct zwlr_foreign_toplevel_handle_v1 *parent) {
    (void)data; (void)handle; (void)parent;
}

static const struct zwlr_foreign_toplevel_handle_v1_listener toplevel_listener = {
    .title = toplevel_handle_title,
    .app_id = toplevel_handle_app_id,
    .output_enter = toplevel_handle_output_enter,
    .output_leave = toplevel_handle_output_leave,
    .state = toplevel_handle_state,
    .done = toplevel_handle_done,
    .closed = toplevel_handle_closed,
    .parent = toplevel_handle_parent,
};

static void toplevel_manager_handle_toplevel(void *data,
                                           struct zwlr_foreign_toplevel_manager_v1 *manager,
                                           struct zwlr_foreign_toplevel_handle_v1 *toplevel) {
    (void)data; (void)manager;
    toplevel_state_t *state = calloc(1, sizeof(toplevel_state_t));
    state->handle = toplevel;
    state->id = app_state.next_id++;

    zwlr_foreign_toplevel_handle_v1_add_listener(toplevel, &toplevel_listener, state);
}

static void toplevel_manager_handle_finished(void *data,
                                            struct zwlr_foreign_toplevel_manager_v1 *manager) {
    (void)data; (void)manager;
}

static const struct zwlr_foreign_toplevel_manager_v1_listener toplevel_manager_listener = {
    .toplevel = toplevel_manager_handle_toplevel,
    .finished = toplevel_manager_handle_finished,
};

static void registry_handle_global(void *data, struct wl_registry *registry,
                                  uint32_t name, const char *interface,
                                  uint32_t version) {
    (void)data;

    if (strcmp(interface, zwlr_foreign_toplevel_manager_v1_interface.name) == 0) {
        app_state.toplevel_manager = wl_registry_bind(registry, name,
            &zwlr_foreign_toplevel_manager_v1_interface,
            version < 3 ? version : 3);
        zwlr_foreign_toplevel_manager_v1_add_listener(app_state.toplevel_manager,
            &toplevel_manager_listener, &app_state);
    } else if (strcmp(interface, wl_seat_interface.name) == 0) {
        app_state.seat = wl_registry_bind(registry, name, &wl_seat_interface,
                                         version < 7 ? version : 7);
    }
}

static void registry_handle_global_remove(void *data,
                                         struct wl_registry *registry,
                                         uint32_t name) {
    (void)data; (void)registry; (void)name;
}

static const struct wl_registry_listener registry_listener = {
    .global = registry_handle_global,
    .global_remove = registry_handle_global_remove,
};

int init_window_manager() {
    if (app_state.initialized) {
        return 0; // Already initialized
    }

    app_state.display = wl_display_connect(NULL);
    if (!app_state.display) {
        fprintf(stderr, "Failed to connect to Wayland display\n");
        return -1;
    }

    app_state.registry = wl_display_get_registry(app_state.display);
    if (!app_state.registry) {
        fprintf(stderr, "Failed to get Wayland registry\n");
        cleanup_window_manager();
        return -1;
    }

    wl_registry_add_listener(app_state.registry, &registry_listener, &app_state);

    wl_display_roundtrip(app_state.display);

    if (!app_state.toplevel_manager) {
        fprintf(stderr, "Wayland compositor does not support wlr-foreign-toplevel-management protocol\n");
        cleanup_window_manager();
        return -1;
    }

    wl_display_roundtrip(app_state.display);

    app_state.initialized = 1;
    return 0;
}

window_list_t* get_window_list() {
    if (!app_state.initialized || !app_state.display) {
        return NULL;
    }

    wl_display_roundtrip(app_state.display);
    return &app_state.window_list;
}

int focus_window(int window_id) {
    for (int i = 0; i < app_state.window_list.count; i++) {
        if (app_state.window_list.windows[i].id == window_id) {
            struct zwlr_foreign_toplevel_handle_v1 *handle =
                (struct zwlr_foreign_toplevel_handle_v1*)app_state.window_list.windows[i].handle;

            if (!app_state.seat) {
                return -2; // No seat available
            }

            zwlr_foreign_toplevel_handle_v1_activate(handle, app_state.seat);
            wl_display_flush(app_state.display);
            return 0;
        }
    }
    return -1;
}

void cleanup_window_manager() {
    if (!app_state.initialized) {
        return;
    }

    free_window_list(&app_state.window_list);

    if (app_state.seat) {
        wl_seat_destroy(app_state.seat);
        app_state.seat = NULL;
    }
    if (app_state.toplevel_manager) {
        zwlr_foreign_toplevel_manager_v1_destroy(app_state.toplevel_manager);
        app_state.toplevel_manager = NULL;
    }
    if (app_state.registry) {
        wl_registry_destroy(app_state.registry);
        app_state.registry = NULL;
    }
    if (app_state.display) {
        wl_display_disconnect(app_state.display);
        app_state.display = NULL;
    }

    app_state.next_id = 0;
    app_state.initialized = 0;
}

void free_window_list(window_list_t *list) {
    if (!list) return;

    for (int i = 0; i < list->count; i++) {
        free(list->windows[i].title);
        free(list->windows[i].app_id);
    }
    free(list->windows);
    list->windows = NULL;
    list->count = 0;
    list->capacity = 0;
}
