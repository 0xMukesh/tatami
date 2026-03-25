# tatami

a minimal, educational X11 window manager written in go. 

tatami enforces a **pure tabbed layout**: every window is a tab, no tiling trees, no floating layers. the design model is very similar to a browser or to the tabbed mode in i3.

## demo

https://github.com/user-attachments/assets/7069d9b2-caa3-4c75-b6e1-68a27384ecf3

## configuration

configuration is to be defined at `~/.config/tatami/config.yaml`. example configuration can be found at [`config.yaml`](./config.yaml). 

tatami currently supports the following actions:

- `exec` - execute an arbitrary command  
- `close_focused` - close the active window  
- `quit` - exit tatami  
- `focus_left` - focus previous window  
- `focus_right` - focus next window  
- `move_left` - shift window left in tab order  
- `move_right` - shift window right in tab order  
- `focus_workspace` - switch to workspace (`args`: workspace index)  
- `move_window_to_workspace` - send window to workspace (`args`: workspace index)  

## references

- https://jichu4n.com/posts/how-x-window-managers-work-and-how-to-write-one-part-i/  
- https://movq.de/git/katriawm/file/README.html  
- https://tronche.com/gui/x/xlib/  
- https://github.com/CurlyMuch/tinywm-go/
