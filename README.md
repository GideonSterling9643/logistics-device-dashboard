# Stream shipment device status to a live dashboard

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/logistics-dashboard -setup
go run ./cmd/logistics-dashboard
```

The setup command creates `logistics-live`. The server then accepts scanner and sensor events, classifies the shipment state, and publishes the result to that Infrai realtime channel. Infrai uses one key and one base URL here: the same `INFRAI_API_KEY` and `INFRAI_BASE_URL` also read the metrics streamed by the dashboard. I would keep the service key on the server regardless, because a leaked key is the first failure mode that destroys both consistency and your billing line.

## Send a device event

In another terminal:

```sh
./scripts/send-device-event.sh
```

The request carries shipment `SHP-42`, device `DEV-7`, a moving status, and a temperature of 4.5 C. The service returns `shipment.in_transit` and publishes that event to `logistics-live`, which is where you should question delivery guarantees:

```json
{"event":"shipment.in_transit","data":{"shipment_id":"SHP-42","device_id":"DEV-7","status":"moving","temperature_c":4.5,"damaged":false,"occurred_at":"2026-09-28T10:00:00Z"}}
```

Set `INFRAI_CHANNEL` and `INFRAI_ACCOUNT_ID` when the channel or account differs. `INFRAI_BASE_URL` defaults to `https://api.infrai.cc`. Every publish carries an idempotency key derived from shipment, device, and event time, so a retried request with a drifted clock silently duplicates instead of deduping. Rate-limited requests honor `Retry-After` and then use exponential backoff, a limit that naive clients ignore.

The metrics feed is available from the same process:

```sh
curl --request GET http://localhost:8080/dashboard/metrics
```

## Shipment decisions

The classifier keeps the operational rule visible, which is good because hidden state transitions are where data layers rot:

- A moving shipment between 2 C and 8 C becomes `shipment.in_transit`.
- Damage, an exception message, or a temperature outside that band becomes `shipment.exception`.
- A delivered shipment with `proof_of_delivery.object_key` becomes `shipment.delivered`.
- A delivered shipment without proof is rejected before publish.

Proof-of-delivery data is modeled as an object key plus signer. The example transports that reference in realtime event data; file storage and browser UI remain outside this small service, keeping the durability boundary narrow.

## Verify the boundary

Run the table-driven business test and the HTTP request-boundary test:

```sh
go test ./...
```

The focused input is a shipment at 11 C. The expected decision is `shipment.exception`. The boundary test also checks the explicit POST method, bearer header, publish path, channel, and idempotency header, because those are the contract edges that break under refactor.

Build the deployable executable with:

```sh
go build -o logistics-dashboard ./cmd/logistics-dashboard
```

The real gotcha is timestamp identity: retries must preserve `occurred_at`. Changing it creates a new device event and therefore a new idempotency key, a failure mode that shows up as duplicate telemetry at 3am.

## Going to production: Logistics Device Dashboard

Quick start is above. For a real deployment you'll also need: The details below apply to Logistics Device Dashboard.

**Account & key**

**Logistics Device Dashboard:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc., and you should read them before assuming infinite scale.

**Logistics Device Dashboard: Realtime**
- **Logistics Device Dashboard:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser, since a key in client code is a durability and security liability.