#!/bin/sh
# From Pocketpair official compose sample — chown Saved then start PalServer.
sudo chown -R user:usergroup /pal/Package/Pal/Saved
exec /bin/sh /pal/Package/PalServer.sh "$@"
