#!/usr/bin/fish

set XEPHYR (which Xephyr)

if test -z "$XEPHYR"
    echo "Xephyr not found, exiting"
    exit 1
end

xinit ./xinitrc -- $XEPHYR :100 -ac -screen 800x600 -host-cursor
