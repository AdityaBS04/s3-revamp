# s3-jepsen — Project Handoff

---

## 1. Who I am and how to work with me

- I'm **Aditya**, a software engineer in Bangalore. For this project, **treat me as a beginner in computer science.**
- **Explain everything in simple, plain English.** If you must use a technical term, first explain it in everyday words, with an analogy if possible, and only then use the term.
- Go **one step at a time.** Don't dump everything at once. After each step, tell me what I should see if it worked.
- When you write code, **comment it generously** and explain what each part does before or after showing it.
- If a command might behave differently on my machine, say so, and help me debug from the exact error I paste.
- Don't sign me up for paid services or create cloud accounts without asking first.

---

## 2. What we're building (in plain words)

**One-line pitch:** "Jepsen for object storage." It's a testing tool that checks whether online file-storage services keep their "only one winner" promise when many requests hit them at the same time and when things break.

**The picture:**
- Services like Amazon **S3** are giant warehouses of lockers (**buckets** = rows of lockers, **objects** = files, **keys** = locker labels). Many others copy Amazon's rules ("S3-compatible"): Cloudflare R2, Google Cloud Storage (GCS), Tigris, and self-hosted programs like SeaweedFS, Garage, Ceph RGW and RustFS.
- They offer two special rules (**conditional writes**):
  - **"Only if empty"**, the `If-None-Match: *` header. Store only if nothing is in the locker yet.
  - **"Only if unchanged"**, the `If-Match: <ETag>` header. Replace only if the file still has the version sticker (**ETag**) I last saw. This is also called **compare-and-swap (CAS)**.
- Apps use these rules as **locks** ("whoever creates `leader.lock` first is the boss"). If two clients both get "success," both think they're the boss, and data gets lost silently.
- Some warehouses break this promise, especially under heavy simultaneous traffic (**race conditions**) or failures.

**What the tool does:**
1. **Crowd the counter.** It fires many simultaneous requests (Go **goroutines**) at the same key.
2. **Write everything down.** It records a **history** (a log): who, what, when it started, when the reply came, and what the reply was.
3. **Judge the log.** It feeds the history to **Porcupine**, a ready-made Go **linearizability checker**. Its question: "Could a single cashier serving one customer at a time have produced these replies?" If not, it's a bug.
4. **Make trouble on purpose.** **Fault injection** with **Toxiproxy** (slow or cut the network) and killing containers.
5. **Publish a report card.** A **cross-vendor anomaly matrix**: vendor × version × operation × condition → ✅ correct / ❌ bug / ⚠️ silently ignored / 🚫 unsupported by design. Every cell comes with an exact reproduction command.

**Positioning (be honest about it):** many of the vendor bugs are already known. The novelty is the **first reusable, checker-backed, fault-injected, cross-vendor matrix that gets re-run regularly**, not "discovering" bugs.

---

## 3. Current status

✅ **Done (roadmap phases 1–4, basic version):**
- `docker-compose.yml` runs **SeaweedFS 4.54** (port 8333) and **Garage v2.5.0** (port 3900), both pinned. Garage sets itself up (bucket `test` + a fixed throwaway key) through `--single-node --default-bucket`.
- The Go tool races N clients with `If-None-Match: *` on one fresh key, records a history, and checks it with Porcupine.
- Result: SeaweedFS has 1 winner and is judged CORRECT. Garage lets every client win and is judged BROKEN.

**Code layout (one file per job, keep it this small and clean):**

| File | Job |
|---|---|
| `main.go` | Entry point: pick a store, run the race, print the verdict |
| `store.go` | The list of stores and how to connect to each (retries off) |
| `race.go` | Fires the simultaneous requests, then one final GET |
| `history.go` | The `Op` record (one line of the log) and the table printer |
| `check.go` | The Porcupine model of one correct locker |
| `check_test.go` | Hand-made good and bad histories that test the checker itself |

**Run it:**
```bash
docker compose up -d
go run . -store seaweedfs
go run . -store garage -clients 32
go test ./...
```

**Not built yet:** `If-Match` (CAS), conditional DELETE, ETags in the history, several ops per client, saving histories to a file, more stores, fault injection, the matrix.

---

## 4. Next step

Add **compare-and-swap** (`If-Match: <ETag>`): record the ETag in each `Op`, add the request kind to `race.go`, and extend the model in `check.go` (plus new cases in `check_test.go`).

---

## 5. Roadmap (about 3–4 months part-time)

| Phase | Weeks | Goal |
|---|---|---|
| 0. Learn | 1–3 | Tour of Go (through Concurrency), Go by Example (goroutines, channels, WaitGroup, context), MDN HTTP conditional requests, Docker "Get started", Kingsbury's "Strong consistency models", Athalye's "Testing Distributed Systems for Linearizability" + Porcupine README, the AWS S3 "Conditional writes" docs |
| 1. First conditional write | 4 | Manual tests on SeaweedFS ✅ and Garage, then the same in Go with **AWS SDK for Go v2** (`IfNoneMatch`, `IfMatch`) |
| 2. Race test | 5–6 | N goroutines (start with 16) wait on one start signal, then all fire `If-None-Match: *` at the same key with **unique bodies** (`client-7-op-3`). **Disable SDK retries** (max attempts = 1). Count 200s. More than one is "two winners." GET the final body |
| 3. Record history | 6–7 | Log `{client, op, key, input, call time, return time, status, etag}` per op. Run all clients in **one process** and use Go's monotonic clock, so there's no cross-machine clock problem. **Timeouts and 5xx = "unknown," never "failed"** |
| 4. Porcupine check | 8–9 | Model one key as a register `{exists, etag, body}`: create-if-absent, CAS-if-etag-matches, GET, conditional DELETE. Map ETag→body from logs. Partition by key. **First validate the checker with hand-made good and bad histories** |
| 5. More backends | 10–12 | Cloud: AWS S3 (the reference), R2, Tigris, GCS via HMAC/XML (plus native `x-goog-if-generation-match: 0` as a control). Self-hosted: Ceph RGW (test **versioned buckets**), RustFS, Versity Gateway. Also test delete, copy and multipart conditions |
| 6. Fault injection | 12–14 | Toxiproxy between client and stores (latency, cut connections), `docker kill` nodes in multi-node clusters. For cloud vendors, only client-side faults are possible, and the write-up must say so |
| 7. Matrix + write-up + disclosure | 14–16 | Publish the versioned matrix with repro commands. File GitHub issues for open-source bugs. Use support channels for cloud vendors. Comment on ceph/s3-tests #583 / PR #763. Run CI monthly to keep it current |

**Tech choices:** Go, AWS SDK for Go v2, Porcupine, Docker Compose, Toxiproxy. Optionally export histories in Jepsen EDN format for Knossos later. **Not** Clojure to start, and **not** Elle (overkill for single-key ops).

---

## 6. Known vendor behaviour (Nov 2026, re-verify before publishing)

| Vendor | Behaviour | Category |
|---|---|---|
| AWS S3 | `If-None-Match` since Aug 2024, `If-Match` since Nov 2024. The reference | ✅ |
| Garage | Ignores conditional headers. "Structurally impossible" without consensus (CRDT design). Issue #1052 | 🚫 by design |
| GCS (S3/XML API) | Ignores If-None-Match/If-Match on PUT. Native `x-goog-if-generation-match` works | ⚠️ ignored |
| Cloudflare R2 | Conditional PUT works, but ignores If-Match on DELETE (re-tested 5 Nov 2026). No conditional copy. **Rate limit: 1 write/sec per key → 429s** | ⚠️ partial |
| SeaweedFS | Multiple winners before 4.18, stalls 4.18–4.28, fixed in **4.29**. Issue #12210: some residual races | ✅ (≥4.29) |
| Ceph RGW | Versioned buckets: two `If-None-Match: *` PUTs both succeed (tracker #80925, fix PR #72118 pending) | ❌ bug |
| Tigris | Claims strong consistency. Tested privately with Antithesis. A good "control" | ✅ (claimed) |
| MinIO | Community edition archived in 2026. Historical "two winners." Only worth testing as a baseline | — |

**Why vendors don't "just fix it":** guaranteeing a single winner across many machines needs them all to agree (**consensus**), which costs speed and availability when the network breaks (the CAP trade-off). Also, HTTP lets servers **silently ignore headers they don't understand**, so unsupported rules return "200 OK" instead of an error.

---

## 7. Competitors / prior partial work

| Who | What they did | Gap we fill |
|---|---|---|
| t4db/t4 (PRs #377, #378, #382–384) | Nightly Jepsen against its own DB on S3/GCS. Found that GCS ignores both headers | Tests t4, not the stores. Only 2 vendors |
| pithecene/lode PR #310 | Races writers on several backends. Found SeaweedFS's issues | Pass/fail only. No history, checker, faults or public matrix |
| Ceph (tracker #80925, s3-tests PR #763) | Found and is fixing its own two-winner bug | Ceph-only |
| Dolt PR #11433 | 8-writer race. Calls it "a smoke test, not a proof" | Exactly our gap |
| Antithesis × Tigris (Apr 2026) | Deep fault-injection testing of Tigris | One vendor, private, not reusable |
| Tigris 68-test matrix, MSST-S3 | Feature-presence checklists | No concurrency or histories |
| Lance, meerkat, ClickHouse, s3nd, sluis, etc. | One-off 16-writer probes | Not reusable or rigorous |
| Faultline (patelpratyush) | Go Jepsen-style harness + checker for etcd/NATS | Good code reference, wrong targets |
| jepsen.io | No object-storage analysis ever published (jepsen issue #461 asks for MinIO) | Open space |

---

## 8. Costs and safety rules

- Everything runs **free locally** until Phase 5.
- Cloud later: AWS (free-tier credits; about $0.005 per 1,000 PUTs), R2 (1M writes/month free), Tigris and GCS (small free tiers). Expect **under $5–10/month**.
- **Before any cloud account:** ask me first, set billing alerts, and auto-delete test buckets.

---

## 9. Key links

- Porcupine: https://github.com/anishathalye/porcupine
- Kingsbury, Strong consistency models: https://aphyr.com/posts/313-strong-consistency-models
- Jepsen: https://github.com/jepsen-io/jepsen
- Garage known issues: https://garagehq.deuxfleurs.fr/documentation/reference-manual/known-issues/
- R2 conditional ops: https://developers.cloudflare.com/r2/api/s3/conditional-operations/
- ceph/s3-tests #583: https://github.com/ceph/s3-tests/issues/583
- Dolt PR #11433: https://github.com/dolthub/dolt/pull/11433
- t4 PR #382: https://github.com/t4db/t4/pull/382
- lode PR #310: https://github.com/pithecene/lode/pull/310
- Antithesis Tigris report: https://antithesis.com/blog/2026/tigris_report/
