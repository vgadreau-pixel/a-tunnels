# A-Tunnels: Système de Gestion de Webhooks et Tunnels Auto-hébergé

## 1. Project Overview

**Nom du projet**: A-Tunnels  
**Type**: Application Go auto-hébergée (serveur + client)  
**Résumé**: Système de tunnels reverse proxy et gestion de webhooks permettant d'exposer des services locaux sur Internet sans ngrok, pilotable par IA via MCP/CLI et administration par API/SDK multi-plateforme.  
**Cible utilisateurs**: Développeurs, DevOps, systèmes pilotés par IA sur VPS personnel.

---

## 2. Architecture Système

```
┌─────────────────────────────────────────────────────────────────┐
│                        VPS (Serveur A-Tunnels)                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │   HTTP GW    │  │   TCP GW     │  │   WebSocket GW        │ │
│  │  :80/:443    │  │  :10000+     │  │   :11000+             │ │
│  └──────┬───────┘  └──────┬───────┘  └──────────┬───────────┘ │
│         │                 │                     │             │
│  ┌──────▼─────────────────▼─────────────────────▼───────────┐ │
│  │                    Tunnel Manager                          │ │
│  │  - Routing (subdomain → tunnel)                           │ │
│  │  - Authentification                                        │ │
│  │  - Rate limiting                                           │ │
│  │  - Logging/Métriques                                       │ │
│  └──────────────────────────┬────────────────────────────────┘ │
│                             │                                   │
│  ┌──────────────────────────▼────────────────────────────────┐ │
│  │              API Server (REST + WebSocket)                 │ │
│  │  - CRUD tunnels                                            │ │
│  │  - Stats temps réel                                        │ │
│  │  - Webhooks callbacks                                      │ │
│  └──────────────────────────┬────────────────────────────────┘ │
│                             │                                   │
│  ┌──────────────────────────▼────────────────────────────────┐ │
│  │              MCP Server (port 27200)                       │ │
│  │              SSH Server (port 2222)                         │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              ▲
                              │ Connexions persistantes
                              │
┌─────────────────────────────▼──────────────────────────────────┐
│                    Client (A-Tunnels CLI)                       │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐                │
│  │ localhost: │  │ localhost: │  │ any:port   │                │
│  │ 8080       │  │ 5432       │  │            │                │
│  └────────────┘  └────────────┘  └────────────┘                │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Spécification Fonctionnelle

### 3.1 Tunnel Types

| Type | Protocole | Usage | Port serveur suggéré |
|------|-----------|-------|---------------------|
| `http` | HTTP/HTTPS | Services web | 80/443 |
| `tcp` | TCP brut | Bases de données, services TCP | 10000-60000 |
| `websocket` | WS/WSS | Apps temps réel | 11000-65000 |

### 3.2 Modes de Déploiement

**Mode Server (VPS)**:
- Démarre tous les gateways (HTTP, TCP, WebSocket)
- Expose API REST, MCP, SSH
- Gère les connexions clientes

**Mode Client**:
- Connecte un ou plusieurs services locaux au serveur
- Authentifie via token/API key
- Peut tourner en arrière-plan (daemon)

### 3.3 Configuration des Tunnels

```yaml
tunnels:
  - name: "mon-webhook-stripe"
    subdomain: "stripe-webhook-vgad"  # Optionnel: sous-domaine personnalisé
    protocol: "http"
    local_addr: "localhost:3000"
    auth:  # Optionnel
      type: "bearer"
      token: "sk_live_xxx"
    headers:  # Headers à ajouter/override
      X-Custom: "value"
    timeout: 30s
    max_conns: 100

  - name: "ma-db-postgres"
    protocol: "tcp"
    local_addr: "localhost:5432"
    remote_port: 15432  # Port fixe ou automatique
```

### 3.4 API REST

| Méthode | Endpoint | Description |
|---------|----------|-------------|
| GET | `/api/v1/tunnels` | Liste tous les tunnels |
| POST | `/api/v1/tunnels` | Créer un tunnel |
| GET | `/api/v1/tunnels/:id` | Détails d'un tunnel |
| PUT | `/api/v1/tunnels/:id` | Modifier un tunnel |
| DELETE | `/api/v1/tunnels/:id` | Supprimer un tunnel |
| GET | `/api/v1/tunnels/:id/stats` | Métriques temps réel |
| POST | `/api/v1/tunnels/:id/restart` | Redémarrer un tunnel |
| GET | `/api/v1/tunnels/:id/logs` | Logs du tunnel |
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/metrics` | Métriques Prometheus |

### 3.5 Contrôle CLI/SSH

Commandes disponibles via CLI interactif ou SSH:

```bash
# Connexion SSH
ssh atunnels@ VPS -p 2222

# Commandes internes SSH
> list                    # Liste tunnels
> create webhook stripe  # Crée tunnel "webhook stripe"
> delete webhook         # Supprime tunnel
> stats webhook          # Affiche stats
> logs webhook --follow  # Logs temps réel
> exit                    # Quitte

# CLI distant
atunnels-cli --server VPS:2222 --token XXX list
atunnels-cli create --name myapi --protocol http --local localhost:8080
```

### 3.6 MCP Server (Model Context Protocol)

Outils exposés pour contrôle par IA:

```json
{
  "tools": [
    {
      "name": "list_tunnels",
      "description": "Liste tous les tunnels actifs"
    },
    {
      "name": "create_tunnel",
      "parameters": {
        "name": "string",
        "protocol": "http|tcp|websocket",
        "local_addr": "string",
        "subdomain": "string (optional)"
      }
    },
    {
      "name": "delete_tunnel",
      "parameters": {
        "name": "string"
      }
    },
    {
      "name": "get_tunnel_stats",
      "parameters": {
        "name": "string"
      }
    },
    {
      "name": "get_tunnel_logs",
      "parameters": {
        "name": "string",
        "lines": "number (default 100)"
      }
    },
    {
      "name": "restart_tunnel",
      "parameters": {
        "name": "string"
      }
    }
  ]
}
```

### 3.7 Authentification

- **API Key**: Token Bearer pour API REST
- **SSH Key**: Authentification par clé SSH
- **MCP**: Token d'accès configurable
- Rotation des tokens via API

### 3.8 Fonctionnalités Avancées

- **Subdomains dynamiques**: Allocation automatique (`a1b2c3d4.vps.domain.com`)
- **SSL automatique**: Let's Encrypt via CertMagic
- **Webhooks de callback**: Notifie un URL lors d'événements (connexion, erreur)
- **Rate limiting**: Limite par tunnel ou global
- ** IP whitelist**: Restriction d'accès par IP
- **Dashboard web optionnel**: Interface d'admin intégrable

---

## 4. Spécification Technique

### 4.1 Stack

- **Go**: 1.22+
- **Dépendances clés**:
  - `github.com/gorilla/mux` - Router HTTP
  - `github.com/gorilla/websocket` - WebSocket
  - `github.com/caddyserver/certmagic` - SSL automatique
  - `github.com/hashicorp/yamux` - Multiplexage TCP
  - `golang.org/x/crypto/ssh` - Serveur SSH
  - `github.com/prometheus/client_golang` - Métriques
  - `github.com/spf13/cobra` - CLI
  - `github.com/josharian/native` - Byte order
  - `github.com/redis/go-redis/v9` - Store optionnel (Redis)
  - `github.com/sqlite/sqlite` - Store local

### 4.2 Structure du Projet

```
a-tunnels/
├── cmd/
│   ├── server/          # Point d'entrée serveur
│   ├── client/          # Point d'entrée client CLI
│   └── cli/             # CLI admin distante
├── internal/
│   ├── tunnel/          # Core tunnel manager
│   ├── gateway/         # HTTP/TCP/WS gateways
│   ├── api/             # REST API handlers
│   ├── mcp/             # MCP protocol server
│   ├── ssh/             # SSH server/client
│   ├── storage/         # Persistence (SQLite)
│   ├── auth/            # Authentification
│   ├── metrics/         # Prometheus metrics
│   └── config/          # Configuration
├── pkg/
│   └── sdk/             # SDKs clients
│       ├── go/
│       ├── python/
│       └── javascript/
├── web/                 # Dashboard optionnel
├── docker/              # Dockerfiles
├── docker-compose.yml
├── atunnels.yml         # Configuration par défaut
├── go.mod
├── go.sum
└── README.md
```

### 4.3 Protocole Tunnel (Spécification)

Le client initie une connexion TCP persistante vers le serveur. Chaque tunnel utilise un channel multiplexé via Yamux.

```
[Client] TCP Conn (Yamux) ─────────────────────────────────► [Serveur]
    │
    ├─► Channel 0: Heartbeat/Control
    ├─► Channel 1: Tunnel "webhook-stripe" (HTTP)
    │       │
    │       └─► Request → Proxy to localhost:3000
    │       └─◄ Response ←
    │
    ├─► Channel 2: Tunnel "db-postgres" (TCP)
    │       │
    │       └─► Raw TCP → localhost:5432
    │       └─◄ Raw TCP ←
    │
    └─► Channel N...
```

### 4.4 Format Configuration (atunnels.yml)

```yaml
server:
  host: "0.0.0.0"
  http_port: 80
  https_port: 443
  tcp_port_start: 10000
  ws_port_start: 11000
  api_port: 8080
  mcp_port: 27200
  ssh_port: 2222

  tls:
    enabled: true
    email: "admin@domain.com"
    cert_cache: "/var/lib/atunnels/certs"

  auth:
    api_keys:
      - "at_sk_xxx1"
      - "at_sk_xxx2"
    ssh_keys:
      - "/etc/atunnels/keys/user1.pub"

  storage:
    type: "sqlite"
    path: "/var/lib/atunnels/atunnels.db"

  limits:
    max_tunnels: 100
    max_conns_per_tunnel: 1000
    rate_limit: "1000/m"

client:
  server_addr: "vps.domain.com:443"
  token: "at_sk_xxx"
  reconnect_interval: 5s

tunnels: []
```

---

## 5. Livrables

### 5.1 Binaire Serveur
- `atunnels-server` - Démarre le serveur complet
- Flags: `--config`, `--version`, `--help`

### 5.2 Binaire Client  
- `atunnels-client` - Agent à déployer côté machine locale
- `atunnels` - CLI unifiée (server + client)

### 5.3 SDK
- **Go**: `import "github.com/atunnels/sdk-go"`
- **Python**: `pip install atunnels-sdk`
- **JavaScript**: `npm install @atunnels/sdk`

### 5.4 MCP Integration
- Connexion via `npx @modelcontextprotocol/connect localhost:27200`

---

## 6. Critères de Succès

1. ✅ Serveur démarre et écoute sur tous les ports configurés
2. ✅ Client peut créer un tunnel HTTP functional
3. ✅ Client peut créer un tunnel TCP functional
4. ✅ API REST permet CRUD complet des tunnels
5. ✅ CLI/SSH permet gestion interactive
6. ✅ MCP server expose les outils pour IA
7. ✅ SSL automatique fonctionne pour sous-domaines
8. ✅ Métriques Prometheus accessibles
9. ✅ SDK Go compilent et fonctionnel
10. ✅ Tests unitaires passent (>70% coverage)
