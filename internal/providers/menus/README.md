### Elephant Menus

Create custom menus.

#### Features

- seamless menus
- use dmenu's as submenus
- drag&drop files into other programs
- copy file/path

#### How to create a menu

Default location for menu definitions is `~/.config/elephant/menus/`. Simply place a file in there, see examples below.

#### Examples

```toml
name = "other"
name_pretty = "Other"
icon = "applications-other"
global_search = true

[[entries]]
text = "Volume"
async = "echo $(wpctl get-volume @DEFAULT_AUDIO_SINK@)"
icon = "audio-volume-high"
action = "wpctl set-volume @DEFAULT_AUDIO_SINK@ %RESULT%"

[[entries]]
text = "System"
async = """echo $(echo "Memory: $(free -h | awk '/^Mem:/ {printf "%s/%s", $3, $2}') | CPU: $(top -bn1 | grep 'Cpu(s)' | awk '{printf "%.1f%%", 100 - $8}')")"""
icon = "computer"
action = ""

[[entries]]
text = "Today"
async = """echo $(date "+%H:%M - %d.%m. %A - KW %V")"""
icon = "clock"
action = ""
```

```toml
name = "screenshots"
name_pretty = "Screenshots"
icon = "camera-photo"
global_search = true

[[entries]]
text = "View"
action = "vimiv ~/Pictures/"

[[entries]]
text = "Annotate"
action = "wl-paste | satty -f -"

[[entries]]
text = "Toggle Record"
action = "record"

[[entries]]
text = "Screenshot Region"
action = "wayfreeze --after-freeze-cmd 'IMG=~/Pictures/$(date +%Y-%m-%d_%H-%M-%S).png && grim -g \"$(slurp)\" $IMG && wl-copy < $IMG; killall wayfreeze'"

[[entries]]
text = "Screenshot Window"
action = "wayfreeze --after-freeze-cmd 'IMG=~/Pictures/$(date +%Y-%m-%d_%H-%M-%S).png && grim $IMG && wl-copy < $IMG; killall wayfreeze'"

[[entries]]
text = "other menu"
submenu = "other"
```
