# Architecture notes for implementers
#
# Reuse from windrose-operator (github.com/DataKnifeAI/windrose-operator):
# - GatewayConfig + EnvoyProxy naming conventions
# - ClusterIP + {name}-envoy backend split
# - PVC + Config/Secret mount patterns
# - Resource auto-selection from player count
# - Status phase/ready/connectionAddress/connectionPort
# - Prefer the publisher's official game-server image as default
#
# Palworld-specific deviations:
# - Primary transport is UDP 8211 (not TCP+UDP 7777)
# - Additional UDP query 27015 and TCP RCON 25575 / REST 8212
# - Default image: ghcr.io/pocketpairjp/palserver (official Pocketpair)
#   Optional: thijsvanloef/palworld-server-docker (env-driven community image)
# - Official image: ConfigMap/INI under /pal/Package/Pal/Saved (not env→INI)
# - Community image: env vars generate PalWorldSettings.ini; mount /palworld
# - Larger default PVC (100Gi) and higher memory tiers
# - Secrets for AdminPassword / ServerPassword (INI or env depending on image)
# - REST API should default to not exposed via Gateway
# - No DataKnifeAI palworld-server-docker repo needed while official GHCR image exists
#
# Official docs:
# - https://docs.palworldgame.com/getting-started/deploy-dedicated-server
# - https://docs.palworldgame.com/settings-and-operation/configuration/
# - https://github.com/pocketpairjp/palworld-dedicated-server-docker
# - https://github.com/orgs/pocketpairjp/packages/container/package/palserver
