"""
A-Tunnels Python SDK
"""

import requests
from typing import List, Optional, Dict


class Tunnel:
    def __init__(self, data: dict):
        self.id = data.get("id")
        self.name = data.get("name")
        self.protocol = data.get("protocol")
        self.local_addr = data.get("local_addr")
        self.subdomain = data.get("subdomain")
        self.status = data.get("status")


class TunnelStats:
    def __init__(self, data: dict):
        self.active_connections = data.get("active_connections", 0)
        self.total_requests = data.get("total_requests", 0)
        self.total_bytes_in = data.get("total_bytes_in", 0)
        self.total_bytes_out = data.get("total_bytes_out", 0)


class ATunnelsClient:
    def __init__(self, server_url: str, token: str):
        self.server_url = server_url.rstrip("/")
        self.token = token
        self.session = requests.Session()
        self.session.headers.update(
            {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}
        )

    def _request(self, method: str, path: str, data: Optional[dict] = None):
        url = f"{self.server_url}{path}"
        resp = self.session.request(method, url, json=data)
        resp.raise_for_status()
        return resp.json()

    def list_tunnels(self) -> List[Tunnel]:
        data = self._request("GET", "/api/v1/tunnels")
        return [Tunnel(t) for t in data]

    def get_tunnel(self, name: str) -> Tunnel:
        data = self._request("GET", f"/api/v1/tunnels/{name}")
        return Tunnel(data)

    def create_tunnel(
        self, name: str, protocol: str, local_addr: str, subdomain: Optional[str] = None
    ) -> Tunnel:
        data = self._request(
            "POST",
            "/api/v1/tunnels",
            {
                "name": name,
                "protocol": protocol,
                "localAddr": local_addr,
                "subdomain": subdomain,
            },
        )
        return Tunnel(data)

    def delete_tunnel(self, name: str):
        self._request("DELETE", f"/api/v1/tunnels/{name}")

    def get_stats(self, name: str) -> TunnelStats:
        data = self._request("GET", f"/api/v1/tunnels/{name}/stats")
        return TunnelStats(data)

    def restart_tunnel(self, name: str):
        self._request("POST", f"/api/v1/tunnels/{name}/restart")

    def health(self) -> bool:
        try:
            data = self._request("GET", "/health")
            return data.get("status") == "ok"
        except:
            return False


__all__ = ["ATunnelsClient", "Tunnel", "TunnelStats"]
