# FAQ

Common issues running Palworld with this operator (or the Compose path).

## “Incapable version” / version mismatch

**Meaning:** the Steam **client** and the **dedicated server** are on different builds. Palworld rejects the join. After 1.0 patches this is almost always **server behind** (Steam auto-updates the client; the dedicated image does not).

**Confirm server version** (REST is cluster-internal by default — port-forward):

```shell
kubectl -n game-servers port-forward deploy/palworld-server 8212:8212 &
ADMIN=$(kubectl get secret palworld-server-secrets -n game-servers \
  -o jsonpath='{.data.admin-password}' | base64 -d)
curl -s -u "admin:${ADMIN}" http://127.0.0.1:8212/v1/api/info | jq '{version,worldguid,servername,days}'
```

**Fix (operator / official image):** bump `spec.serverImage` to a current Pocketpair tag from the [GHCR package](https://github.com/pocketpairjp/palworld-dedicated-server-docker/pkgs/container/palserver). Prefer an explicit version tag over a stale node-cached `latest`.

```shell
# Example — use the newest published tag that matches the client
kubectl -n game-servers patch palworldserver palworld-server --type=merge \
  -p '{"spec":{"serverImage":"ghcr.io/pocketpairjp/palserver:v1.0.1.100619","imagePullPolicy":"Always"}}'
```

Or enable **opt-in** auto-update (`spec.update.autoUpdateImage: true`) so the operator polls GHCR and pins the newest `vX.Y.Z.W` tag when safe (world pin learned, optional maintenance cron, prefer empty server). Status shows `runningVersion`, `latestAvailableVersion`, `updateAvailable`.

Wait for Ready, then re-check REST `version`. After a manual roll you can set `imagePullPolicy` back to `IfNotPresent` if you pin a digest/tag.

**Players:** update Palworld via Steam (or your storefront) so the client matches the server. A 1.0.x client needs a matching dedicated build.

**Live incident note (prd-apps, Jul 2026):** last confirmed REST version was `v1.0.0.100427` on image `ghcr.io/pocketpairjp/palserver:latest` (`@sha256:3a36c93e…`). Newest published tag was `v1.0.1.100619` (same digest as `latest` on GHCR). Clients on the newer Steam build hit “incapable version” until the CR was bumped / the node pulled the new digest.

Details: [PALWORLD_SERVER.md — Updating](PALWORLD_SERVER.md#updating-the-game-server-steam--patches).

## “No password entered”

The world has a **join** password (`ServerPassword`). Direct connect needs **Enter password** checked with the `server-password` Secret value — not the admin password.

```shell
kubectl get secret palworld-server-secrets -n game-servers \
  -o jsonpath='{.data.server-password}' | base64 -d; echo
```

Credentials come from bring-your-own Secret refs or `spec.generateSecrets: true` (operator creates `{cr-name}-secrets`). Full join flow: [CONNECT.md](CONNECT.md).

## How do I connect from the game?

Admins share `status.connectionAddress:status.connectionPort` (default `8211` UDP). In Palworld: **Join Multiplayer Game** → direct-connect `IP:PORT` → optional join password → **Connect**.

Step-by-step: [CONNECT.md](CONNECT.md). Landing-page summary: [site § Connect](https://dataknifeai.github.io/palworld-operator/#connect).

## World changed / empty after restart

Palworld loads the world named by `DedicatedServerName` in `GameUserSettings.ini` (folder under `SaveGames/0/`). The operator **learns** REST `worldguid` into `status.dedicatedServerName` and seeds `GameUserSettings.ini` on the Saved PVC (alongside `PalWorldSettings.ini`) so Recreate / auto-update rolls keep the world. You can also set `spec.dedicatedServerName` explicitly (recommended for GitOps).

After any roll, confirm REST `worldguid` still matches the intended save folder.

See [PALWORLD_SERVER.md — World selection](PALWORLD_SERVER.md#world-selection-across-restarts).

## Game settings reset after restart / PVC INI wiped

The `seed-settings` init **overwrites** `PalWorldSettings.ini` on the Saved PVC from the operator ConfigMap every start. Edits made only inside the PVC (or via REST) do not stick across rolls.

Put lasting gameplay settings in `spec.optionSettings` — a passthrough map of [PalWorldSettings.ini OptionSettings](https://docs.palworldgame.com/settings-and-operation/configuration/) keys (Pocketpair’s list is the source of truth). Management keys (`ServerName`, passwords, ports, RCON/REST, `CrossplayPlatforms`) come from dedicated CR fields and override the same keys in the map.

```yaml
spec:
  optionSettings:
    bExistPlayerAfterLogout: "True"
    WorkSpeedRate: "1.5"
```

Apply with a **merge-patch** (so other spec fields stay), wait for the ConfigMap, then **roll the game Deployment** so `seed-settings` re-copies the INI onto the PVC. The world save stays intact — do not delete the CR/PVC; keep `dedicatedServerName` / `worldguid`. Players disconnect briefly during the roll.

## Can I load server mods on Linux / the official image?

**Depends which kind.** This operator defaults to Linux `ghcr.io/pocketpairjp/palserver`. Pocketpair’s [official Workshop loader](https://docs.palworldgame.com/settings-and-operation/mod/) (`Mods/`, `PalModSettings.ini`, `-workshopdir`) is **Windows-only** — Linux PalServer does not load it. The live image has **no** `Mods/` dir; the saves PVC is `/pal/Package/Pal/Saved` only.

[Yorkhost’s UE4SS notes](https://yorkhost.fr/docs/en/palworld/mods-ue4ss) (community host, not Pocketpair): native Linux can take **PAK** files under `Pal/Content/Paks/`; **UE4SS** needs Windows DLL injection (`Pal/Binaries/Win64/`) / Proton, which is **not** `PalServer-Linux-Shipping`.

| Kind | Official Linux image |
|------|----------------------|
| Workshop (`Mods/`, `PalModSettings.ini`, `-workshopdir`) | Not loaded. Opt-in PVC still mounts `/pal/Package/Mods` (forward-looking Pocketpair path). |
| `.pak` / `.sig` | Possible if version-matched. Overlay is `Paks/~WorkshopMods` and `Paks/LogicMods` only — **never** the whole `Paks/` dir (would hide `Pal-LinuxServer.pak`). |
| UE4SS / Lua / Win64 | No. Different stack. |
| Client mods | `spec.optionSettings.bAllowClientMod` is **join policy only** — does not install mods on the server. Consoles cannot load PC client mods. |
| Config / balance | Use `spec.optionSettings` → `PalWorldSettings.ini`, not the mods PVC. |

`spec.mods.enabled: true` creates `{name}-mods` and (by default) those Paks **subpath** overlays. Enabling rolls the game pod; leave it off on a live world unless you accept the Recreate. **Back up the saves PVC and pin `spec.serverImage` first** — a bad PAK can stop the server from starting. Server content mods can also **lock consoles out** of a crossplay world.

Copy files with a short-lived pod (see [PALWORLD_SERVER.md — Mods](PALWORLD_SERVER.md#mods--linux-vs-windows-honest)). Optional `activeModList` seeds `PalModSettings.ini` (Windows loader / future Linux). `-workshopdir` stays off unless `useWorkshopDirArg: true`.

## How do server updates work with Steam / game patches?

| Image | How updates land |
|-------|------------------|
| **Official** `ghcr.io/pocketpairjp/palserver` (operator default) | **Bump `spec.serverImage` tag**, or opt in with `spec.update.autoUpdateImage`. No SteamCMD on boot. |
| Community SteamCMD images | In-container `app_update` / `UPDATE_ON_BOOT` (`spec.updateOnBoot`); auto-update image bumps are skipped unless the image is from `spec.update.imageRepository`. |

Auto-update is **off by default**. When enabled it lists GHCR tags anonymously, compares `vX.Y.Z.W`, defers the **image roll** while players are online (`onlyWhenEmpty`), optional cron windows (`checkSchedule` / `applySchedule`, timezone default **UTC**), and optional staged in-game warns via REST `POST /v1/api/announce` (`notifyPlayers` + `notifySchedule`, default 60m→10s; legacy `notifyLeadTime` = single stage). Warnings still fire while players are online. Pocketpair has **deprecated RCON**; this operator does not use RCON Broadcast.

Full table: [PALWORLD_SERVER.md — Updating](PALWORLD_SERVER.md#updating-the-game-server-steam--patches).

## Local PC vs Kubernetes cluster

| Path | When |
|------|------|
| **Docker Compose** | Gaming PC / laptop, no cluster — [LOCAL.md](LOCAL.md), `make compose-up` |
| **Operator** | Shared Kubernetes, Envoy Gateway, PVC, CRDs — [README](../README.md) |

Same official Pocketpair image either way.

## Glitchy / laggy performance

Palworld dedicated is heavy. Brief sizing hints:

- Prefer a **dedicated worker** (avoid control-plane / busy game nodes such as Windrose’s).
- Sample CR aims at ~8 Gi nodes: ~3 Gi request / 6 Gi limit, multi-core CPU, `multithreading: true`. Raise memory if OOM; keep `maxPlayers` modest.
- Compose path: ~8 Gi free RAM recommended; default mem cap `6g` — raise if the container is killed.
- After a move or restart, confirm you’re still on the intended world (`worldguid`) so “empty/glitchy” isn’t actually a fresh save.

More: [PALWORLD_SERVER.md](PALWORLD_SERVER.md) resources section, [LOCAL.md](LOCAL.md).

## Related

- [CONNECT.md](CONNECT.md) — join from the client
- [PALWORLD_SERVER.md](PALWORLD_SERVER.md) — ports, mounts, optionSettings, Linux vs Windows mods, updates, world pin, crossplay
- [LOCAL.md](LOCAL.md) — Compose on a PC
- [ARCHITECTURE.md](ARCHITECTURE.md) — owned resources / Gateway
