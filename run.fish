#!/usr/bin/fish

set XEPHYR (which Xephyr)

if test -z "$XEPHYR"
    echo "Xephyr not found, exiting"
    exit 1
end

xinit ./xinitrc -- $XEPHYR :100 -ac -screen 1000x800 -host-cursor
