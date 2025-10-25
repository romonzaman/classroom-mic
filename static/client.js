function wsConnect(role, id, name, channel = '') {
  const url = new URL('/ws', window.location.href);
  url.protocol = (location.protocol === 'https:') ? 'wss:' : 'ws:';
  url.searchParams.set('role', role);
  url.searchParams.set('id', id);
  url.searchParams.set('name', name);
  if (channel) {
    url.searchParams.set('channel', channel);
  }
  const ws = new WebSocket(url.toString());
  return ws;
}
