// Package windows provides common functions for handling wayland windows.
package windows

/*
#cgo LDFLAGS: -lwayland-client
#include "window_manager.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
	"unsafe"
)

type Window struct {
	ID    int
	Title string
	AppID string
}

var (
	IsSetup bool
	mu      sync.Mutex
)

func Init() {
	mu.Lock()
	defer mu.Unlock()

	if IsSetup {
		return
	}

	count := 0

	for count < 10 {
		if err := InitWindowManager(); err != nil {
			slog.Error("windows", "init", err)
			slog.Info("windows", "setup", "retrying initWindowManager")
			count++
			time.Sleep(1 * time.Second)
		} else {
			IsSetup = true
			return
		}
	}

	slog.Error("windows", "init", "couldn't init window manager")
}

func InitWindowManager() error {
	result := C.init_window_manager()
	if result != 0 {
		return fmt.Errorf("failed to initialize window manager")
	}
	return nil
}

func GetWindowList() ([]Window, error) {
	windowList := C.get_window_list()
	if windowList == nil {
		return nil, fmt.Errorf("failed to get window list - window manager may not be initialized or no Wayland compositor found")
	}

	count := int(windowList.count)
	if count == 0 {
		return []Window{}, nil
	}

	if windowList.windows == nil {
		return nil, fmt.Errorf("window list array is null")
	}

	windows := make([]Window, count)

	// Access the C array safely
	windowArray := (*[1000]C.window_info_t)(unsafe.Pointer(windowList.windows))

	for i := range count {
		window := windowArray[i]

		var title, appID string
		if window.title != nil {
			title = C.GoString(window.title)
		}
		if window.app_id != nil {
			appID = C.GoString(window.app_id)
		}

		windows[i] = Window{
			ID:    int(window.id),
			Title: title,
			AppID: appID,
		}
	}

	return windows, nil
}

func FocusWindow(windowID int) error {
	result := C.focus_window(C.int(windowID))
	switch result {
	case 0:
		return nil
	case -1:
		return fmt.Errorf("window with ID %d not found", windowID)
	case -2:
		return fmt.Errorf("no seat available for focusing (may need input device)")
	default:
		return fmt.Errorf("failed to focus window with ID %d (error %d)", windowID, int(result))
	}
}
