# Manual stdio JSON-RPC examples

Build the server:

```bash
go build -o bin/k8s-mcp-server ./cmd/k8s-mcp-server
```

Start it:

```bash
./bin/k8s-mcp-server --mode=local --namespace=default
```

Then send newline-delimited JSON-RPC messages.

## initialize

```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"manual","version":"dev"}}}
```

## tools/list

```json
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
```

## cluster_info

```json
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"cluster_info","arguments":{}}}
```

## list pods

```json
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_pods","arguments":{"namespace":"default"}}}
```

## can-i

```json
{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"can_i","arguments":{"verb":"list","group":"","resource":"pods","namespace":"default"}}}
```
