---
name: atunnels
description: Tunnel service (like ngrok) with URL shortener, MCP server, and SSH access. Use this skill when working with a-tunnels for exposing local services.
---

# A-Tunnels — LLM Usage Guide

## Qu'est-ce que a-tunnels ?

A-Tunnels est un service de tunneling auto-hébergé (alternative à ngrok) qui permet d'exposer des services locaux sur internet.

**Fonctionnalités :**
- HTTP/HTTPS/TCP/WebSocket tunneling
- URL shortener intégré
- Serveur MCP pour integration agentique
- SSH shell intégré
- Nettoyage automatique des tunnels inactifs
- Multi-langue (EN, FR, ES)

## Configuration

```yaml
# atunnels.yml
server:
  host: "0.0.0.0"
  http_port: 80
  https_port: 443
  api_port: 8080
  domain: "your-domain.com"
  
  # Modes de serveur (désactiver pour réduire la surface d'attaque)
  http_enabled: true
  https_enabled: true
  tcp_enabled: true
  ws_enabled: true
  api_enabled: true
  ssh_enabled: false
  mcp_enabled: false
  
  # Auto cleanup (désactive après X, supprime après Y)
  cleanup_enabled: true
  cleanup_interval: "1h"
  disable_after: "720h"   # 30 jours
  delete_after: "8760h"   # 1 an
  
  shortener:
    enabled: true
    base_path: "/s/"
    default_ttl: 24       # heures
    
  auth:
    api_keys:
      - "at_sk_your_key"
```

## CLI Usage

```bash
# Lister les tunnels
a-tunnels -token xxx list

# Créer un tunnel (nom aléatoire si absent)
a-tunnels -token xxx create http localhost:3000
a-tunnels -token xxx create mon-tunnel http localhost:3000

# Voir les stats
a-tunnels -token xxx stats mon-tunnel

# Supprimer
a-tunnels -token xxx delete mon-tunnel

# Langue (EN, FR, ES)
a-tunnels -lang fr -token xxx list
```

## API Endpoints

| Méthode | Chemin | Description |
|--------|--------|-------------|
| GET | `/api/v1/tunnels` | Liste tous les tunnels |
| POST | `/api/v1/tunnels` | Crée un tunnel |
| GET | `/api/v1/tunnels/{name}` | Détails d'un tunnel |
| DELETE | `/api/v1/tunnels/{name}` | Supprime un tunnel |
| GET | `/api/v1/tunnels/{name}/stats` | Stats d'un tunnel |
| GET | `/generate-name` | Génère un nom aléatoire |
| GET | `/metrics` | Métriques Prometheus |

## Créer 10 tunnels avec noms aléatoires

```bash
for i in $(seq 1 10); do
  a-tunnels -token xxx create http "localhost:300$i"
done
```

## Intégration MCP (pour agents)

Le serveur MCP expose des outils pour gérer les tunnels :

```
Tools:
- list_tunnels: Liste tous les tunnels
- get_tunnel: Détails d'un tunnel
- create_tunnel: Crée un tunnel
- delete_tunnel: Supprime un tunnel
- get_tunnel_stats: Stats d'un tunnel
```

## Variables d'environnement

| Variable | Description |
|----------|-------------|
| `ATUNNELS_TOKEN` | Token API (alternative au flag -token) |
| `ATUNNELS_LANG` | Langue (en, fr, es) |

## Démarrage rapide

```bash
# Compiler
go build -o atunnels ./cmd/server

# Démarrer le serveur
./atunnels -config atunnels.yml

# Ou avec le CLI
go run ./cmd/cli/main.go -token xxx list
```

## Pour un LLM utilisant a-tunnels

Si tu as besoin d'exposer un service local pour un webhook ou test :

1. Vérifie que le serveur tourne (`/health`)
2. Crée un tunnel : `a-tunnels create webhook http localhost:3000`
3. Utilise l'URL retournée pour ton usage
4. Pour webhook GitHub : l'URL doit être exposée publiquement (utiliser Cloudflare Tunnel)

## Sécurité

- Clés API configurables dans `auth.api_keys`
- Rate limiting sur l'API
- IP whitelist optionnelle par tunnel
- Métriques protégées par auth
- MCP avec token dédié
