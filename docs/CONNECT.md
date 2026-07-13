# Connect from inside Palworld

How players join a world run by **palworld-operator**, from the Palworld client.

Official server networking notes: [Requirements](https://tech.palworldgame.com/getting-started/requirements) (UDP **8211** by default) and [Deploy dedicated server](https://docs.palworldgame.com/getting-started/deploy-dedicated-server).

## Get the address (admins)

After the `PalworldServer` is Ready, read connection details from status:

```shell
kubectl get palworldserver -n game-servers
# NAME               READY   ADDRESS            PORT
# palworld-server    True    192.168.14.187     8211

kubectl get palworldserver palworld-server -n game-servers \
  -o jsonpath='{.status.connectionAddress}:{.status.connectionPort}{"\n"}'
```

Share with players as:

```text
<connectionAddress>:<connectionPort>
```

Example: `192.168.14.187:8211`

- **Default game port:** `8211` / **UDP** (overridable via `spec.gamePort`)
- Status `connectionAddress` comes from the Gateway / `spec.gateway.address`
- Status `connectionPort` is the game port players type in the client

### Join password vs admin password

The credentials Secret uses two keys (bring-your-own or operator-generated via
`spec.generateSecrets: true`):

| Secret key | Used for | Share with players? |
|------------|----------|---------------------|
| `server-password` | In-game **join** / `ServerPassword` | Yes (trusted players only) |
| `admin-password` | In-game admin / RCON (`AdminPassword`) | **No** — operators only |

Find the Secret name from status (sample BYO name shown; auto-gen defaults to `{cr-name}-secrets`):

```shell
kubectl get palworldserver palworld-server -n game-servers \
  -o jsonpath='{.status.credentialsSecretName}{"\n"}'
```

```shell
# Reveal join password only when you intend to share it
kubectl get secret palworld-server-secrets -n game-servers \
  -o jsonpath='{.data.server-password}' | base64 -d; echo

# Reveal admin / RCON password (operators only — do not share with players)
kubectl get secret palworld-server-secrets -n game-servers \
  -o jsonpath='{.data.admin-password}' | base64 -d; echo
```

Do not put passwords in the CR or commit them to git. Auto-gen never writes
plaintext passwords into `status` — only the Secret name and
`credentialsGenerated: true`.

## In-game steps (PC / Steam — direct connect)

Direct connect is the most reliable path for private operator-hosted worlds.

1. Launch **Palworld** and open the main menu.
2. Choose **Join Multiplayer Game**.
3. At the bottom of the multiplayer screen, find the direct-connect address field.
4. Enter the address as `IP:PORT` using status values, e.g. `203.0.113.25:8211`.
5. If the world uses a join password:
   - Enable **Enter password** (checkbox / toggle next to the address field).
   - Enter the **server** password (`server-password` Secret key), not the admin password.
6. Click **Connect**.

Everyone should be on a compatible Palworld client build (update through your storefront before joining).

### Password prompt quirks

If the client fails to ask for a password on direct connect (a long-standing client quirk on some builds):

1. Open **Community Servers**.
2. Select any **password-locked** community entry and enter *your* join password when prompted, then cancel / decline joining that random server.
3. Immediately enter your real `IP:PORT` in the direct-connect field and **Connect** again.

Prefer checking **Enter password** first on current builds; use the workaround only if the prompt never appears.

## Community browser vs direct connect

| Path | When to use | Operator notes |
|------|-------------|----------------|
| **Direct connect** (`IP:PORT`) | Private worlds, friends, most reliable | Default recommendation. Needs UDP game port reachable (default **8211**). |
| **Community Servers** list | Public discovery / consoles that cannot direct-IP | Requires `spec.community.enabled: true`, correct public listing settings, and Steam **query** port (**27015** / UDP) exposed. Listing can be slow or incomplete under load. |

Console players (Xbox, PlayStation, and some Game Pass / storefront clients) typically **cannot** use direct IP connect. They need the community list (or invite flows where available), so enable community listing and crossplay if you host those platforms.

## Crossplay

Dedicated servers can allow Steam / Xbox / PS5 / Mac via `spec.crossplayPlatforms` (maps to `CrossplayPlatforms` in settings). Example from the sample CR:

```yaml
crossplayPlatforms: "(Steam,Xbox,PS5,Mac)"
```

Crossplay does not remove the join-path difference: Steam PC can usually direct-connect; many console clients still need community listing. Align platform allow-list, community listing, and which address you share with each player group.

## Ports players care about

| Port | Proto | Player-facing? |
|------|-------|----------------|
| **8211** | UDP | **Yes** — game traffic / direct connect (default) |
| 27015 | UDP | Community browser / Steam query (when listing) |
| 25575 | TCP | RCON — ops only, not for joining |
| 8212 | TCP | REST API — ops only; keep off the public internet unless intentional |

## Quick share template

```text
World: <your serverName>
Address: <connectionAddress>:<connectionPort>
Join password: <server-password value>
Notes: Update Palworld first. Use Join Multiplayer Game → direct connect.
       Do not use the admin password to join.
```

## Related

- [LOCAL.md](LOCAL.md) — Docker Compose on a PC (no Kubernetes)
- [PALWORLD_SERVER.md](PALWORLD_SERVER.md) — ports, INI/env, persistence, resources
- [ARCHITECTURE.md](ARCHITECTURE.md) — Gateway / UDPRoute layout
- Sample CR: [`config/samples/palworld_v1alpha1_palworldserver.yaml`](../config/samples/palworld_v1alpha1_palworldserver.yaml)
