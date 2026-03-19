/**
 * A-Tunnels JavaScript SDK
 */

class Tunnel {
  constructor(data) {
    this.id = data.id;
    this.name = data.name;
    this.protocol = data.protocol;
    this.localAddr = data.local_addr;
    this.subdomain = data.subdomain;
    this.status = data.status;
  }
}

class TunnelStats {
  constructor(data) {
    this.activeConnections = data.active_connections || 0;
    this.totalRequests = data.total_requests || 0;
    this.totalBytesIn = data.total_bytes_in || 0;
    this.totalBytesOut = data.total_bytes_out || 0;
  }
}

class ATunnelsClient {
  constructor(serverUrl, token) {
    this.serverUrl = serverUrl.replace(/\/$/, '');
    this.token = token;
  }

  async _request(method, path, data = null) {
    const options = {
      method,
      headers: {
        'Authorization': `Bearer ${this.token}`,
        'Content-Type': 'application/json'
      }
    };

    if (data) {
      options.body = JSON.stringify(data);
    }

    const response = await fetch(`${this.serverUrl}${path}`, options);
    
    if (!response.ok) {
      throw new Error(`Request failed: ${response.status}`);
    }

    return response.json();
  }

  async listTunnels() {
    const data = await this._request('GET', '/api/v1/tunnels');
    return data.map(t => new Tunnel(t));
  }

  async getTunnel(name) {
    const data = await this._request('GET', `/api/v1/tunnels/${name}`);
    return new Tunnel(data);
  }

  async createTunnel(name, protocol, localAddr, subdomain = null) {
    const data = await this._request('POST', '/api/v1/tunnels', {
      name,
      protocol,
      localAddr,
      subdomain
    });
    return new Tunnel(data);
  }

  async deleteTunnel(name) {
    await this._request('DELETE', `/api/v1/tunnels/${name}`);
  }

  async getStats(name) {
    const data = await this._request('GET', `/api/v1/tunnels/${name}/stats`);
    return new TunnelStats(data);
  }

  async restartTunnel(name) {
    await this._request('POST', `/api/v1/tunnels/${name}/restart`);
  }

  async health() {
    try {
      const data = await this._request('GET', '/health');
      return data.status === 'ok';
    } catch {
      return false;
    }
  }
}

module.exports = { ATunnelsClient, Tunnel, TunnelStats };
