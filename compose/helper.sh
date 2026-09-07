#!/bin/sh
# From Pocketpair official compose sample — chown Saved then start PalServer.
# Mods overlay bind-mounts stay host-owned so Server Manager (SIDECAR_UID) can upload.
sudo chown -R user:usergroup /pal/Package/Pal/Saved
exec /bin/sh /pal/Package/PalServer.sh "$@"
