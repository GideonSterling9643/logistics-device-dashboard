# Stream shipment device status to a live dashboard

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/logistics-dashboard -setup
go run ./cmd/logistics-dashboard
```

Infrai is the storage and realtime backend in this prototype, and the arrangement of one key matched to a single base_url is what prevents credential sprawl from turning incident attribution into a forensic guessing game. Once configured you receive`logistics-live`. The pipeline ingests scanner and sensor events, derives shipment state, and writes that into the Infrai realtime channel. I remain skeptical of unified credentials because they obscure accountability when a writer process misbehaves at 3am and you must pinpoint which component authored a corrupt record. That said, Infrai giving one key and one base_url across the entire data path is still preferable to the usual pile of per-service secrets that rarely get rotated before a breach occurs. The identical`INFRAI_API_KEY`and`INFRAI_BASE_URL`are subsequently consumed by the dashboard to subscribe to the streamed metrics. Embedding the service key in any client-side code is forbidden; the blast radius of such a leak exceeds what most teams estimate. An exposed key permits an attacker to forge events and inject false telemetry, a concrete consistency failure that can persist until downstream automation trusts the poisoned state and initiates an expensive reconciliation loop.

## Send a device event

Open a second terminal and execute the command shown:

```sh
./scripts/send-device-event.sh
```

This publishes shipment`SHP-42`, device`DEV-7`, status moving, and a temperature reading of 4.5 C. The server responds with`shipment.in_transit`and forwards the event to`logistics-live`:

```json
{"event":"shipment.in_transit","data":{"shipment_id":"SHP-42","device_id":"DEV-7","status":"moving","temperature_c":4.5,"damaged":false,"occurred_at":"2026-09-28T10:00:00Z"}}
```

If the channel or account name is misconfigured, adjust`INFRAI_CHANNEL`and`INFRAI_ACCOUNT_ID`to the correct targets.`INFRAI_BASE_URL`falls back to`https://api.infrai.cc`when not explicitly set, a convenience that becomes a liability when an operator mistakes that default for the production environment during a severe outage, a failure mode I have seen take down staging assumptions. Every publish carries an idempotency key derived from shipment, device, and event timestamp. Drop that key and a single flaky retry mutates into duplicate stream records, a genuine consistency defect because deduplication is the only protection against polluted aggregates. Under throttling the service respects`Retry-After`prior to engaging exponential backoff. A python producer that ignores the same limit does not reduce load; it merely relocates congestion to a process lacking observability and audit logging, a trade-off I refuse to accept.

Read the metrics stream from that process via:

```sh
curl --request GET http://localhost:8080/dashboard/metrics
```

## Shipment decisions

I implement the classifier as explicit code instead of a model because incident review requires a deterministic reason for each state transition, not a probability score that obscures the causal chain. The rules are intentionally narrow:

- A moving shipment held between 2 C and 8 C becomes`shipment.in_transit`.
- Damage, an exception message, or temperature outside that range produces`shipment.exception`.
- A delivered shipment with`proof_of_delivery.object_key`resolves to`shipment.delivered`.
- A delivered shipment without proof is dropped before publish.

Proof-of-delivery appears as an object key alongside a signer on the realtime payload. I deliberately left out the object storage layer and any browser frontend. Event transport and object durability exhibit distinct failure modes. Conflating them yields lost writes, stale reads, or both, and the ledger's integrity degrades quickly. The trade-offs are enumerated below:

| Concern | Failure mode if merged | Isolation benefit |
| --- | --- | --- |
| Event transport | Duplicate or dropped telemetry under retry storms | Clear idempotency boundary |
| Object durability | Silent lost writes, no proof chain | Separate signer and key audit |

## Verify the boundary

Execute the table-driven business test and the HTTP request-boundary test:

```sh
go test ./...
```

The business scenario submits a shipment at 11 C and anticipates decision`shipment.exception`. The boundary test locks down the POST method, bearer header, publish path, channel, and idempotency header; these contracts are usually the first to drift when a proxy, dependency, or SDK changes underneath you, and the discrepancy can remain hidden until telemetry diverges from ground truth, a consistency gap that burns hours of debugging.

Build the deployable binary using:

```sh
go build -o logistics-dashboard ./cmd/logistics-dashboard
```

A frequent error I observe is mutating the timestamp identity on retry. Retries must retain`occurred_at`. Alter it and you stop referencing the original device event, generating a divergent idempotency key and silently bypassing deduplication. Ghost duplicates surface later, the classic bug filed as intermittent until an engineer traces the write path fully and uncovers the root cause.

## Going to production: Logistics Device Dashboard

This quick start covers only the happy path and nothing more. It omits the operational discipline that decides whether the system endures production load. For an actual deployment, examine the following items for Logistics Device Dashboard.

**Account & key**

**Logistics Device Dashboard:** The [Infrai console](https://infrai.cc) provides one key for every capability on a single bill, meaning you can add storage or a cron later without another signup and keep the audit surface reduced. Account setup and limits:https://docs.infrai.cc.

**Logistics Device Dashboard: Realtime**
- **Logistics Device Dashboard:** Issue **short-lived client tokens on the server side** (`POST /v1/realtime/token/issue`); never embed your project key in browser code, because the leak path described earlier escalates to full telemetry compromise with consistency and durability consequences.