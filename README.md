# A-Tunnels

Système de gestion de tunnels/webhooks auto-hébergé - alternative à ngrok.

## Fonctionnalités

- **Tunnels multiples**: HTTP, TCP, WebSocket
- **API REST**: Administration complète des tunnels
- **MCP Server**: Contrôle par IA via Model Context Protocol
- **SSH CLI**: Interface interactive via SSH
- **SDK Multi-langage**: Go, Python, JavaScript
- **Auto-hébergement**: Sur votre VPS personnel

## Installation

```bash
# Compiler le serveur
go build -o atunnels-server ./cmd/server

# Compiler le client
go build -o atunnels-client ./cmd/client
```

## Utilisation Serveur

```bash
./atunnels-server --config atunnels.yml
```

Ports:
- `80/443`: HTTP/HTTPS gateways
- `8080`: API REST
- `2222`: SSH CLI
- `27200`: MCP Server

## Utilisation Client

```bash
./atunnels-client --config client.yml
```

## API REST

```bash
# Lister les tunnels
curl -H "Authorization: Bearer at_sk_xxx" http://localhost:8080/api/v1/tunnels

# Créer un tunnel
curl -X POST -H "Authorization: Bearer at_sk_xxx" \
  -H "Content-Type: application/json" \
  -d '{"name":"webhook","protocol":"http","localAddr":"localhost:3000"}' \
  http://localhost:8080/api/v1/tunnels
```

## MCP Server

Connexion pour IA:
```
localhost:27200
```

## SSH CLI

```bash
ssh atunnels@localhost -p 2222
# Mot de passe: votre API key
```

## SDK

### Go
```go
client := atunnels.NewClient("http://localhost:8080", "at_sk_xxx")
tunnels, _ := client.ListTunnels()
```

### Python
```python
from atunnels import ATunnelsClient
client = ATunnelsClient("http://localhost:8080", "at_sk_xxx")
tunnels = client.list_tunnels()
```

### JavaScript
```javascript
const { ATunnelsClient } = require('@atunnels/sdk');
const client = new ATunnelsClient('http://localhost:8080', 'at_sk_xxx');
const tunnels = await client.listTunnels();
```

## Docker

```bash
docker-compose up -d
```
